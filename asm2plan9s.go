package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"

	"github.com/pkg/errors"
)

var (
	reg, xreg *regexp.Regexp
)

func init() {
	reg = regexp.MustCompile("([^R])(AX|CX|DX|BX|SP|BP|SI|DI)")
	xreg = regexp.MustCompile("([^0A-D])(X|Y)([^M])")
}

// Assemble will provide the byte codes for any instructions flagged with the sigil // + [INSTR]
func Assemble(input io.Reader) ([]byte, error) {
	inBuf := bufio.NewScanner(input)
	result := bytes.NewBuffer(nil)
	output := bufio.NewWriter(result)

	var line []byte
	sigil := []byte("// +")
	for ln := 1; inBuf.Scan(); ln++ {
		line = inBuf.Bytes()
		start := bytes.Index(line, sigil)
		if start == -1 {
			fmt.Fprintf(output, "%s\n", line)
			continue
		}
		instr := bytes.TrimSpace(bytes.SplitN(line[start+len(sigil):], []byte("/*"), 2)[0])
		if idx := bytes.Index(line[:start], []byte(`\`)); idx != -1 {
			start = idx // Adjust for in-macro lines
		}
		if idx := bytes.Index(line[:start], []byte("#define")); idx != -1 {
			spl := bytes.SplitN(line[idx:start], []byte{' '}, 3)
			fmt.Fprintf(output, "%s %s", spl[0], spl[1])
		}
		byteCode, err := yasm(convertInstr(instr))
		if err != nil {
			return nil, errors.Wrapf(err, "Line %d", ln)
		}

		toPlan9s(byteCode, output)
		fmt.Fprintf(output, "%s\n", line[start:])
	}

	if err := output.Flush(); err != nil {
		errors.Wrapf(err, "Bufio error")
	}
	return result.Bytes(), nil
}

// convertInstr converts the GoAsm format (plan9 order) into Intel style for yasm.
func convertInstr(instr []byte) []byte {

	instr = bytes.ToUpper(instr)
	if reg.Match(instr) || xreg.Match(instr) {
		instr = reg.ReplaceAll(instr, []byte("${1}R$2"))
		instr = xreg.ReplaceAll(instr, []byte("${1}${2}MM$3"))
		flds := bytes.FieldsFunc(instr, func(r rune) bool {
			return r == ' ' || r == '\t' || r == ','
		})

		switch {
		case bytes.Contains(flds[1], []byte{'$'}) &&
			!bytes.Contains(flds[len(flds)-1], []byte{'$'}):
			// [f3, ][f4, ]f2, f1
			switch len(flds) {
			case 5:
				instr = []byte(fmt.Sprintf("%s %s, %s, %s, %s", flds[0], flds[4], flds[2], flds[3], flds[1]))
			case 4:
				instr = []byte(fmt.Sprintf("%s %s, %s, %s", flds[0], flds[3], flds[2], flds[1]))
			case 3:
				instr = []byte(fmt.Sprintf("%s %s, %s", flds[0], flds[2], flds[1]))
			}
		case !bytes.Contains(flds[len(flds)-1], []byte{'$'}):
			// f2, [f3, ][f4, ]f1
			switch len(flds) {
			case 5:
				instr = []byte(fmt.Sprintf("%s %s, %s, %s, %s", flds[0], flds[4], flds[1], flds[2], flds[3]))
			case 4:
				instr = []byte(fmt.Sprintf("%s %s, %s, %s", flds[0], flds[3], flds[1], flds[2]))
			case 3:
				instr = []byte(fmt.Sprintf("%s %s, %s", flds[0], flds[2], flds[1]))
			}
		}
		instr = bytes.Replace(instr, []byte{'$'}, []byte{' '}, -1)
		instr = bytes.Replace(instr, []byte{'('}, []byte{'['}, -1)
		instr = bytes.Replace(instr, []byte{')'}, []byte{']'}, -1)
	}

	return instr
}

//
// yasm-assemble-disassemble-roundtrip-sse.txt
//
// franks-mbp:sse frankw$ more assembly.asm
// [bits 64]
//
// VPXOR   YMM4, YMM2, YMM3    ; X4: Result
// franks-mbp:sse frankw$ yasm assembly.asm
// franks-mbp:sse frankw$ hexdump -C assembly
// 00000000  c5 ed ef e3                                       |....|
// 00000004
// franks-mbp:sse frankw$ echo 'lbl: db 0xc5, 0xed, 0xef, 0xe3' | yasm -f elf - -o assembly.o
// franks-mbp:sse frankw$ gobjdump -d -M intel assembly.o
//
// assembly.o:     file format elf32-i386
//
//
// Disassembly of section .text:
//
// 00000000 <.text>:
// 0:   c5 ed ef e3             vpxor  ymm4,ymm2,ymm3

func yasm(instr []byte) ([]byte, error) {

	tmpfile, err := ioutil.TempFile("", "asm2plan9s")
	if err != nil {
		return nil, err
	}
	asmFile := tmpfile.Name() + ".asm"
	objFile := tmpfile.Name() + ".obj"
	os.Rename(tmpfile.Name(), asmFile)
	defer os.Remove(asmFile)

	if _, err := tmpfile.Write(append([]byte("[bits 64]\n"), instr...)); err != nil {
		return nil, err
	}
	if err := tmpfile.Close(); err != nil {
		return nil, err
	}

	app := "yasm"
	arg0 := "-o" + objFile
	cmd := exec.Command(app, arg0, asmFile)

	defer os.Remove(objFile) // output file created by yasm
	cmb, err := cmd.CombinedOutput()
	if err != nil {
		yasmErr := bytes.Replace(cmb, []byte(asmFile+":2:"), []byte("\t"), -1)
		return nil, errors.Errorf("YASM error on '%s':\n %s", bytes.TrimSpace(instr), yasmErr)
	}

	objcode, err := ioutil.ReadFile(objFile)
	if err != nil {
		return nil, err
	}

	return objcode, nil
}

func toPlan9s(objcode []byte, output io.Writer) {

	output.Write([]byte("    "))
	for ln := len(objcode); ln > 0; {
		switch {
		case ln >= 4:
			fmt.Fprintf(output, "LONG $0x%02X%02X%02X%02X",
				objcode[3], objcode[2], objcode[1], objcode[0])
			objcode = objcode[4:]
			ln -= 4
		case ln >= 2:
			fmt.Fprintf(output, "WORD $0x%02X%02X", objcode[1], objcode[0])
			objcode = objcode[2:]
			ln -= 2
		default:
			fmt.Fprintf(output, "BYTE $0x%02X", objcode[0])
			objcode = objcode[1:]
			ln--
		}
		if ln != 0 {
			fmt.Fprint(output, "; ")
		} else {
			fmt.Fprint(output, " ")
		}
	}
}
