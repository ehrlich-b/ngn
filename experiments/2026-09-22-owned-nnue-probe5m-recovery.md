# K4 5M post-label recovery

Status at 2026-09-22 05:30 UTC: the isolated retry pipeline is running. No
training or playing result follows from this recovery note.

The first 50 main-expansion shards finished labeling with `STATE=LABELED`.
All 50 pack receipts exist and contain 4,327,066 accepted expansion records,
above the required four million. The original post-label service then failed
before creating the finalizer binary directory, finalized corpus, or training
directory. Its systemd journal reported:

```text
line 69: ((: accepted >= 4_000_000: value too great for base
```

Bash arithmetic does not accept the digit separators in that literal. The
original panel and lower-rate services stopped only because they observed the
upstream `FAILED` states. The frozen original scripts and their hashes remain
as evidence of the failed attempt.

[Post-label retry 1](2026-09-22-owned-nnue-probe5m-postlabel-retry1.sh), SHA-256
`f5c2e04b43a72d2ce34ddd3ef8c8c7551a4ac3a0f78d051afa60062cc665e2ed`,
uses `4000000` and admits only the completed-label, failed-post-label state
with no finalizer or training output. It copies the old state to
`POSTLABEL_STATE.attempt0-failed`. The resulting preflight receipt binds all
4,327,066 accepted records and the pinned finalizer, bridge and trainer binary
hashes. At 05:30 UTC it was running finalization as user unit
`ngn-k4-probe5m-postlabel-retry1-20260922.service`.

[Panel retry 1](2026-09-22-owned-nnue-probe5m-panel-retry1.sh), SHA-256
`c093ee439067d21a6381427b1c1526e819bc62c252c0f0708933c1c7fbfd7e32`,
archives the previous panel failure state and executes the unchanged pinned
panel script. Unit `ngn-k4-probe5m-panel-retry1-20260922.service` was waiting
for calibration at 05:30 UTC.

[Lower-rate retry 1](2026-09-22-owned-nnue-probe5m-lr1-retry1.sh), SHA-256
`efd9f8e4e22f1bf743ac45e5ea650ac362fb60521e0e24582898e26792f84a50`,
archives the previous lower-rate run, which contained only its `FAILED` state,
and executes the unchanged pinned launcher. Unit
`ngn-k4-probe5m-lr1-retry1-20260922.service` was waiting for the original 5M
panel at 05:30 UTC. These three retry stages are event-driven. The 20M
expansion remains held pending quality and game evidence.
