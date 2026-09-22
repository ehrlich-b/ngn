//go:build amd64 && amd64.v3

#include "textflag.h"

// Each signed int32 accumulator is clipped to 0..255 before squaring.
// VPMULLD and VPADDD preserve the exact int32 arithmetic of the fast-safe
// Go oracle, including modular behavior for adversarial test inputs.
TEXT ·k4OutputDotAVX2(SB), NOSPLIT, $0-36
	MOVQ stm+0(FP), AX
	MOVQ nonSTM+8(FP), BX
	MOVQ stmWeights+16(FP), CX
	MOVQ nonSTMWeights+24(FP), DX

	VPXOR Y12, Y12, Y12
	VPXOR Y15, Y15, Y15
	MOVL $255, R8
	VMOVD R8, X14
	VPBROADCASTD X14, Y14

	MOVL $96, R9
k4_output_block:
	VMOVDQU (AX), Y0
	VPMAXSD Y15, Y0, Y0
	VPMINSD Y14, Y0, Y0
	VPMULLD Y0, Y0, Y0
	VPMOVSXWD (CX), Y1
	VPMULLD Y1, Y0, Y0
	VPADDD Y0, Y12, Y12

	VMOVDQU (BX), Y0
	VPMAXSD Y15, Y0, Y0
	VPMINSD Y14, Y0, Y0
	VPMULLD Y0, Y0, Y0
	VPMOVSXWD (DX), Y1
	VPMULLD Y1, Y0, Y0
	VPADDD Y0, Y12, Y12

	ADDQ $32, AX
	ADDQ $32, BX
	ADDQ $16, CX
	ADDQ $16, DX
	DECL R9
	JNZ k4_output_block

	VEXTRACTI128 $1, Y12, X0
	VPADDD X0, X12, X12
	VPHADDD X12, X12, X12
	VPHADDD X12, X12, X12
	VMOVD X12, R8
	MOVL R8, ret+32(FP)
	VZEROUPPER
	RET
