package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"bufio"
	"os"
	"os/exec"
	"strings"
)

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


func yasm(instr string) (string, error) {

	instrFields := strings.Split(instr, "/*")
	content := []byte("[bits 64]\n" + instrFields[0])
	tmpfile, err := ioutil.TempFile("", "asm2plan9s")
	if err != nil {
		return "", nil
	}

	if _, err := tmpfile.Write(content); err != nil {
		return "", nil
	}
	if err := tmpfile.Close(); err != nil {
		return "", nil
	}

	asmFile := tmpfile.Name() + ".asm"
	objFile := tmpfile.Name() + ".obj"
	os.Rename( tmpfile.Name(), asmFile)

	defer os.Remove(asmFile) // clean up
	defer os.Remove(objFile) // clean up

	app := "yasm"

	arg0 := "-o"
	arg1 := objFile
	arg2 := asmFile

	cmd := exec.Command(app, arg0, arg1, arg2)
	_, err = cmd.Output()

	if err != nil {
		return "", nil
	}

	return toPlan9s(objFile, instr)
}

func toPlan9s(objFile, instr string) (string, error) {
	objcode, err := ioutil.ReadFile(objFile)
	if err != nil {
		return "", err
	}

	sline := "    "
	for i, b := range objcode {
		if i != 0 {
			sline += "; "
		}
		sline += fmt.Sprintf("BYTE $0x%02x", b)
	}

	if len(sline) < 65 {
		sline += strings.Repeat(" ", 65 - len(sline))
	}

	sline += "//" + instr

	return sline, nil
}

// readLines reads a whole file into memory
// and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
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

func main() {

	if len(os.Args) < 2 {
		fmt.Printf("error: no input specified\n\n")
		fmt.Println("usage: asm2plan9s file")
		fmt.Println("  will in-place update the assembly file with proper BYTE sequence as generated by YASM")
		return
	}
	fmt.Println(os.Args[1])
	lines, err := readLines(os.Args[1])
	if err != nil {
		log.Fatalf("readLines: %s", err)
	}

	var result []string
	for _, line := range lines {
		line := strings.Replace(line, "\t", "    ", -1)
		fields := strings.Split(line, "//")
		if len(fields[0]) == 65 && len(fields) == 2 {
			sline, err := yasm(fields[1])
			if err != nil {
				log.Fatalf("yasm(%s): %s", line, err)
			}
			fmt.Println(sline)
			result = append(result, sline)
		} else {
			result = append(result, line)
		}
	}

	err = writeLines(result, os.Args[1])
	if err != nil {
		log.Fatalf("writeLines: %s", err)
	}
}