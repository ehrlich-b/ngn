# September correctness release deployment

Accepted source commit: **`71399c063ee9688f499e71e1253ebaf81b211813`**.
The committed runtime diff matches the exact tested patch SHA-256 in the
[validation record](2026-09-04-fast-correctness.md). The short fixed-cap check
passed at +23.5 [+1,+46] over 400 games, with zero errors/flags. The separate
absolute gauntlet has now completed at **2726 [2668,2784]**, clearing the
original 2600 gate. [Full result and limitations](2026-09-04-strength-check.md).
Subsequent offline tuning repairs do not change these frozen playing artifacts.

## Installed artifacts

| Platform | SHA-256 |
|---|---|
| Linux amd64/v3 | `5f3b6078ba62afb86cc4f95ead318f64b5adf04c2fed8c2680fdc63c3ee423e0` |
| Windows amd64/v3 | `7b5af06a4a0ff02a0d9c51aa5eb041dd8ddf9935bbe4691d65dd76146a515828` |
| Mac arm64 | `d8f99879b13d9a9a512865f0f918a422f6e505cab29cb35e9b2b0871f64d7b8b` |

All are the frozen Go1.26.2 artifacts built before the source commit. Go VCS
metadata therefore names `b378c43+dirty`; verify this release using the hashes
and source-patch mapping, not that metadata alone.

- WSL source: `/home/ehrli/repos/ngn`, fast-forwarded from `7587eb6` to `71399c0`.
  Default binaries: `build/ngn` (Linux), `build/ngn.exe` (Windows).
- Windows default engine: `C:\Users\ehrli\ngn\ngn.exe`; also installed as
  `C:\Users\ehrli\ngn\repin\ngn.exe`.
- Corrected SPRT (`63ef86d9b986fdb2c260782149af259e278845d89595d010df3970e6698cc9ed`)
  installed at `C:\Users\ehrli\ngn\sprt.exe` and `ngn\sprt\sprt.exe`.
- Corrected gauntlet (`5d1b6ca586420c04bffed729f5e48c775105c2fb04b2c267a15bd6b6a6769b9a`)
  installed at `C:\Users\ehrli\ngn\repin\gauntlet.exe`.
- Mac defaults: `/Users/ehrlich/repos/ngn/build/ngn` and `build/ngn.exe`.

The WSL checkout and binaries had been from June 7. The root Windows engine
and SPRT were from June 28. The root `ngn_crashes.log` contained initialization
messages only, not an actual crash record.

## Rollback

WSL retains branch `backup/pre-repair-20260904` and binary copies
`build/ngn.pre-repair-20260904` / `build/ngn.exe.pre-repair-20260904`.
Each replaced Windows executable has a sibling `<filename>.pre-repair-20260904`
when a prior file existed. Mac binaries have the same backup suffix.
The deployment refused a dirty WSL checkout and required a fast-forward.

## Verification

Before activation, WSL Go1.25.5 ran `go test -short ./... -count=1` and
`go test -short -race ./... -count=1`: both passed. The race suite's engine
package took 24.077s; the shared UCI package took 1.016s.

UCI tests against the installed Linux, Windows and Mac defaults passed:

- Sole black en-passant evasion returns `bestmove e4d3`.
- Mate on the fiftieth move returns `bestmove a7a8`, score mate 1.
- Engines answer the handshake and exit cleanly on `quit`.

The WSL install reported `DEPLOYMENT_VERIFIED 71399c063ee9688f499e71e1253ebaf81b211813`;
the Windows default reported `WINDOWS_DEFAULT_ENGINE_VERIFIED`.
Native match workers had exited before deployment. All build/test/smoke work
finished before launching the subsequent absolute rating games.

Logs and deployment scripts: `output/recovery-2026-09-04/`:
`wsl-deploy.txt`, `windows-deploy.txt`, `windows-smoke.txt`, `deploy_wsl.sh`,
`activate_windows.ps1`, and the UCI verification scripts.
