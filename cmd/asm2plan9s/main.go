package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/Kunde21/asm2plan9s"
	"github.com/klauspost/asmfmt"
)

var (
	help  = flag.Bool("h", false, "Print usage instructions")
	write = flag.Bool("w", false, "Write changes to file")
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: asmfmt [flags] [path ...]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	fmt.Println(os.Args)

	flag.Parse()
	if *help {
		usage()
	}

	exitCode := 0
	files := []string{}
	for _, g := range os.Args[1:] {
		f, err := filepath.Glob(g)
		if err != nil {
			log.Println(err)
			exitCode = 2
		}
		if f != nil {
			files = append(files, f...)
		}
	}
	if len(files) == 0 {
		log.Println("No files processed.")
		exitCode = 2
	}

	for _, file := range files {
		fmt.Fprintln(os.Stderr, "Processing ", file)
		source, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ReadFile %s: %s", file, err)
			exitCode = 2
			continue
		}

		inBuf := bytes.NewReader(source)

		result, err := asm2plan9s.Assemble("// @", inBuf)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			exitCode = 2
			continue
		}

		result, err = asmfmt.Format(bytes.NewReader(result))
		if err != nil {
			fmt.Fprintf(os.Stderr, "asmfmt error %s: %s", file, err)
			exitCode = 2
			continue
		}

		result, err = asmfmt.Format(bytes.NewReader(result))
		if err != nil {
			log.Fatalf("asmfmt error: %s", err)
			exitCode = 2
		}

		if *write {
			err = ioutil.WriteFile(file, result, 0600)
			if err != nil {
				fmt.Fprintf(os.Stderr, "WriteFile %s: %s", file, err)
				exitCode = 2
			}
		} else {
			fmt.Println(string(result))
		}
	}
	os.Exit(exitCode)
}
