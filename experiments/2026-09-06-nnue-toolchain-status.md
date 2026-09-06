# NNUE public toolchain status

The WSL GPU/toolchain entry gates pass. Root read the actual native/NVRTC/cuBLAS logs and version/source receipts; hashes are retained in output/nnue-public-preflight-20260906/root-gpu-smoke-review.json. Sol performed installation and tests in /home/ehrli/nnue-public-toolchain-20260906. The accepted engine deployment and rollback are unchanged.

Pinned Bullet 629ee50000b2afb7b3337595401c830d3b1e0f42 declares Rust1.87 but uses let-chains in crates/compiler/src/tensor/pattern.rs. The original rustc1.87 build failed with 136 E0658 errors before GPU execution (Cargo101, 0.70s, peak RSS181928KB). Its original source, toolchain and log remain preserved. The user explicitly authorized full WSL access and bidirectional source/log transfers, resolving the earlier automatic-review block on reading that diagnostic.

The minimal corrective toolchain is official-checksummed user-local Rust1.88.0, with unchanged pinned Bullet source, Cargo.lock and offline dependencies. CUDA12.8.1 is toolkit-only, retaining the Windows GPU driver; the installed nvcc identifies itself as V12.8.93. Do not infer component versions from the installer name.

Completed actual checks:
- Native four-element CUDA kernel compiled with -arch=sm_120, correct output, cuobjdump target verified.
- cargo test --locked --offline --release -p bullet-gpu --features cuda runtime::tests::cuda::compile_load_execute_kernel -- --exact --nocapture: one test passed, 28.00s including build, peak RSS365784KB.
- cargo test --locked --offline --release -p bullet-gpu --features cuda tests::cuda::axby -- --exact --nocapture: one test passed, 1.60s, peak RSS567192KB. This exercises one narrow FP32 cuBLAS layout, not every matrix operation or derivatives.

The next smoke remains source-review gated. Sol is preparing a bounded 128-hidden dual-perspective SCReLU forward/backward/AdamW harness, numeric checks of representative derivatives, direct checked exports, and Go bridge parity. The bundled public batch fixture is sufficient; no T80 payload or real training is needed yet.

A further pinned upstream setting bug was verified: LocalSettings.batch_queue_size is not passed by ValueTrainer::run; crates/trainer/src/run.rs hardcodes sync_channel(32). The isolated harness must wire the configured capacity through, prove the actual bound, use two loader threads and queue8, and propagate errors. Setting queue8 in LocalSettings alone is not resource enforcement.

Earlier research corrections remain in 2026-09-06-nnue-public-preflight-review.md: ignored test_set, time-seeded SF loader ordering, swallowed checkpoint export errors, and unverified historical teacher score units. No trained network, production NNUE integration, or playing-strength verdict is established by these toolchain tests.
