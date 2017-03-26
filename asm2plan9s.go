package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

func main() {

	if len(os.Args) < 2 {
		fmt.Println(`error: no input specified

usage: asm2plan9s file
  will in-place update the assembly file with proper BYTE sequence as generated by YASM`)
		return
	}
	fmt.Println("Processing", os.Args[1])
	source, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}

	result, err := assemble(source)
	if err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(os.Args[1], result, 0600)
	if err != nil {
		log.Fatalf("writeLines: %s", err)
	}
}

// assemble assembles an array to lines into their
// resulting plan9 equivalent
func assemble(lines []byte) ([]byte, error) {
	result := bytes.NewBuffer(make([]byte, 0, len(lines)))
	inBuf := bufio.NewScanner(bytes.NewReader(lines))
	outBuf := bufio.NewWriter(result)

	var line []byte
	sigil := []byte("// +")
	for ln := 1; inBuf.Scan(); ln++ {
		line = inBuf.Bytes()
		if !bytes.Contains(line, sigil) {
			_, err := outBuf.Write(line)
			if err != nil {
				return nil, errors.Wrapf(err, "Line %d", ln)
			}
			continue
		}
		start := bytes.Index(line, sigil)
		instr := bytes.TrimSpace(bytes.SplitN(line[start+len(sigil):], []byte("/*"), 2)[0])
		byteCode, err := yasm(instr)
		//byteCode, err := convertInstr(instr)
		if err != nil {
			return nil, errors.Wrapf(err, "Line %d", ln)
		}

		outLine, err := toPlan9s(byteCode, string(instr), len(instr), false)
		byteCode = []byte(outLine)

		if bytes.Contains(line[:start], []byte(`\`)) {
			byteCode = bytes.Replace(byteCode, []byte("//"), []byte(`\ //`), 1)
			//outBuf.Write([]byte(` \ `))
		}
		outBuf.Write([]byte(byteCode))
		//outBuf.Write(line[start:])
		if err := outBuf.Flush(); err != nil {
			fmt.Println("Bufio error", err)
		}
	}
	return result.Bytes(), nil
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

	content := append([]byte("[bits 64]\n"), instr...)
	tmpfile, err := ioutil.TempFile("", "asm2plan9s")
	if err != nil {
		return nil, err
	}

	if _, err := tmpfile.Write(content); err != nil {
		return nil, err
	}
	if err := tmpfile.Close(); err != nil {
		return nil, err
	}

	asmFile := tmpfile.Name() + ".asm"
	objFile := tmpfile.Name() + ".obj"
	os.Rename(tmpfile.Name(), asmFile)

	defer os.Remove(asmFile) // clean up
	defer os.Remove(objFile) // clean up

	app := "yasm"
	arg0 := "-o"
	arg1 := objFile
	arg2 := asmFile

	cmd := exec.Command(app, arg0, arg1, arg2)
	cmb, err := cmd.CombinedOutput()
	if err != nil {
		yasmErrs := bytes.Split(cmb[len(asmFile)+1:], []byte(":"))
		yasmErr := bytes.Join(yasmErrs[1:], []byte(":"))
		return nil, errors.Errorf("YASM error on '%s': %s", bytes.TrimSpace(instr), yasmErr)
	}

	objcode, err := ioutil.ReadFile(objFile)
	if err != nil {
		return nil, err
	}

	return objcode, nil
}

func toPlan9s(objcode []byte, instr string, commentPos int, inDefine bool) (string, error) {

	sline := "    "
	i := 0
	// First do LONGs (as many as needed)
	for ; len(objcode) >= 4; i++ {
		if i != 0 {
			sline += "; "
		}
		sline += fmt.Sprintf("LONG $0x%02x%02x%02x%02x", objcode[3], objcode[2], objcode[1], objcode[0])

		objcode = objcode[4:]
	}

	// Then do a WORD (if needed)
	if len(objcode) >= 2 {

		if i != 0 {
			sline += "; "
		}
		sline += fmt.Sprintf("WORD $0x%02x%02x", objcode[1], objcode[0])

		i++
		objcode = objcode[2:]
	}

	// And close with a BYTE (if needed)
	if len(objcode) == 1 {
		if i != 0 {
			sline += "; "
		}
		sline += fmt.Sprintf("BYTE $0x%02x", objcode[0])

		i++
		objcode = objcode[1:]
	}

	if inDefine {
		if commentPos > commentPos-2-len(sline) {
			if commentPos-2-len(sline) > 0 {
				sline += strings.Repeat(" ", commentPos-2-len(sline))
			}
		} else {
			sline += " "
		}
		sline += `\ `
	} else {
		if commentPos > len(sline) {
			if commentPos-len(sline) > 0 {
				sline += strings.Repeat(" ", commentPos-len(sline))
			}
		} else {
			sline += " "
		}
	}

	sline += "// + " + instr

	return sline, nil
}

// startsAfterLongWordByteSequence determines if an assembly instruction
// starts on a position after a combination of LONG, WORD, BYTE sequences
func startsAfterLongWordByteSequence(prefix string) bool {

	if len(strings.TrimSpace(prefix)) != 0 && !strings.HasPrefix(prefix, "    LONG $0x") &&
		!strings.HasPrefix(prefix, "    WORD $0x") && !strings.HasPrefix(prefix, "    BYTE $0x") {
		return false
	}

	length := 4 + len(prefix) + 1

	for objcodes := 3; objcodes <= 8; objcodes++ {

		ls, ws, bs := 0, 0, 0

		oc := objcodes

		for ; oc >= 4; oc -= 4 {
			ls++
		}
		if oc >= 2 {
			ws++
			oc -= 2
		}
		if oc == 1 {
			bs++
		}
		size := 4 + ls*(len("LONG $0x")+8) + ws*(len("WORD $0x")+4) + bs*(len("BYTE $0x")+2) + (ls+ws+bs-1)*len("; ")

		if length == size+2 || // comment starts after a space
			length == size+4 { // comment starts after a space, bash slash and another space
			return true
		}
	}
	return false
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}
