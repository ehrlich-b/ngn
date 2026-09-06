# Pinned Go opponent provisioning on WSL

No head-to-head strength result is claimed. These are frozen public-release artifacts prepared for subsequent matched-condition testing. Every build and executable ran on WSL; the deployed NGN tree remained unchanged.

| Opponent | Source revision | Selected binary SHA-256 | Protocol state |
|---|---|---|---|
| Counter 5.5 | 63c487ca724c620f71c129d62129c6fb9109c872 | 6c48fb52934d49d3774633e32f0b4fb4796b1e2c63925f24167ef0f0c0d761c8 | UCI/ready/quit pass |
| Zahak 10.0 | 72ff8066e2d8348d8d3847fab62c4a6b8344240d | fdb23a317860bf8ec847d4818c62557a40601c7ee61a176acb764c17104beac0 | UCI/ready/quit pass |
| Blunder 8.5.5 AVX2 | 89230a74a966b3400610271eca49c4be241e94cd | 87e219dca06b24212a762d1ef5c220d761c3ffef99e1ec67e7d2700ea9677a3e | Sequential UCI/ready/quit pass |
| Chess-3 v4.0 PGO | a33531629cbe82eef6810f982ec814e79d52a3b3 | 28335b31fd2e818dc8db6e6188a76744fee8d9223762fe3e59046d22798db3ec | Final PGO UCI/ready/quit pass |

Evidence root: /home/ehrli/repos/ngn-next/output/go-opponent-preflight-20260906. Counter/Zahak are recorded in handshake.json. Blunder's selected archive member and hash are recorded in blunder-binary-identity.json; its successful protocol receipt is blunder-handshake-attempt3/receipt.json. Two earlier batched-input attempts are preserved: the console-to-UCI reader handoff lost pre-buffered commands. Sequential ready barriers solve the handshake without source changes.

Chess-3 was built from the exact release tag using Go 1.25.5, Linux amd64.v3, readonly modules and the release Makefile's prescribed PGO workflow: unprofiled build, built-in 50-position benchmark profile, final PGO build. The bounded profile stage completed successfully with no surviving descendants. Profile SHA-256: 498c324ed40c73a996aebc3d7febe9848792c0653460332bc630bd42fbe34154. Final build and protocol receipts are chess3-pgo-build-v1/receipt.json and chess3-pgo-handshake-v1/receipt.json. Profiling the prescribed workload is provisioning evidence, not a speed or strength comparison.

Counter, Zahak and Chess-3 advertise Threads maximum 1 in these exact artifacts; Blunder advertises no Threads option. Future match profiles must use these actual capabilities. Counter has specific benign version/network startup stderr; Zahak prints a startup banner. The NGN diagnostic runner's exact no-stderr profile therefore cannot be silently reused for external opponents.

Legal-search smoke and full audited matches remain pending. Recheck the public release inventory before a leadership claim; do not substitute a development branch or assume published ratings share these settings.
