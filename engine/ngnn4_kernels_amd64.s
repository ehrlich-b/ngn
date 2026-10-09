//go:build amd64 && amd64.v3 && !purego && !race

#include "textflag.h"

// Four independent row dots share each 32-byte input vector. Pair saturation
// cannot occur at +/-64 weights. Lane sums and horizontal reductions are
// bounded by the full absolute dot (66,846,720), so reassociation is exact.
TEXT ·ngnn4LinearRows(SB), NOSPLIT, $0-72
	MOVQ dst_base+0(FP), DI
	MOVQ dst_len+8(FP), CX
	MOVQ input_len+32(FP), R12
	MOVQ weights_base+48(FP), R8
	MOVL $0x00010001, AX
	MOVD AX, X5
	VPBROADCASTD X5, Y5
Rows_outputs:
	LEAQ (R8)(R12*1), R9
	LEAQ (R9)(R12*1), R10
	LEAQ (R10)(R12*1), R11
	MOVQ input_base+24(FP), SI
	MOVQ R12, DX
	VPXOR Y0, Y0, Y0
	VPXOR Y1, Y1, Y1
	VPXOR Y2, Y2, Y2
	VPXOR Y3, Y3, Y3
Rows_inputs:
	VMOVDQU (SI), Y4
	VPMADDUBSW (R8), Y4, Y6
	VPMADDWD Y5, Y6, Y6
	VPADDD Y6, Y0, Y0
	VPMADDUBSW (R9), Y4, Y6
	VPMADDWD Y5, Y6, Y6
	VPADDD Y6, Y1, Y1
	VPMADDUBSW (R10), Y4, Y6
	VPMADDWD Y5, Y6, Y6
	VPADDD Y6, Y2, Y2
	VPMADDUBSW (R11), Y4, Y6
	VPMADDWD Y5, Y6, Y6
	VPADDD Y6, Y3, Y3
	ADDQ $32, SI
	ADDQ $32, R8
	ADDQ $32, R9
	ADDQ $32, R10
	ADDQ $32, R11
	SUBQ $32, DX
	JNZ Rows_inputs
	// Reduce each eight-lane partial sum, then pack the four scalar dots.
	VEXTRACTI128 $1, Y0, X4
	VPADDD X4, X0, X0
	VEXTRACTI128 $1, Y1, X4
	VPADDD X4, X1, X1
	VEXTRACTI128 $1, Y2, X4
	VPADDD X4, X2, X2
	VEXTRACTI128 $1, Y3, X4
	VPADDD X4, X3, X3
	VPHADDD X1, X0, X0
	VPHADDD X3, X2, X2
	VPHADDD X2, X0, X0
	VMOVDQU X0, (DI)
	MOVQ R11, R8
	ADDQ $16, DI
	SUBQ $4, CX
	JNZ Rows_outputs
	VZEROUPPER
	RET

// Signed division toward zero: (x + ((x >> 31) & 63)) >> 6.
// Adding the correction cannot overflow int32 because it is zero for x>=0.
// Bias is sign-extended before the [0,255] clamp. Each squared weighted
// product fits int32 (255^2*32768); widen before adding any two products.
TEXT ·ngnn4Output(SB), NOSPLIT, $0-80
	MOVQ affine_base+0(FP), SI
	MOVQ bias_base+24(FP), DI
	MOVQ weights_base+48(FP), R8
	MOVQ affine_len+8(FP), CX
	VPXOR Y5, Y5, Y5
	VPXOR Y6, Y6, Y6
	MOVL $255, AX
	MOVD AX, X7
	VPBROADCASTD X7, Y7
	MOVL $63, AX
	MOVD AX, X4
	VPBROADCASTD X4, Y4
Layer2Output_loop:
	VMOVDQU (SI), Y0
	VPSRAD $31, Y0, Y1
	VPAND Y4, Y1, Y1
	VPADDD Y1, Y0, Y0
	VPSRAD $6, Y0, Y0
	VPMOVSXWD (DI), Y1
	VPADDD Y1, Y0, Y0
	VPMAXSD Y5, Y0, Y0
	VPMINSD Y7, Y0, Y0
	VPMULLD Y0, Y0, Y0
	VPMOVSXWD (R8), Y1
	VPMULLD Y1, Y0, Y0
	VPMOVSXDQ X0, Y2
	VEXTRACTI128 $1, Y0, X1
	VPMOVSXDQ X1, Y3
	VPADDQ Y2, Y6, Y6
	VPADDQ Y3, Y6, Y6
	ADDQ $32, SI
	ADDQ $16, DI
	ADDQ $16, R8
	SUBQ $8, CX
	JNZ Layer2Output_loop
	VEXTRACTI128 $1, Y6, X0
	VPADDQ X0, X6, X6
	VPSHUFD $0x4e, X6, X0
	VPADDQ X0, X6, X6
	MOVQ X6, AX
	MOVQ AX, ret+72(FP)
	VZEROUPPER
	RET

// Original NGN affine kernel, derived from unsigned activations and the
// [-64,64] weight bound. Four inputs times eight outputs per register.
TEXT ·ngnn4Linear(SB), NOSPLIT, $0-72
	MOVQ dst_base+0(FP), DI
	MOVQ dst_len+8(FP), CX
	MOVQ input_base+24(FP), SI
	MOVQ weights_base+48(FP), R8
	MOVQ CX, R9
	SHLQ $2, R9
	MOVL $0x00010001, AX
	MOVD AX, X6
	VPBROADCASTD X6, Y6
Layer2_outputs:
	VPXOR Y0, Y0, Y0
	MOVQ SI, R10
	MOVQ R8, R11
	MOVQ input_len+32(FP), R12
Layer2_inputs:
	VPBROADCASTD (R10), Y1
	VMOVDQU (R11), Y2
	VPMADDUBSW Y2, Y1, Y3
	VPMADDWD Y6, Y3, Y3
	VPADDD Y3, Y0, Y0
	ADDQ $4, R10
	ADDQ R9, R11
	SUBQ $4, R12
	JNZ Layer2_inputs
	VMOVDQU Y0, (DI)
	ADDQ $32, DI
	ADDQ $32, R8
	SUBQ $8, CX
	JNZ Layer2_outputs
	VZEROUPPER
	RET

// Each 16-lane block is clamped before narrowing. Permute qwords after
// int16 packing to restore square order across AVX2's 128-bit lanes.
TEXT ·ngnn4PackInput(SB), NOSPLIT, $0-72
	MOVQ dst_base+0(FP), DI
	MOVQ us_base+24(FP), SI
	MOVQ them_base+48(FP), R8
	MOVQ us_len+32(FP), CX
	MOVQ CX, R9
	MOVQ $2, R10
	VPXOR Y5, Y5, Y5
	MOVL $255, AX
	MOVD AX, X6
	VPBROADCASTD X6, Y6
PackInput_loop:
	VMOVDQU (SI), Y0
	VMOVDQU 32(SI), Y1
	VPMAXSD Y5, Y0, Y0
	VPMAXSD Y5, Y1, Y1
	VPMINSD Y6, Y0, Y0
	VPMINSD Y6, Y1, Y1
	VPACKSSDW Y1, Y0, Y0
	VPERMQ $0xD8, Y0, Y0
	VEXTRACTI128 $1, Y0, X1
	VPACKUSWB X1, X0, X0
	VMOVDQU X0, (DI)
	ADDQ $64, SI
	ADDQ $16, DI
	SUBQ $16, CX
	JNZ PackInput_loop
	SUBQ $1, R10
	JZ PackInput_done
	MOVQ R8, SI
	MOVQ R9, CX
	JMP PackInput_loop
PackInput_done:
	VZEROUPPER
	RET
