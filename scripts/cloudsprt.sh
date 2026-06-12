#!/usr/bin/env bash
# cloudsprt.sh — disposable AWS spot workers for SPRT runs (2026-06-11).
#
# Fire-and-forget Tier-1/Tier-2 gates on ~$0.10/hr Graviton spot instances so
# the Mac carries zero load. Every worker has THREE shutdown paths:
#   1. hard TTL at boot (user-data `shutdown -h +N`) — fires even if the run wedges;
#      one-time spot instances TERMINATE on OS shutdown, root EBS is delete-on-term
#   2. self-shutdown on run completion (after the final S3 log upload)
#   3. `cloudsprt.sh kill` sweep + `ps` audit (tag app=cloudsprt)
# Worst-case leak = the TTL cap (default 6h x spot price ~= $0.75).
#
# Logs upload to S3 every 60s + at exit, so results survive spot eviction AND
# the Mac sleeping. Workers need no AWS credentials (instance profile, PutObject
# only). Engines run with -lowpower=false (taskpolicy is macOS-only).
#
#   setup   one-time, idempotent: keypair, security group, S3 bucket (30d expiry), IAM
#   smoke   ~$0.001 end-to-end proof: nano spot up -> S3 write -> SELF-shutdown observed
#   launch  cross-compile worktree (cand) + HEAD/-baseref (base) for linux/arm64,
#           shard the openings, start N workers; default args = Tier-1 gate
#   status  live instances + latest result line per shard (via S3, lags <=1 min)
#   fetch   pool shard logs -> elo/CI/LLR; verdict+sweep once NO live workers remain
#   ps      ALL cloudsprt instances in any state (audit — empty when idle)
#   kill    terminate instances (all, or -runid X)
#   prices  current spot $/hr for the candidate Graviton types
#
# Examples:
#   scripts/cloudsprt.sh launch                                   # Tier-1, 1 worker
#   scripts/cloudsprt.sh launch -shards 3 -- -tc 10+0.1 -elo0 -5 -elo1 0 -mingames 64 -maxgames 3000
#   scripts/cloudsprt.sh fetch -runid 260611-120000
#
# Pool shards only with shards (same instance type) — never mix cloud and Mac
# games in one verdict. Graviton types only (binaries are linux/arm64).

set -euo pipefail

APP=cloudsprt
die() { echo "cloudsprt: $*" >&2; exit 1; }
command -v aws >/dev/null 2>&1 || die "aws CLI not installed (brew install awscli)"

REGION=${AWS_REGION:-$(aws configure get region 2>/dev/null || true)}
[ -n "$REGION" ] || REGION=us-east-1
A() { aws --region "$REGION" --output text "$@"; }
PEM=$HOME/.ssh/$APP.pem
SSHOPTS=(-i "$PEM" -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR -o ConnectTimeout=5)
REPO=$(cd "$(dirname "$0")/.." && pwd)

acct()   { A sts get-caller-identity --query Account; }
bucket() { echo "$APP-$(acct)-$REGION"; }
ami()    { A ssm get-parameter --name /aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-arm64 --query Parameter.Value; }
sgid()   { A ec2 describe-security-groups --filters Name=group-name,Values=$APP --query 'SecurityGroups[0].GroupId' 2>/dev/null || echo None; }

cmd_setup() {
    local acctid
    acctid=$(acct) || die "no AWS credentials — run: aws configure"
    echo "account $acctid, region $REGION"

    if ! A ec2 describe-key-pairs --key-names $APP >/dev/null 2>&1; then
        A ec2 create-key-pair --key-name $APP --query KeyMaterial > "$PEM"
        chmod 400 "$PEM"; echo "keypair: created ($PEM)"
    elif [ ! -f "$PEM" ]; then
        A ec2 delete-key-pair --key-name $APP
        A ec2 create-key-pair --key-name $APP --query KeyMaterial > "$PEM"
        chmod 400 "$PEM"; echo "keypair: recreated (local pem was missing)"
    else
        echo "keypair: ok"
    fi

    local vpc sg
    vpc=$(A ec2 describe-vpcs --filters Name=is-default,Values=true --query 'Vpcs[0].VpcId')
    [ "$vpc" != "None" ] || die "no default VPC in $REGION"
    sg=$(sgid)
    if [ "$sg" = "None" ]; then
        sg=$(A ec2 create-security-group --group-name $APP --description "$APP disposable spot workers" --vpc-id "$vpc" --query GroupId)
        A ec2 authorize-security-group-ingress --group-id "$sg" --protocol tcp --port 22 --cidr 0.0.0.0/0 >/dev/null
        echo "security group: created ($sg, ssh key-only)"
    else
        echo "security group: ok ($sg)"
    fi

    local b; b=$(bucket)
    if ! aws --region "$REGION" s3api head-bucket --bucket "$b" >/dev/null 2>&1; then
        if [ "$REGION" = "us-east-1" ]; then
            A s3api create-bucket --bucket "$b" >/dev/null
        else
            A s3api create-bucket --bucket "$b" --create-bucket-configuration LocationConstraint="$REGION" >/dev/null
        fi
        A s3api put-bucket-lifecycle-configuration --bucket "$b" --lifecycle-configuration \
            '{"Rules":[{"ID":"expire","Status":"Enabled","Filter":{},"Expiration":{"Days":30}}]}'
        echo "bucket: created (s3://$b, 30-day expiry)"
    else
        echo "bucket: ok (s3://$b)"
    fi

    # Worker role/profile are normally pre-created by scripts/cloudsprt_bootstrap.sh
    # (run as root in CloudShell) so the operator key here needs NO iam-write perms.
    if aws iam get-role --role-name $APP-role >/dev/null 2>&1; then
        echo "iam role: ok"
    elif aws iam create-role --role-name $APP-role --assume-role-policy-document \
            '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ec2.amazonaws.com"},"Action":"sts:AssumeRole"}]}' >/dev/null 2>&1; then
        aws iam put-role-policy --role-name $APP-role --policy-name s3put --policy-document \
            "{\"Version\":\"2012-10-17\",\"Statement\":[{\"Effect\":\"Allow\",\"Action\":\"s3:PutObject\",\"Resource\":\"arn:aws:s3:::$APP-*/*\"}]}"
        echo "iam role: created"
    else
        die "iam role missing and this key can't create it — run scripts/cloudsprt_bootstrap.sh in CloudShell first"
    fi
    if aws iam get-instance-profile --instance-profile-name $APP-profile >/dev/null 2>&1; then
        echo "instance profile: ok"
    elif aws iam create-instance-profile --instance-profile-name $APP-profile >/dev/null 2>&1; then
        aws iam add-role-to-instance-profile --instance-profile-name $APP-profile --role-name $APP-role
        echo "instance profile: created (allow ~10s to propagate)"
    else
        die "instance profile missing and this key can't create it — run scripts/cloudsprt_bootstrap.sh in CloudShell first"
    fi
    echo "setup complete — next: scripts/cloudsprt.sh smoke"
}

# launch_one TYPE TTL_MINUTES RUNID SHARD -> instance id (uses AMI_ID, SG_ID globals)
# Iterates the region's default subnets (one per AZ) on spot-capacity misses.
launch_one() {
    local ud id try subnets sn snarg arr n rot rotated k
    subnets=$(A ec2 describe-subnets --filters Name=default-for-az,Values=true --query 'Subnets[].SubnetId')
    [ -n "$subnets" ] || subnets=DEFAULT
    # Spread shards across AZs: rotate the AZ (subnet) preference list by shard number
    # so a multi-shard run prefers DIFFERENT AZs (independent shards = zero inter-shard
    # cost; de-correlates spot reclaims, which hit by AZ+type — observed both 4xlarge
    # taken together in us-east-2a). Each shard still falls back to the other AZs on a
    # capacity miss; single-shard (shard 0) rotation is a no-op.
    if [ "$subnets" != DEFAULT ]; then
        arr=($subnets); n=${#arr[@]}; rot=$(( ${4:-0} % n )); rotated=""
        for ((k = 0; k < n; k++)); do rotated="$rotated ${arr[$(((k + rot) % n))]}"; done
        subnets=$rotated
    fi
    ud=$(mktemp)
    printf '#!/bin/bash\nshutdown -h +%s\n' "$2" > "$ud"
    for try in 1 2 3 4 5 6; do
        for sn in $subnets; do
            snarg=""
            [ "$sn" != DEFAULT ] && snarg="--subnet-id $sn"
            # shellcheck disable=SC2086
            if id=$(A ec2 run-instances --image-id "$AMI_ID" --instance-type "$1" --key-name $APP \
                    --security-group-ids "$SG_ID" --iam-instance-profile Name=$APP-profile $snarg \
                    --instance-market-options 'MarketType=spot,SpotOptions={SpotInstanceType=one-time,InstanceInterruptionBehavior=terminate}' \
                    --user-data "file://$ud" \
                    --tag-specifications "ResourceType=instance,Tags=[{Key=app,Value=$APP},{Key=runid,Value=$3},{Key=shard,Value=$4},{Key=Name,Value=$APP-$3-s$4}]" \
                    --query 'Instances[0].InstanceId' 2>/tmp/cloudsprt_err.$$); then
                rm -f "$ud"; echo "$id"; return 0
            fi
            grep -qi 'capacity' /tmp/cloudsprt_err.$$ && continue          # try next AZ
            if grep -qi 'instance profile' /tmp/cloudsprt_err.$$; then     # IAM propagation
                sleep 5; continue 2
            fi
            break 2                                                        # real error
        done
        break  # every AZ lacked capacity — caller decides (other type/region)
    done
    cat /tmp/cloudsprt_err.$$ >&2; rm -f "$ud" /tmp/cloudsprt_err.$$
    return 1
}

# sweep_runid RUNID — terminate every worker of a run (idempotent, never fails)
sweep_runid() {
    local ids
    ids=$(A ec2 describe-instances --filters Name=tag:app,Values=$APP "Name=tag:runid,Values=$1" \
        Name=instance-state-name,Values=pending,running --query 'Reservations[].Instances[].InstanceId' 2>/dev/null || true)
    # shellcheck disable=SC2086
    [ -n "$ids" ] && A ec2 terminate-instances --instance-ids $ids >/dev/null 2>&1 || true
}

# wait_ssh INSTANCE_ID -> public ip once sshable (fails fast if the instance dies first)
wait_ssh() {
    local ip="" st n=0
    while :; do
        st=$(A ec2 describe-instances --instance-ids "$1" --query 'Reservations[0].Instances[0].State.Name' 2>/dev/null || echo unknown)
        case $st in shutting-down|terminated) return 1;; esac
        ip=$(A ec2 describe-instances --instance-ids "$1" --query 'Reservations[0].Instances[0].PublicIpAddress' 2>/dev/null || true)
        [ -n "$ip" ] && [ "$ip" != "None" ] && break
        n=$((n+1)); [ $n -gt 30 ] && return 1
        sleep 5
    done
    n=0
    until ssh "${SSHOPTS[@]}" "ec2-user@$ip" true 2>/dev/null; do
        n=$((n+1)); [ $n -gt 30 ] && return 1
        sleep 5
    done
    echo "$ip"
}

cmd_smoke() {
    local b runid id ip state t
    b=$(bucket); AMI_ID=$(ami); SG_ID=$(sgid)
    [ "$SG_ID" != "None" ] || die "run setup first"
    runid=smoke-$(date +%H%M%S)
    local ty
    id=""
    for ty in t4g.nano t4g.micro c7g.xlarge c8g.xlarge; do
        echo "launching $ty spot (TTL 20 min)..."
        if id=$(launch_one "$ty" 20 "$runid" 0); then break; fi
    done
    [ -n "$id" ] || die "no spot capacity on any smoke type in $REGION (or quota — Service Quotas -> 'All Standard Spot Instance Requests')"
    echo "instance $id — waiting for ssh (~1 min)..."
    ip=$(wait_ssh "$id") || { A ec2 terminate-instances --instance-ids "$id" >/dev/null; die "ssh never came up (instance terminated)"; }
    echo "up at $ip — S3 marker write + self-shutdown..."
    ssh "${SSHOPTS[@]}" "ec2-user@$ip" \
        "sudo shutdown -h +20 >/dev/null 2>&1; for i in 1 2 3 4 5 6 7 8 9 10; do echo SMOKE_OK | aws s3 cp - s3://$b/$runid/ok.txt >/dev/null 2>&1 && break; sleep 3; done; sudo shutdown -h now" || true
    t=0
    until aws --region "$REGION" s3api head-object --bucket "$b" --key "$runid/ok.txt" >/dev/null 2>&1; do
        t=$((t+1)); [ $t -gt 24 ] && { A ec2 terminate-instances --instance-ids "$id" >/dev/null; die "S3 marker never appeared (instance terminated) — instance-profile problem?"; }
        sleep 5
    done
    echo "S3 upload: OK (instance-profile creds work)"
    t=0
    while :; do
        state=$(A ec2 describe-instances --instance-ids "$id" --query 'Reservations[0].Instances[0].State.Name')
        case $state in shutting-down|terminated) break;; esac
        t=$((t+1)); [ $t -gt 36 ] && { A ec2 terminate-instances --instance-ids "$id" >/dev/null; die "instance did NOT self-shutdown (force-terminated) — do not trust launch until investigated"; }
        sleep 5
    done
    echo "self-shutdown -> $state: OK — the shutdown guarantee holds"
    A ec2 terminate-instances --instance-ids "$id" >/dev/null 2>&1 || true
    echo "SMOKE PASS (cost ~\$0.001)"
}

cmd_launch() {
    local shards=1 type=c8g.2xlarge ttl=6 conc=0 baseref=HEAD openings=$REPO/output/sprt_openings.txt nullrun=0
    while [ $# -gt 0 ]; do
        case $1 in
            -shards)   shards=$2; shift 2;;
            -type)     type=$2; shift 2;;
            -ttl)      ttl=$2; shift 2;;
            -conc)     conc=$2; shift 2;;
            -baseref)  baseref=$2; shift 2;;
            -openings) openings=$2; shift 2;;
            -null)     nullrun=1; shift;;
            --) shift; break;;
            *) die "unknown launch flag $1 (sprt args go after --)";;
        esac
    done
    local sprtargs=("$@")
    [ ${#sprtargs[@]} -gt 0 ] || sprtargs=(-tc 10+0.1 -elo0 -5 -elo1 0 -mingames 64 -maxgames 600)
    [ -f "$openings" ] || die "openings file not found: $openings"
    case $type in
        c8g.*|c7g.*|m8g.*|m7g.*|r8g.*|r7g.*|t4g.*) ;;
        *) die "$type is not a Graviton type (binaries are linux/arm64)";;
    esac
    local b; b=$(bucket); AMI_ID=$(ami); SG_ID=$(sgid)
    [ "$SG_ID" != "None" ] || die "run setup first"
    local runid work
    runid=$(date +%y%m%d-%H%M%S)
    work=$(mktemp -d)
    # bake values in now (locals are gone when the EXIT trap fires)
    trap "git -C '$REPO' worktree remove --force '$work/basesrc' >/dev/null 2>&1; git -C '$REPO' worktree prune >/dev/null 2>&1; true" EXIT

    if [ "$nullrun" = 1 ]; then
        echo "building linux/arm64 NULL run (both engines = $baseref) + sprt (worktree)..."
        (cd "$REPO" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o "$work/sprt" ./cmd/sprt)
        git -C "$REPO" worktree add --detach "$work/basesrc" "$baseref" >/dev/null
        (cd "$work/basesrc" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o "$work/ngn_base" .)
        git -C "$REPO" worktree remove --force "$work/basesrc"
        cp "$work/ngn_base" "$work/ngn_cand"
    else
        echo "building linux/arm64: cand (worktree) + base ($baseref) + sprt..."
        (cd "$REPO" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o "$work/ngn_cand" . \
                    && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o "$work/sprt" ./cmd/sprt)
        git -C "$REPO" worktree add --detach "$work/basesrc" "$baseref" >/dev/null
        (cd "$work/basesrc" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o "$work/ngn_base" .)
        git -C "$REPO" worktree remove --force "$work/basesrc"
    fi

    # Launch all instances first. Shard 0 resolves the type through a capacity
    # fallback chain; remaining shards reuse it (shard pooling must be homogeneous).
    # A shard lost to capacity/reclaim is dropped INDIVIDUALLY (terminated if it
    # exists); the run continues with the survivors. Zero survivors => sweep + die.
    local i id ip ty chosen="" ids=() shardnos=() ttlmin=$((ttl * 60))
    for ((i = 0; i < shards; i++)); do
        if [ -z "$chosen" ]; then
            for ty in $type c7g.2xlarge c8g.4xlarge c7g.4xlarge; do
                if id=$(launch_one "$ty" "$ttlmin" "$runid" "$i"); then chosen=$ty; break; fi
                echo "no spot capacity: $ty — trying next type"
            done
            [ -n "$chosen" ] || { sweep_runid "$runid"; die "no spot capacity on any allowed type in $REGION right now"; }
        else
            if ! id=$(launch_one "$chosen" "$ttlmin" "$runid" "$i"); then
                echo "shard $i: no spot capacity — dropped, continuing with the rest"
                continue
            fi
        fi
        ids+=("$id"); shardnos+=("$i")
        echo "shard $i: $id ($chosen)"
    done
    if [ "$conc" = 0 ]; then
        case $chosen in
            *.4xlarge) conc=15;;
            *.2xlarge) conc=7;;
            *.xlarge)  conc=3;;
            *)         conc=1;;
        esac
    fi
    local j live=0
    for ((j = 0; j < ${#ids[@]}; j++)); do
        i=${shardnos[$j]}; id=${ids[$j]}
        awk "NR % $shards == $i" "$openings" > "$work/open$i.txt"
        cat > "$work/run$i.sh" <<EOF
#!/bin/bash
cd /home/ec2-user
sudo shutdown -h +$ttlmin >/dev/null 2>&1   # second TTL path, independent of user-data
(while sleep 60; do aws s3 cp sprt.log s3://$b/$runid/shard$i.log >/dev/null 2>&1; done) &
KEEPER=\$!
./sprt -new ./ngn_cand -base ./ngn_base -lowpower=false -concurrency $conc -openings open.txt ${sprtargs[*]} > sprt.log 2>&1
kill \$KEEPER 2>/dev/null
echo CLOUDSPRT_DONE >> sprt.log
aws s3 cp sprt.log s3://$b/$runid/shard$i.log
sudo shutdown -h now
EOF
        echo "shard $i: waiting for ssh..."
        if ! ip=$(wait_ssh "$id"); then
            A ec2 terminate-instances --instance-ids "$id" >/dev/null 2>&1 || true
            echo "shard $i: gone before provisioning (spot reclaim) — terminated, dropped"
            continue
        fi
        # Push files with RETRY + VERIFY. Boot-window ssh/scp flakiness was silently
        # dropping the openings file on the LATER shards (they came up with no open.txt,
        # so sprt sat idle producing 0 games — a 4-shard run degenerated to single).
        # Retry the scp's up to 4x and VERIFY open.txt landed with the right line count
        # before starting — a starved shard is worse than a cleanly-dropped one.
        nopen=$(wc -l < "$work/open$i.txt" | tr -d ' ')
        pushed=0
        for _ in 1 2 3 4; do
            if scp "${SSHOPTS[@]}" "$work/sprt" "$work/ngn_cand" "$work/ngn_base" "ec2-user@$ip:" >/dev/null 2>&1 \
               && scp "${SSHOPTS[@]}" "$work/open$i.txt" "ec2-user@$ip:open.txt" >/dev/null 2>&1 \
               && scp "${SSHOPTS[@]}" "$work/run$i.sh" "ec2-user@$ip:run.sh" >/dev/null 2>&1 \
               && [ "$(ssh "${SSHOPTS[@]}" "ec2-user@$ip" 'wc -l < open.txt' 2>/dev/null | tr -d ' ')" = "$nopen" ]; then
                pushed=1; break
            fi
            sleep 5
        done
        if [ "$pushed" = 0 ]; then
            A ec2 terminate-instances --instance-ids "$id" >/dev/null 2>&1 || true
            echo "shard $i: openings/binaries push failed after retries — terminated, dropped"
            continue
        fi
        if ! ssh "${SSHOPTS[@]}" "ec2-user@$ip" 'chmod +x sprt ngn_cand ngn_base run.sh && nohup ./run.sh >/dev/null 2>&1 & sleep 1'; then
            A ec2 terminate-instances --instance-ids "$id" >/dev/null 2>&1 || true
            echo "shard $i: run.sh start failed — terminated, dropped"
            continue
        fi
        # Liveness probe — LOG ONLY, never terminate. A fresh inbound ssh to a small box
        # already RUNNING sprt at full concurrency can't always be serviced promptly
        # (sshd starved of CPU), so a failed probe was FALSE-KILLING productive workers
        # (terminated boxes had already played ~100 games). The no-leak guarantee does
        # NOT depend on this terminate: a genuinely dead sprt uploads no S3 log and is
        # reaped by the two boot/run TTLs; the in-sprt watchdog handles wedged games.
        # So: best-effort probe, log if unconfirmed, and PROCEED.
        alive=0
        for _ in 1 2 3; do
            sleep 3
            if ssh "${SSHOPTS[@]}" "ec2-user@$ip" 'pgrep -f "\./sprt" >/dev/null' 2>/dev/null; then alive=1; break; fi
        done
        [ "$alive" = 1 ] || echo "shard $i: liveness unconfirmed (busy box / slow sshd) — proceeding; S3 progress + TTL are the real backstops"
        live=$((live+1))
        echo "shard $i: running on $ip"
    done
    [ "$live" -gt 0 ] || { sweep_runid "$runid"; die "zero shards survived provisioning — ALL $runid workers terminated"; }
    find "$work" -mindepth 1 -delete && rmdir "$work"
    echo
    echo "runid $runid — $live/$shards shard(s) x $chosen conc-$conc, TTL ${ttl}h, sprt args: ${sprtargs[*]}"
    echo "  watch:  scripts/cloudsprt.sh status"
    echo "  result: scripts/cloudsprt.sh fetch -runid $runid"
}

cmd_status() {
    local b r k
    b=$(bucket)
    aws --region "$REGION" ec2 describe-instances \
        --filters Name=tag:app,Values=$APP Name=instance-state-name,Values=pending,running \
        --query 'Reservations[].Instances[].[InstanceId,InstanceType,Tags[?Key==`runid`]|[0].Value,PublicIpAddress,LaunchTime]' \
        --output table
    for r in $(A ec2 describe-instances \
            --filters Name=tag:app,Values=$APP Name=instance-state-name,Values=pending,running \
            --query 'Reservations[].Instances[].[Tags[?Key==`runid`]|[0].Value]' | sort -u); do
        echo "runid $r (S3 lags <=1 min):"
        for k in $(A s3api list-objects-v2 --bucket "$b" --prefix "$r/" --query 'Contents[].Key' 2>/dev/null); do
            [ "$k" = "None" ] && continue
            aws --region "$REGION" s3 cp "s3://$b/$k" "/tmp/cloudsprt_tail.$$" >/dev/null 2>&1 \
                && echo "  $k: $(grep -E '^G[0-9]+' "/tmp/cloudsprt_tail.$$" | tail -1)"
        done
    done
    rm -f "/tmp/cloudsprt_tail.$$"
}

cmd_fetch() {
    local runid="" elo0=-5 elo1=0
    while [ $# -gt 0 ]; do
        case $1 in
            -runid) runid=$2; shift 2;;
            -elo0)  elo0=$2; shift 2;;
            -elo1)  elo1=$2; shift 2;;
            *) die "unknown fetch flag $1";;
        esac
    done
    [ -n "$runid" ] || die "usage: fetch -runid <id> [-elo0 -5] [-elo1 0]"
    local b dir live nlive=0
    b=$(bucket); dir=$REPO/output/cloud/$runid
    mkdir -p "$dir"
    aws --region "$REGION" s3 sync "s3://$b/$runid/" "$dir/" >/dev/null
    # Verdict gate = ZERO LIVE WORKERS, not all-shards-DONE: an evicted shard never
    # writes CLOUDSPRT_DONE, but its last 60s upload is pooled as partial-final.
    live=$(A ec2 describe-instances --filters Name=tag:app,Values=$APP "Name=tag:runid,Values=$runid" \
        Name=instance-state-name,Values=pending,running --query 'Reservations[].Instances[].InstanceId' 2>/dev/null || true)
    if [ -n "$live" ]; then nlive=$(echo "$live" | wc -w | tr -d ' '); fi
    local rc=0
    python3 - "$dir" "$elo0" "$elo1" "$nlive" <<'PY' || rc=$?
import sys, os, re, math, glob
d, e0, e1, nlive = sys.argv[1], float(sys.argv[2]), float(sys.argv[3]), int(sys.argv[4])
W = D = L = 0; done = logs = 0
for f in sorted(glob.glob(os.path.join(d, "shard*.log"))):
    logs += 1
    txt = open(f, errors="replace").read()
    m = None
    for m in re.finditer(r"^G\d+\s+(\d+)W\s+(\d+)D\s+(\d+)L", txt, re.M):
        pass
    fin = "CLOUDSPRT_DONE" in txt
    done += fin
    if m:
        W += int(m.group(1)); D += int(m.group(2)); L += int(m.group(3))
    state = "done" if fin else ("RUNNING" if nlive else "EVICTED->partial-final")
    tail = m.group(0) if m else (txt.strip().splitlines()[-1] if txt.strip() else "(empty)")
    print(f'{os.path.basename(f)}: {state}  {tail}')
if logs == 0 or W + D + L == 0:
    if nlive:
        print(f"no pooled games yet — {nlive} live worker(s), uploads land every 60s")
        sys.exit(3)
    sys.exit("no games and no live workers — run never produced results")
n = W + D + L
s = (W + D / 2) / n
var = (W * (1 - s) ** 2 + D * (0.5 - s) ** 2 + L * s ** 2) / n
se = math.sqrt(var / n) if var > 0 else 0.0
def elo(x):
    x = min(max(x, 1e-9), 1 - 1e-9)
    return -400 * math.log10(1 / x - 1)
def lo(e):
    return 1 / (1 + 10 ** (-e / 400))
s0, s1 = lo(e0), lo(e1)
llr = 0.0 if var == 0 else (s1 - s0) * (2 * s - s0 - s1) / (2 * var) * n
print(f"\npooled: {n}g  {W}W {D}D {L}L  score {100*s:.1f}%  "
      f"elo {elo(s):+.1f} [{elo(s-1.96*se):+.1f},{elo(s+1.96*se):+.1f}]  "
      f"LLR {llr:+.2f}  (elo0 {e0} elo1 {e1}, bounds +/-2.94)")
if nlive:
    print(f"IN PROGRESS: {nlive} live worker(s) — NOT a verdict yet")
    sys.exit(3)
if done < logs:
    print(f"note: {logs-done} shard(s) evicted — pooled at last upload (complete games only; verdict is FINAL)")
PY
    if [ "$rc" = 0 ]; then
        sweep_runid "$runid"
        echo "(no live workers — verdict is final; sweep was an idempotent no-op if already self-terminated)"
    elif [ "$rc" = 3 ]; then
        echo "(run still in progress — workers stay up; TTL still bounds them)"
    else
        return "$rc"
    fi
}

cmd_ps() {
    aws --region "$REGION" ec2 describe-instances --filters Name=tag:app,Values=$APP \
        --query 'Reservations[].Instances[].[InstanceId,State.Name,InstanceType,Tags[?Key==`runid`]|[0].Value,LaunchTime]' \
        --output table
}

cmd_kill() {
    local ids filters=("Name=tag:app,Values=$APP" "Name=instance-state-name,Values=pending,running")
    if [ "${1:-}" = "-runid" ]; then
        [ -n "${2:-}" ] || die "kill -runid <id>"
        filters+=("Name=tag:runid,Values=$2")
    fi
    ids=$(A ec2 describe-instances --filters "${filters[@]}" --query 'Reservations[].Instances[].InstanceId')
    [ -n "$ids" ] || { echo "nothing running"; return 0; }
    # shellcheck disable=SC2086
    A ec2 terminate-instances --instance-ids $ids --query 'TerminatingInstances[].[InstanceId,CurrentState.Name]'
}

cmd_prices() {
    aws --region "$REGION" ec2 describe-spot-price-history \
        --instance-types c8g.2xlarge c8g.4xlarge c7g.2xlarge c7g.4xlarge t4g.nano \
        --product-descriptions "Linux/UNIX" --start-time "$(date -u +%Y-%m-%dT%H:%M:%S)" \
        --query 'sort_by(SpotPriceHistory,&InstanceType)[].[InstanceType,AvailabilityZone,SpotPrice]' \
        --output table
}

case "${1:-help}" in
    setup)  shift; cmd_setup "$@";;
    smoke)  shift; cmd_smoke "$@";;
    launch) shift; cmd_launch "$@";;
    status) shift; cmd_status "$@";;
    fetch)  shift; cmd_fetch "$@";;
    ps)     shift; cmd_ps "$@";;
    kill)   shift; cmd_kill "$@";;
    prices) shift; cmd_prices "$@";;
    *) sed -n '2,36p' "$0" | sed 's/^# \{0,1\}//'; exit 1;;
esac
