//go:build amd64 && amd64.v3 && !purego && !race

#include "textflag.h"

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
