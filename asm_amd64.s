
// +build !race

#include "textflag.h"

TEXT ·casUint32(SB),NOSPLIT,$0-17
	MOVQ	addr+0(FP), BP
	MOVL	old+8(FP), AX
	MOVL	new+12(FP), CX
	LOCK
	CMPXCHGL	CX, 0(BP)
	SETEQ	swapped+16(FP)
	RET


TEXT ·spin(SB),NOSPLIT,$0-0
	MOVL	cycles+0(FP), AX
again:
	PAUSE
	SUBL	$1, AX
	JNZ	again
	RET

