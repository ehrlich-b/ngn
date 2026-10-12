#!/usr/bin/env bash
set -euo pipefail
s=/home/ehrli/ngn-data/night-20261007/session74-clock-panel
cd "$s"
finish() { local rc=$?; echo "$rc" > panel-checks.exit; date -Is > panel-checks.done; }
trap finish EXIT
python3 - "$s" <<'PY'
import ctypes,os,select,sys,time
from pathlib import Path
p=Path(sys.argv[1]);lib=ctypes.CDLL(None);fd=lib.inotify_init1(os.O_CLOEXEC|os.O_NONBLOCK)
assert fd>=0 and lib.inotify_add_watch(fd,os.fsencode(p),0x8|0x80)>=0
end=time.monotonic()+3600
while not (p/'checks.done').exists():
 remaining=end-time.monotonic();assert remaining>0 and select.select([fd],[],[],remaining)[0]
 os.read(fd,65536)
os.close(fd)
assert (p/'checks.exit').read_text().strip()=='0'
PY
export PATH=/home/ehrli/go/bin:/usr/local/go/bin:$PATH GOAMD64=v3 GOMAXPROCS=2
export GOCACHE=/home/ehrli/ngn-data/night-20261007/session17f/go-cache
for variant in base horizon; do
 cp panel_test.go "$variant/engine/campaign_panel_test.go"
 cp panel_see_test.go "$variant/engine/campaign_panel_see_test.go"
 cd "$s/$variant"
 gofmt -w engine/campaign_panel_test.go engine/campaign_panel_see_test.go
 NGN_PANEL_FILE="$s/panel.json" go test -p 2 -short ./engine -run '^TestCampaignPanelControls$' -count=1 -v > "$s/$variant-panel-controls.log" 2>&1
 NGN_PANEL_FILE="$s/panel.json" NGN_PANEL_NET="$s/../session72-see-rig/base.nnue" NGN_PANEL_VARIANT="$variant" NGN_PANEL_SMOKE=1 go test -p 2 -short ./engine -run '^TestCampaignPanelDiagnostic$' -count=1 -v > "$s/$variant-panel-smoke.log" 2>&1
 go test -p 2 -pgo=default.pgo -c -o "$s/$variant-panel.test" ./engine > "$s/$variant-panel-build.log" 2>&1
 cd "$s"
done
sha256sum base-panel.test horizon-panel.test panel.json panel_test.go panel_see_test.go > panel-inputs.sha256
