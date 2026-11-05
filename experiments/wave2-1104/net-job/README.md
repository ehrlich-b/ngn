# Net job consumer fixture

The owned integer H16 network is deterministic and has no external training
inputs. Reproduce its 24,704 bytes with `python3 experiments/wave2-1104/net-job/make_model.py --check`.

From the repository root, with the normal Go cache environment and an external
output directory:

```
nice -n 10 python3 experiments/wave2-1104/net-job/time_v.py python3 -m unittest scripts.wave2.test_net_job -v
nice -n 10 python3 experiments/wave2-1104/net-job/time_v.py python3 experiments/wave2-1104/net-job/run_fixture.py --output /absolute/evidence
```

`result.json` is the actual deterministic output of the real consumer fixture,
recorded with `run_fixture.py --record`. A normal run compares every recorded
field. The Go build, candidate source tarball, binary, parity/startup logs,
identity report and adoption manifest remain in the external output directory.
The independent oracle checks the selected embedded model as well as the
format-specific existing trainer parity fixture. The four positions include
castling context, en passant and an endgame. Both consumers run fresh processes,
Threads=1, Hash=64, OwnBook=false and depth4.

The synthetic training and match programs are process fixtures for receipt and
adoption behavior. They do not train on a corpus or play chess. The real consumer
build uses the bounded source recipe in run_fixture.py and does not copy the
original large models. No files are removed from either the checkout or the
private candidate to satisfy checks. No fixture result changes default play or
makes an Elo claim. The source/binary/evidence packet can exceed 5 MiB; its
manifest measures retained bytes and states why those artifacts are retained.
