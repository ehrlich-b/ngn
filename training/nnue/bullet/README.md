# NGN K4 Bullet harness

`ngn_k4_train.rs` is maintained here and copied unchanged into
`crates/bullet_lib/examples/` of the pinned Bullet
`629ee50000b2afb7b3337595401c830d3b1e0f42` execution snapshot. Cargo then
builds it as the `ngn_k4_train` example with the snapshot's locked offline
dependencies and CUDA feature.

The harness has three frozen modes:

| mode | BF records | updates | checkpoint interval |
| --- | ---: | ---: | ---: |
| smoke | 1,000,000 | 256 | 256 |
| pilot | 1,000,000 | 1,024 | 128 |
| main | 20,000,000 | 16,384 | 512 |

Every mode starts from seed 26092001 with no imported weights or optimizer
state. Targets are 100% record score (`result_weight=0`) under a scale-400
sigmoid. The 768-feature factorizer is zero-initialized, trained jointly, and
merged into the four transformer buckets on export. Candidate zero and every
scheduled checkpoint retain raw tensors, quantized tensors, and AdamW state.

`raw.bin` is the deployed float tensor stream, not Bullet's default raw save.
The default raw writer omits save-format transforms; this harness instead
applies the factorizer merge and output-layer transpose independently, then
requires every quantized i16 to match that transformed stream. The `probe`
subcommand loads a retained optimizer checkpoint and evaluates a FEN corpus
through Bullet's own feature mapper and graph:

```text
ngn_k4_train probe CHECKPOINT_DIRECTORY FENS_FILE
```

Its tab-delimited output is consumed by `ngnk4bridge parity`, which checks all
raw-to-integer tensors, independent float evaluation, Bullet-vs-Go output,
scalar-vs-incremental Go evaluation, and the prospective quantization gates.

This harness does not decide checkpoint eligibility or promotion. The selector,
manifest-bound `ngnk4bridge`, Go parity tests, and playing gates own those
decisions. The harness refuses an existing output directory and exact-checks BF,
raw, quantized, trailer and optimizer-state sizes before reporting success.
