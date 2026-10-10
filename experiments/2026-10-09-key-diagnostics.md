# Key-order diagnostics before any replacement regate — session38

The current October9 plan asks for diagnostics of the failed key replacement
before considering another provenance gate. This does not reopen its completed
400-game verdict, extend that sample or authorize seed shopping.

Compare two clean snapshots of accepted `ac658e2`, differing only in the key
initializer from the already tested `night/key-after-see` patch. Both use the
same diagnostic-only TT counters, Hash64, NGNN3 be34532c, eight frozen FENs
and iterative depth8. Run on forge, CPUs8–11, nice10, inside the required
CPUQuota400%/MemoryMax5G/MemorySwapMax0 scope, Go1.25.5/GOAMD64v3.
Games and production code are not changed.

Print duplicates/zero keys among all789 active keys and the one-bit count for
every bit. The old and candidate TT both select buckets with the lowest20 bits
at Hash64 and store/check all64 bits. Frozen diagnostic threshold: zero
duplicates, zero zero-keys, no individual bit farther than4.5 binomial standard
deviations from balance. This is a gross-defect diagnostic, not proof of perfect
randomness or Elo neutrality. Print per-FEN probes, true full-key hits, occupied
index mismatches and evictions. Those counts diagnose concentration; they have
no strength acceptance threshold and never substitute for games.

If a concrete defect appears, fix that defect before the follow-up gate.
The current plan explicitly requests one more gate after these diagnostics;
with no defect, reuse the exact initializer on the immediate accepted base,
without inventing a seed or pooling/altering the previous failed sample.
That completed experiment remains shelved. Freeze the new actual-base
manifest before games, use the provenance upper95>0/zero-fault rule, and
permit no extension or automatic adoption. No architecture adoption, claim
or publication follows from these diagnostics.
