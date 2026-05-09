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

loop:
    TESTQ CX, CX
    JZ done

    // Process elements 0-7
    VMOVUPS 0(BX), Y3
    VSUBPS Y0, Y3, Y3
    VMULPS Y3, Y3, Y3         // Y3 = diff^2 for [0:8]

    // Process elements 8-11
    VMOVUPS 32(BX), X4
    VSUBPS X1, X4, X4
    VMULPS X4, X4, X4         // X4 = diff^2 for [8:12]

    // Process elements 12-13
    VMOVSD 48(BX), X5
    VSUBPS X2, X5, X5
    VMULPS X5, X5, X5         // X5 = diff^2 for [12:14]

    // Add elements 12-13 to 8-11
    VADDPS X5, X4, X4         // X4 = sums for upper half

    // Extract upper 128 bits of Y3 to X6
    VEXTRACTF128 $1, Y3, X6

    // Add upper 128 (X6) and lower 128 (X3) of Y3
    VADDPS X6, X3, X7

    // Add X4 to X7
    VADDPS X4, X7, X7

    // Horizontal add to sum the 4 floats in X7
    VHADDPS X7, X7, X7
    VHADDPS X7, X7, X7

    // Store the resulting single float
    VMOVSS X7, 0(DX)

    // Advance pointers
    ADDQ $60, BX     // sizeof(VectorRecord)
    ADDQ $4, DX      // sizeof(float32)
    DECQ CX
    JMP loop

done:
    VZEROUPPER
    RET
