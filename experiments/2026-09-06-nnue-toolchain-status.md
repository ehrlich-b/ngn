# NNUE public toolchain status

Status: execution resumed after the user explicitly authorized full WSL access and bidirectional file/source/log transfers between both home-network development machines. No trained network or strength verdict exists.

The retained log now proves Cargo101 was a compiler-version incompatibility: pinned Bullet uses let-chains in crates/compiler/src/tensor/pattern.rs despite declaring Rust1.87; Rust1.87 reports E0658 before GPU execution. Sol is installing exact user-local Rust1.88.0 with official checksum verification, retaining the original toolchain and failed evidence, and rerunning unchanged locked/offline source and dependencies. No GPU driver change is involved.

The historical permission-blocked attempt below is preserved as evidence; it no longer describes the active authorization.

The reduced-data WSL preflight was approved after automatic review rejected exporting detailed host metadata. The on-host assertions returned fixed PREFLIGHT_PASS. Sol then reported completed RUST_PASS, CUDA_PASS, BULLET_FETCH_PASS, and NATIVE_SM120_PASS statuses. Rust 1.87.0 and CUDA 12.8.1 toolkit-only installation are in the isolated WSL user prefix; the Windows GPU driver was retained.

The native four-element sm_120 kernel passed. The first pinned Bullet NVRTC test attempt returned Cargo exit 101 before GPU execution. The agent confirmed the tracked command is terminal and its resource monitor reaped. The Cargo failure log is retained at:

    /home/ehrli/nnue-public-toolchain-20260906/logs/bullet-nvrtc-smoke.log

Automatic approval review rejected streaming that log because it could disclose local paths, environment details, or other private diagnostics to the Mac. An approved on-host classifier returned only UNCLASSIFIED_FAILURE. No diagnostic payload was relayed or retried. Root verified the public Cargo package name bullet-gpu and its cuda feature, finding no definite command error to correct without the log.

Detailed receipts and installed files remain under the isolated WSL prefix. The fixed statuses support progress reporting; full independent receipt review remains pending. No private source review, training-corpus download, NNUE pilot, real-data training, or NGN integration was performed by this provisioning step.

Resumed execution queue:
1. Inspect the existing terminal Cargo log and resolve the actual build failure.
2. Finish the exact NVRTC kernel test.
3. Run the bounded public tests::cuda::axby test to exercise FP32 cuBLAS.
4. Prepare a resource-bounded forward/backward and optimizer smoke. The upstream test3 uses the bundled public fixture but sets four loader threads and queue depth 512, so it must not be launched unchanged under the current two-thread, small-queue budget. It prints expected loss and does not prove numeric gradients.
5. Resume independent PV repair review, corrected-baseline control, and the existing staged NNUE/SMP roadmap.

All installed tools and prior work are preserved. The accepted engine deployment and rollback remain unchanged.
