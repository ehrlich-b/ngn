# NNUE public-source preflight review

This is a research addendum, not a provisioning or training receipt. Public primary sources were checked on September 6, 2026. No installation, download of the training payload, GPU smoke, or training was performed for this review.

## GPU/toolchain acceptance

NVIDIA lists RTX 5080 as compute capability 12.0. CUDA 12.8.1's compiler documentation explicitly supports compute_120 and sm_120. The generic archived Blackwell compatibility guide emphasizes compute capability 10.0, so its examples alone are insufficient to specify this consumer GPU's target.

- https://developer.nvidia.com/cuda/gpus
- https://docs.nvidia.com/cuda/archive/12.8.1/cuda-compiler-driver-nvcc/index.html
- https://docs.nvidia.com/cuda/archive/12.8.1/blackwell-compatibility-guide/index.html

The WSL installation must provide Linux toolkit components while retaining the Windows-supplied GPU driver. NVIDIA's guide excludes the cuda, cuda-12-x, and cuda-drivers meta-packages for this purpose. It also documents limited pinned host memory and incomplete NVML queries. Those constraints favor a small queue and bounded memory in the first trainer smoke; an absent NVML process listing cannot prove that a process has stopped.

- https://docs.nvidia.com/cuda/wsl-user-guide/index.html

CUDA 12.8 Update 1 lists nvcc 12.8.90, NVRTC 12.8.93, and cuBLAS 12.8.4.1. Its updated release notes also list a separate cuBLAS 12.8.5 patch. The listed fixes concern particular cuBLASLt algorithms and ztrmm; this review does not establish that the planned simple FP32 network triggers them. Freeze the actual loaded component versions and library paths; do not treat a toolkit directory name as a complete dependency pin.

- https://docs.nvidia.com/cuda/archive/12.8.1/cuda-toolkit-release-notes/index.html#cublas-patch-release-12-8-5

Acceptance still requires an actual GPU context, compiled/runtime kernel path, forward/backward test, finite optimizer update, checkpoint export, and independent Go parity on the host. Documentation proves support in principle, not a working Bullet installation.

## Teacher data provenance

The publisher's archive identifies its T80 data as converted Leela training data and links the converter and filtering tools. Stockfish PR 5333 separately lists test80-2022-08-aug-16tb7p.v6-dd.min.binpack as one training component. Use by Stockfish does not make these labels current Stockfish UCI centipawns.

- https://robotmoon.com/nnue-training-data/
- https://github.com/official-stockfish/Stockfish/pull/5333
- https://github.com/linrock/lc0-data-converter
- https://github.com/linrock/nnue-data

The public conversion script currently requests best score, best move, tablebase rescoring, and deblundering. This is evidence about the published workflow, not a receipt that binds one exact converter revision and invocation to the frozen August 2022 file.

- https://github.com/linrock/lc0-data-converter/blob/master/rescore_tar_files.sh

Before accepting the real-data pilot: identify the score transform and side-to-move/result convention, retain literal decoded examples, verify feature indices against the independent reader, and record any remaining teacher-unit assumption. Do not apply current Stockfish UCI normalization to this archive merely because its transport is binpack. The transport's signed integer score representation alone does not establish its semantic scale.

The official 2023 training wiki reports better results from staged Stockfish-generated then Leela-derived data than from Leela-only training. This is historical evidence to test if our minimal model plateaus, not a claim about the best 2026 recipe or a reason to enlarge the first pipeline pilot.

- https://github.com/official-stockfish/nnue-pytorch/wiki/Training-datasets

The immediate sequence remains: synthetic GPU/gradient/export smoke, bounded real-data pilot with verified label contract, then a separately declared strength experiment. Exact model-format parity is necessary; it does not prove that labels or playing strength are correct.
