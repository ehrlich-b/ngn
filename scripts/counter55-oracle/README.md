# Counter 5.5 portable oracle source

`upstream_oracle_test.go` is a test-only adapter for CounterGo commit `63c487ca724c620f71c129d62129c6fb9109c872`. It is copied into a detached checkout of that exact tree and built only with tag `ngn_counter_oracle`. The upstream project is GPL-3.0; the pinned license text and source hashes are retained in the experiment evidence.

The oracle is not linked into NGN and is not a production evaluator. `fixtures.tsv` is shared with NGN's independent parser/evaluator test. Both sides use `GOAMD64=v1` and no Counter `avx` tag for the canonical portable float32 operation order.
