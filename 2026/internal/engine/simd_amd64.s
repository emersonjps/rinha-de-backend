//go:build amd64
#include "textflag.h"

// func batchDistAVX2(target *[14]float32, records unsafe.Pointer, numRecords int, distances *float32)
TEXT ·batchDistAVX2(SB), NOSPLIT, $0-32
    MOVQ target+0(FP), AX
    MOVQ records+8(FP), BX
    MOVQ numRecords+16(FP), CX
    MOVQ distances+24(FP), DX

    // Load target vector into AVX registers
    VMOVUPS 0(AX), Y0         // Y0 = target[0:8]
    VMOVUPS 32(AX), X1        // X1 = target[8:12]
    VMOVSD  48(AX), X2        // X2 = target[12:14] (lower 64 bits)

l4:
    CMPQ CX, $4
    JL l1

    // --- Record 1 ---
    VMOVUPS 0(BX), Y3
    VSUBPS Y0, Y3, Y3
    VMULPS Y3, Y3, Y3

    VMOVUPS 32(BX), X4
    VSUBPS X1, X4, X4
    VMULPS X4, X4, X4

    VMOVSD 48(BX), X5
    VSUBPS X2, X5, X5
    VMULPS X5, X5, X5

    VADDPS X5, X4, X4
    VEXTRACTF128 $1, Y3, X6
    VADDPS X6, X3, X7
    VADDPS X4, X7, X7
    VHADDPS X7, X7, X7
    VHADDPS X7, X7, X7
    VMOVSS X7, 0(DX)

    // --- Record 2 ---
    VMOVUPS 60(BX), Y3
    VSUBPS Y0, Y3, Y3
    VMULPS Y3, Y3, Y3

    VMOVUPS 92(BX), X4
    VSUBPS X1, X4, X4
    VMULPS X4, X4, X4

    VMOVSD 108(BX), X5
    VSUBPS X2, X5, X5
    VMULPS X5, X5, X5

    VADDPS X5, X4, X4
    VEXTRACTF128 $1, Y3, X6
    VADDPS X6, X3, X7
    VADDPS X4, X7, X7
    VHADDPS X7, X7, X7
    VHADDPS X7, X7, X7
    VMOVSS X7, 4(DX)

    // --- Record 3 ---
    VMOVUPS 120(BX), Y3
    VSUBPS Y0, Y3, Y3
    VMULPS Y3, Y3, Y3

    VMOVUPS 152(BX), X4
    VSUBPS X1, X4, X4
    VMULPS X4, X4, X4

    VMOVSD 168(BX), X5
    VSUBPS X2, X5, X5
    VMULPS X5, X5, X5

    VADDPS X5, X4, X4
    VEXTRACTF128 $1, Y3, X6
    VADDPS X6, X3, X7
    VADDPS X4, X7, X7
    VHADDPS X7, X7, X7
    VHADDPS X7, X7, X7
    VMOVSS X7, 8(DX)

    // --- Record 4 ---
    VMOVUPS 180(BX), Y3
    VSUBPS Y0, Y3, Y3
    VMULPS Y3, Y3, Y3

    VMOVUPS 212(BX), X4
    VSUBPS X1, X4, X4
    VMULPS X4, X4, X4

    VMOVSD 228(BX), X5
    VSUBPS X2, X5, X5
    VMULPS X5, X5, X5

    VADDPS X5, X4, X4
    VEXTRACTF128 $1, Y3, X6
    VADDPS X6, X3, X7
    VADDPS X4, X7, X7
    VHADDPS X7, X7, X7
    VHADDPS X7, X7, X7
    VMOVSS X7, 12(DX)

    ADDQ $240, BX
    ADDQ $16, DX
    SUBQ $4, CX
    JMP l4

l1:
    TESTQ CX, CX
    JZ done

    VMOVUPS 0(BX), Y3
    VSUBPS Y0, Y3, Y3
    VMULPS Y3, Y3, Y3

    VMOVUPS 32(BX), X4
    VSUBPS X1, X4, X4
    VMULPS X4, X4, X4

    VMOVSD 48(BX), X5
    VSUBPS X2, X5, X5
    VMULPS X5, X5, X5

    VADDPS X5, X4, X4
    VEXTRACTF128 $1, Y3, X6
    VADDPS X6, X3, X7
    VADDPS X4, X7, X7
    VHADDPS X7, X7, X7
    VHADDPS X7, X7, X7
    VMOVSS X7, 0(DX)

    ADDQ $60, BX
    ADDQ $4, DX
    DECQ CX
    JMP l1

done:
    VZEROUPPER
    RET

// func batchDistAVX2_64(target *[14]float32, records unsafe.Pointer, numRecords int, distances *float32)
TEXT ·batchDistAVX2_64(SB), NOSPLIT, $0-32
    MOVQ target+0(FP), AX
    MOVQ records+8(FP), BX
    MOVQ numRecords+16(FP), CX
    MOVQ distances+24(FP), DX

    // Load target vector into AVX registers
    VMOVUPS 0(AX), Y0         // Y0 = target[0:8]
    VMOVUPS 32(AX), X1        // X1 = target[8:12]
    VMOVSD  48(AX), X2        // X2 = target[12:14] (lower 64 bits)

l4_64:
    CMPQ CX, $4
    JL l1_64

    // --- Record 1 ---
    VMOVUPS 0(BX), Y3
    VSUBPS Y0, Y3, Y3
    VMULPS Y3, Y3, Y3

    VMOVUPS 32(BX), X4
    VSUBPS X1, X4, X4
    VMULPS X4, X4, X4

    VMOVSD 48(BX), X5
    VSUBPS X2, X5, X5
    VMULPS X5, X5, X5

    VADDPS X5, X4, X4
    VEXTRACTF128 $1, Y3, X6
    VADDPS X6, X3, X7
    VADDPS X4, X7, X7
    VHADDPS X7, X7, X7
    VHADDPS X7, X7, X7
    VMOVSS X7, 0(DX)

    // --- Record 2 ---
    VMOVUPS 64(BX), Y3
    VSUBPS Y0, Y3, Y3
    VMULPS Y3, Y3, Y3

    VMOVUPS 96(BX), X4
    VSUBPS X1, X4, X4
    VMULPS X4, X4, X4

    VMOVSD 112(BX), X5
    VSUBPS X2, X5, X5
    VMULPS X5, X5, X5

    VADDPS X5, X4, X4
    VEXTRACTF128 $1, Y3, X6
    VADDPS X6, X3, X7
    VADDPS X4, X7, X7
    VHADDPS X7, X7, X7
    VHADDPS X7, X7, X7
    VMOVSS X7, 4(DX)

    // --- Record 3 ---
    VMOVUPS 128(BX), Y3
    VSUBPS Y0, Y3, Y3
    VMULPS Y3, Y3, Y3

    VMOVUPS 160(BX), X4
    VSUBPS X1, X4, X4
    VMULPS X4, X4, X4

    VMOVSD 176(BX), X5
    VSUBPS X2, X5, X5
    VMULPS X5, X5, X5

    VADDPS X5, X4, X4
    VEXTRACTF128 $1, Y3, X6
    VADDPS X6, X3, X7
    VADDPS X4, X7, X7
    VHADDPS X7, X7, X7
    VHADDPS X7, X7, X7
    VMOVSS X7, 8(DX)

    // --- Record 4 ---
    VMOVUPS 192(BX), Y3
    VSUBPS Y0, Y3, Y3
    VMULPS Y3, Y3, Y3

    VMOVUPS 224(BX), X4
    VSUBPS X1, X4, X4
    VMULPS X4, X4, X4

    VMOVSD 240(BX), X5
    VSUBPS X2, X5, X5
    VMULPS X5, X5, X5

    VADDPS X5, X4, X4
    VEXTRACTF128 $1, Y3, X6
    VADDPS X6, X3, X7
    VADDPS X4, X7, X7
    VHADDPS X7, X7, X7
    VHADDPS X7, X7, X7
    VMOVSS X7, 12(DX)

    ADDQ $256, BX
    ADDQ $16, DX
    SUBQ $4, CX
    JMP l4_64

l1_64:
    TESTQ CX, CX
    JZ done_64

    VMOVUPS 0(BX), Y3
    VSUBPS Y0, Y3, Y3
    VMULPS Y3, Y3, Y3

    VMOVUPS 32(BX), X4
    VSUBPS X1, X4, X4
    VMULPS X4, X4, X4

    VMOVSD 48(BX), X5
    VSUBPS X2, X5, X5
    VMULPS X5, X5, X5

    VADDPS X5, X4, X4
    VEXTRACTF128 $1, Y3, X6
    VADDPS X6, X3, X7
    VADDPS X4, X7, X7
    VHADDPS X7, X7, X7
    VHADDPS X7, X7, X7
    VMOVSS X7, 0(DX)

    ADDQ $64, BX
    ADDQ $4, DX
    DECQ CX
    JMP l1_64

done_64:
    VZEROUPPER
    RET
