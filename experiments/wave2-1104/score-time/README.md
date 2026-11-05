W2ScoreTime is a UCI spin option (0 or 1), shipped at 0, outside the SPSA
registry. Only completed, in-window root iterations supply a score. An omitted
score, either mate range, and each search reset break the sequence. A cp decline
adds min(loss/200, 0.25) to a neutral factor of 1; that multiplies stability and
node effort before the existing composed bounds. Hard budgets and the reserve
remain independent.

From the repository root, run the exact focused fragment with:

    nice -n 10 experiments/wave2-1104/score-time/run.sh

Append `-v` to emit the trace. The wrapper sanitizes credential variables, pins
two CPUs and tensor threads, selects shared language caches and a sibling tmp
directory, and reports native wait4 elapsed time, CPU time and peak RSS. These
are correctness runs, not speed samples.

traces.json contains the score inputs and expected off/on admission decisions.
result.log is the captured verbose execution; result.json binds its input hashes
and measured outcome. Authoritative execution evidence also lives outside the
checkout; the result copy here is required for artifact review.

The off trace is compared with the existing callback without score inputs.
The stable 100cp decline keeps the move and node share unchanged: at 901ms the
old 1s soft target stops, while the 1.25s target continues; both stop at the same
hard deadline. The suite also runs a real completed aspiration re-search,
checks the callback before and after a reused manager starts another search,
interrupts a real failed aspiration before completion, and compares real
1000-node searches with the feature off and on. Fake-clock probes force the
existing rate limiter to sample; no production timing hook was added.

timed-match.json remains DEFERRED. There are no real-clock games, so this fixture
establishes no Elo result and does not justify a default change.
