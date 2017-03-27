package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

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
	inBuf := bufio.NewScanner(bytes.NewReader(lines))
	result := bytes.NewBuffer(make([]byte, 0, len(lines)))
	outBuf := bufio.NewWriter(result)

	var line []byte
	sigil := []byte("// +")
	for ln := 1; inBuf.Scan(); ln++ {
		line = inBuf.Bytes()
		if !bytes.Contains(line, sigil) {
			outBuf.Write(line)
			continue
		}
		start := bytes.Index(line, sigil)
		instr := bytes.TrimSpace(bytes.SplitN(line[start+len(sigil):], []byte("/*"), 2)[0])
		byteCode, err := yasm(instr)
		//byteCode, err := convertInstr(instr)
		if err != nil {
			return nil, errors.Wrapf(err, "Line %d", ln)
		}

		toPlan9s(byteCode, outBuf)

		if idx := bytes.Index(line[:start], []byte(`\`)); idx > 0 {
			start = idx
		}
		outBuf.Write(line[start:])
	}

	if err := outBuf.Flush(); err != nil {
		return nil, errors.Wrap(err, "Bufio error")
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
			fmt.Fprintf(output, "LONG $0x%02x%02x%02x%02x",
				objcode[3], objcode[2], objcode[1], objcode[0])
			objcode = objcode[4:]
			ln -= 4
		case ln >= 2:
			fmt.Fprintf(output, "WORD $0x%02x%02x", objcode[1], objcode[0])
			objcode = objcode[2:]
			ln -= 2
		default:
			fmt.Fprintf(output, "BYTE $0x%02x", objcode[0])
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
