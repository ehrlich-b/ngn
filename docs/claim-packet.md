The `claimpacket` command checks an existing confirmation receipt and its screen
without starting engines, searching positions, probing occupancy or GPU state,
or creating a new game receipt:

```
go build -o claimpacket ./cmd/claimpacket
./claimpacket --receipt /absolute/path/confirmation/receipt.json
./claimpacket --positions /absolute/path/first-80-openings.txt
```

Receipt verification includes seals, artifact hashes, source and build identity,
the fixed plan, an independent replay of the screen audit and analysis, and the
same candidate/tool/options/anchor identity comparison used for confirmation.
A retained numerical analysis must be reproduced exactly. Use its original Go
version and GOAMD64 build target for historical verification; different targets
can produce a one-step floating-point difference and correctly fail equality.
The position mode replays legal UCI histories and emits canonical board, side,
castling and legally capturable en passant keys, so transpositions can be audited
without a search or external chess library.

Passing verifies historical inputs only. It does not resume an interrupted run,
authorize a tournament, or transfer an old screen to a different engine. Keep
future engine identity, build, net, provenance and matching admitted screen as
explicit prospective inputs. Shared screen/confirmation openings and prior
exposure must be reported; do not replace a frozen selection to hide overlap.
