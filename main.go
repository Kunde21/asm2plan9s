package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/klauspost/asmfmt"
)

var (
	help = flag.Bool("h", false, "Print usage instructions")
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: asmfmt [flags] [path ...]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {

	flag.Parse()
	if *help {
		usage()
	}

	if len(os.Args) == 1 {
		result, err := Assemble(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		result, err = asmfmt.Format(bytes.NewReader(result))
		if err != nil {
			log.Fatalf("asmfmt error: %s", err)
		}
		_, err = os.Stdout.Write(result)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	files := []string{}
	for _, g := range os.Args[1:] {
		f, err := filepath.Glob(g)
		if err != nil {
			log.Println(err)
		}
		if f != nil {
			files = append(files, f...)
		}
	}
	if len(files) == 0 {
		log.Println("No files processed.")
	}

	for _, file := range files {
		fmt.Println("Processing ", file)
		source, err := ioutil.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ReadFile %s: %s", file, err)
			continue
		}

		inBuf := bytes.NewReader(source)

		result, err := Assemble(inBuf)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}

		result, err = asmfmt.Format(bytes.NewReader(result))
		if err != nil {
			fmt.Fprintf(os.Stderr, "asmfmt error %s: %s", file, err)
			continue
		}

		err = ioutil.WriteFile(os.Args[1], result, 0600)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WriteFile %s: %s", file, err)
		}
	}
}
