package main

import (
	"bufio"
	"bytes"
	"log"
	"testing"
)

func TestInstruction(t *testing.T) {

	for n, tst := range []struct {
		testName, ins, out string
		err                error
	}{
		{testName: "Add byte codes",
			ins: "                                 // + VPADDQ  XMM0,XMM1,XMM8",
			out: "    LONG $0xd471c1c4; BYTE $0xc0 // + VPADDQ  XMM0,XMM1,XMM8",
			err: nil,
		},
		{testName: "Correct byte codes",
			ins: "    LONG $0xd471c1c4; BYTE $0xc0 // + VPADDQ  XMM0,XMM1,XMM8",
			out: "    LONG $0xd471c1c4; BYTE $0xc0 // + VPADDQ  XMM0,XMM1,XMM8",
			err: nil,
		},
		{testName: "Incorrect byte codes",
			ins: "    LONG $0x003377bb; BYTE $0xff // + VPADDQ  XMM0,XMM1,XMM8",
			out: "    LONG $0xd471c1c4; BYTE $0xc0 // + VPADDQ  XMM0,XMM1,XMM8",
			err: nil,
		},
		{testName: "In-macro codes",
			ins: `    LONG $0x00000000; BYTE $0xdd                               \ // + VPADDQ  XMM0,XMM1,XMM8`,
			out: `    LONG $0xd471c1c4; BYTE $0xc0 \ // + VPADDQ  XMM0,XMM1,XMM8`,
			err: nil,
		},
		{testName: "Insert byte codes",
			ins: "                                   // + VPALIGNR XMM8, XMM12, XMM12, 0x8",
			out: "    LONG $0x0f1943c4; WORD $0x08c4 // + VPALIGNR XMM8, XMM12, XMM12, 0x8",
			err: nil,
		},
	} {
		inBuf := bytes.NewReader([]byte(tst.ins))
		result := bytes.NewBuffer(make([]byte, 0, len(tst.ins)))
		outBuf := bufio.NewWriter(result)

		err := assemble(inBuf, outBuf)
		if err != tst.err {
			t.Error(err)
		}
		if err = outBuf.Flush(); err != nil {
			log.Fatal("Bufio error: ", err)
		}
		if result.String() != tst.out {
			t.Errorf("Test %d (%s) expected %s\ngot%s", n, tst.testName, tst.out, result.String())
		}
	}
}
