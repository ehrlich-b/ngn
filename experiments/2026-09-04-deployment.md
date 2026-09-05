# Current September correctness deployment

**Deployed playing source: `70eb527`**, the proved internal-castling/root-terminal
repair. The completed 400-game gate passed its predeclared correctness rule:
99/205/96 W/D/L, paired +2.6 [−20,+25], zero flags/errors. Strength is inconclusive;
the last absolute pin remains **2726 [2668,2784]**. See the
[full acceptance record](2026-09-04-castling-repair.md).

| Installed platform | SHA-256 |
|---|---|
| Linux amd64/v3 | `80b7488a16ad66f80ff0c618b64baa6445d4c24cdedb78e5d2e87c3a57d32b3e` |
| Windows amd64/v3 | `131b9bbf1cb064bd6c1edd5513db8e6c670eba154231f3aadd66cc1942a85e7d` |
| Mac arm64 | `1010178789b5560583c0ee430934dc4e12473a7dbbdbf07598a30bb3185a3a0f` |

WSL `/home/ehrli/repos/ngn` fast-forwarded cleanly from `71399c0` to `e02be60`
(runtime `70eb527` plus offline tuning repairs and review/acceptance records).
Defaults are WSL `build/ngn` and `build/ngn.exe`, Windows
`C:\Users\ehrli\ngn\ngn.exe` and `ngn\repin\ngn.exe`, and Mac `build/ngn` /
`build/ngn.exe`. The frozen native Windows game-tested binary was installed
unchanged. The existing corrected harness binaries remain installed.

Native WSL Go1.25.5 full short and short/race suites passed (engine packages
5.835s and 23.765s). Installed Linux, Windows and Mac binaries passed black EP
evasion, illegal-castle rejection (Kf1), and fifty-move mate-priority UCI checks,
including clean quit. Native test workers exited before deployment and remain idle.

Rollback: WSL branch `backup/pre-castle-20260904`; all replaced defaults have
`.pre-castle-20260904` siblings. The earlier `.pre-repair-20260904` backups remain.
Logs and reviewed installation scripts are `output/recovery-2026-09-04/castle-*`,
`deploy_castle_wsl.sh`, `activate_castle_windows.ps1` and verification scripts.
No qsearch prototype or T8 vector is installed.

The earlier deployment below is retained as historical provenance.

# Earlier September correctness deployment

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
