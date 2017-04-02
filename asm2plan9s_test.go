package main

import (
	"bytes"
	"testing"
)

func TestInstruction(t *testing.T) {

	for n, tst := range []struct {
		testName, ins, out string
		err                error
	}{
		{testName: "Add byte codes",
			ins: "                                 // + VPADDQ  XMM0,XMM1,XMM8",
			out: "    LONG $0xD471C1C4; BYTE $0xC0 // + VPADDQ  XMM0,XMM1,XMM8\n",
			err: nil,
		},
		{testName: "Correct byte codes",
			ins: "    LONG $0xD471C1C4; BYTE $0xC0 // + VPADDQ  XMM0,XMM1,XMM8",
			out: "    LONG $0xD471C1C4; BYTE $0xC0 // + VPADDQ  XMM0,XMM1,XMM8\n",
			err: nil,
		},
		{testName: "Incorrect byte codes",
			ins: "    LONG $0x003377BB; BYTE $0xFF // + VPADDQ  XMM0,XMM1,XMM8",
			out: "    LONG $0xD471C1C4; BYTE $0xC0 // + VPADDQ  XMM0,XMM1,XMM8\n",
			err: nil,
		},
		{testName: "In-macro codes",
			ins: "    LONG $0x00000000; BYTE $0xDD                               \\ // + VPADDQ  XMM0,XMM1,XMM8",
			out: "    LONG $0xD471C1C4; BYTE $0xC0 \\ // + VPADDQ  XMM0,XMM1,XMM8\n",
			err: nil,
		},
		{testName: "Insert byte codes",
			ins: "                                   // + VPALIGNR XMM8, XMM12, XMM12, 0x8",
			out: "    LONG $0x0F1943C4; WORD $0x08C4 // + VPALIGNR XMM8, XMM12, XMM12, 0x8\n",
			err: nil,
		},
		{testName: "Multiple lines",
			ins: `                                   // + VPALIGNR XMM8, XMM12, XMM12, 0x8`,
			out: "    LONG $0x0F1943C4; WORD $0x08C4 // + VPALIGNR XMM8, XMM12, XMM12, 0x8\n",
			err: nil,
		},
		{testName: "Plan9 instr",
			ins: "    LONG $0xD471C1C4; BYTE $0xC0 // + VPADDQ  X1, X8, X0",
			out: "    LONG $0xD471C1C4; BYTE $0xC0 // + VPADDQ  X1, X8, X0\n",
			err: nil,
		},
		{testName: "Plan9 avx instr const",
			ins: "    LONG $0xD471C1C4; BYTE $0xC0 // + VSHUFPD $1, X1, X8, X0",
			out: "    LONG $0xC671C1C4; WORD $0x01C0 // + VSHUFPD $1, X1, X8, X0\n",
			err: nil,
		},
		{testName: "Intl instr const",
			ins: "    LONG $0xD471C1C4; BYTE $0xC0 // + SHUFPD XMM0, XMM1, 0X3",
			out: "    LONG $0xC1C60F66; BYTE $0x03 // + SHUFPD XMM0, XMM1, 0X3\n",
			err: nil,
		},
		{testName: "Plan9 instr const",
			ins: "    LONG $0xD471C1C4; BYTE $0xC0 // + SHUFPD $3, X1, X0",
			out: "    LONG $0xC1C60F66; BYTE $0x03 // + SHUFPD $3, X1, X0\n",
			err: nil,
		},
		{testName: "Macro start",
			ins: " #define macro   LONG $0xD471C1C4; BYTE $0xC0 \\ // + SHUFPD $3, X1, X0",
			out: "#define macro    LONG $0xC1C60F66; BYTE $0x03 \\ // + SHUFPD $3, X1, X0\n",
			err: nil,
		},
		{testName: "Intel addressing",
			ins: "     // + VFMADD132PD xmm1, xmm0, [rax]",
			out: "    LONG $0x98F9E2C4; BYTE $0x08 // + VFMADD132PD xmm1, xmm0, [rax]\n",
			err: nil,
		},
		{testName: "GoAsm addressing",
			ins: "     // + VFMADD132PD X0, (AX), X1",
			out: "    LONG $0x98F9E2C4; BYTE $0x08 // + VFMADD132PD X0, (AX), X1\n",
			err: nil,
		},
		{testName: "Go register in instr",
			ins: "     // + ADDSUBPD X0, X1",
			out: "    LONG $0xC8D00F66 // + ADDSUBPD X0, X1\n",
			err: nil,
		},
	} {
		inBuf := bytes.NewReader([]byte(tst.ins))

		result, err := Assemble(inBuf)
		if err != tst.err {
			t.Error(err)
		}
		if string(result) != tst.out {
			t.Errorf("Test %d (%s) expected \n%s\ngot\n%s", n, tst.testName, tst.out, result)
		}
	}
}
