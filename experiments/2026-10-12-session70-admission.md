# Final14 immediate-admission nomination — session 70

The singular-correction gate completed at 01:08:35 EDT with 400 games,
200 pairs, 88/224/88 and penta 0.00 [−19.73,+19.73], zero faults and
intact inputs. It is shelved under its frozen positive-lower-bound rule.
Playing source therefore remains f1d0a03, external a4d90b41, compiled
final14 defaults and factorizer off. Rodent retains that frozen package.

Manually nominate only the existing immediate iteration-admission patch
on this immediate base. Reuse candidate a093ac32 and baseline 4f8f631f;
the candidate passed four controls, engine/full short/race, vet and the
accepted-PGO build in session 69. No changed source, baseline rebuild,
training, parity or repeated checks. The tracked session69 patch and
uncommitted topic night/time-admission-final14 preserve the candidate.

Run on forge in a new /work/lanes/ngn-s70-admission-final14 directory,
taskset 4–11, c2, nice10, GOMAXPROCS1, GOAMD64v3, Hash64, Threads1,
OwnBook=false, explicit EvalFile on both sides. Scope MemoryMax3GiB,
MemorySwapMax0, CPUQuota800%; explicitly pin the controller because
AllowedCPUs alone was not enforced in the previous user scope. Guest
currently has 16 vCPUs. Physical host pinning after the 20:37 reboot
remains unverified; record current background load. Its own control
admits only this setup, without resolving the older SEE baseline flag.

Before games write nomination.json, game-manifest.json and input pins.
Fresh own A/A40 at 60+0.6, c2, the same machine/affinity/options/instrument.
Both printed and unrounded penta intervals must contain zero, with no
flag-outs, crashes, illegal/no-move results, watchdogs or incomplete
games. A failed control ends the nomination without candidate games or
retry. Do not reuse singular's 10+0.1 control or the clock diagnostic.

After a passing control, run the original paired SPRT [0,+6], alpha/beta
.05, min400/cap4000, at 60+0.6. KEEP requires natural H1 plus printed
AND unrounded paired lower95 > 0, zero operational faults and intact
inputs. H0/cap shelves; timeout/fault is operationally unscored. Existing
parser preserves its <12 unmatched terminal-game limit and reports
unpaired games separately. This normal-approximation CI is not an exact
statistical or independently calibrated sequential interval. No pooling,
extension, automatic adoption or old-reference sample continuation.

A/A operational cap four hours; gate cap 72 hours. Detached controller
appends status.jsonl and controller.exit/done. A natural winner requires
review and a separately declared 120+1 transfer on its actual package.
Only two forge games run concurrently across all NGN jobs. No new CPU
load may be added to this lane during scored games. Rig Rodent, separate
GPU work, hopper and readers remain untouched.
