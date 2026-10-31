# Existing key-order patch after SEE — session32 prepared gate

This controller may be used only
after independently reviewing the complete SEE400 result and integrating
the exact tested SEE source as a provisional provenance keep. The original
key-order A/A40 failure on39eb128 remains terminal, with zero candidate
games. Do not repeat that experiment if SEE is not kept.

One change on the genuinely changed immediate base: apply the existing
[key-order patch](2026-10-09-key-order-gate.patch), unchanged seed/draw
order. The [controller](2026-10-09-key-after-see.sh) reruns full short/race
and vet on13,15 because the base changed, builds the candidate with the
same PGO profile, and reuses SEE's exact accepted binary as baseline.
No other behavior/net change. C:>=27GiB admission leaves2GiB above the
mandatory25GiB floor. SPSA0–11 and architecture12,14 retain their lanes.

Freeze `session32-key/nomination.json` with `see_reviewed:true`,
`see_verdict:provisional-provenance-keep`, actual `base_source`,
`candidate_change:existing-key-order-only`, topic, baseline binary/net
hashes and unchanged patch hash. Archive that base, copy the exact SEE
baseline, unchanged sprt/parser/openings/net, controller and patch; write
`frozen-inputs.sha256` before detached setsid/nohup launch. The controller
writes the full binary/source/Go/OS/profile/argv manifest before games.

Same predeclared instrument:10+0.1,concurrency2,taskset13,15,nice19,
GOMAXPROCS1/GOAMD64v3,Hash64/Threads1/OwnBook=false. Require a fresh
complete40-game/20-pair A/A with printed and recomputed paired95% interval
containing zero, zero faults. Then exactly400 games/200 pairs. Keep
provisionally only if printed and recomputed upper95>0, zero flags,
crashes, illegal/no-move results, complete raw counts and intact inputs.
Otherwise shelve; no retry, extension or automatic adoption. Combined
provenance SPRT[0,+6] against5b882e2/be34532c remains required.

Terminal `controller.{exit,done}`, `aa-summary.json`, `gate-summary.json`,
raw logs and input checks trigger review. Expected checks4min,A/A10–15min,
gate1.5–2.5h; retain PID and leave it detached across sessions.

Session32 reviewed SEE400, integrated exact tested source as **f8686be**,
and launched **22682 at18:41:28 EDT**. Isolated topic
`night/key-after-see` contains only the unchanged key patch on that base.
Actual nomination and game manifest appear in `session32-key/`; checks
then A/A40/gate400 proceed detached. No automatic adoption.
