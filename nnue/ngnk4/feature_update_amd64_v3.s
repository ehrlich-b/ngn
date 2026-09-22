//go:build amd64 && amd64.v3

#include "textflag.h"

// The wide int32 accumulator is updated with signed-extended int16 rows.
// Vector addition/subtraction matches the Go oracle lane for lane.
TEXT ·k4AddFeatureRowAVX2(SB), NOSPLIT, $0-16
	MOVQ destination+0(FP), AX
	MOVQ row+8(FP), BX
	MOVL $96, CX
k4_add_feature_block:
	VMOVDQU (AX), Y0
	VPMOVSXWD (BX), Y1
	VPADDD Y1, Y0, Y0
	VMOVDQU Y0, (AX)
	ADDQ $32, AX
	ADDQ $16, BX
	DECL CX
	JNZ k4_add_feature_block
	VZEROUPPER
	RET

TEXT ·k4SubtractFeatureRowAVX2(SB), NOSPLIT, $0-16
	MOVQ destination+0(FP), AX
	MOVQ row+8(FP), BX
	MOVL $96, CX
k4_subtract_feature_block:
	VMOVDQU (AX), Y0
	VPMOVSXWD (BX), Y1
	VPSUBD Y1, Y0, Y0
	VMOVDQU Y0, (AX)
	ADDQ $32, AX
	ADDQ $16, BX
	DECL CX
	JNZ k4_subtract_feature_block
	VZEROUPPER
	RET
