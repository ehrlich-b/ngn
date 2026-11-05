// ngnn5-convert adds a zero-weight sparse threat block to an NGNN3 network.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ehrlich-b/ngn/engine"
)

func convert(input, output string) error {
	src, err := os.Open(input)
	if err != nil {
		return err
	}
	defer src.Close()
	// Exclusive creation prevents accidental replacement of an existing net.
	dst, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	err = engine.ConvertNGNN3ToNGNN5(dst, src)
	closeErr := dst.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func main() {
	input := flag.String("in", "", "source NGNN3 file")
	output := flag.String("out", "", "new NGNN5 file")
	flag.Parse()
	if *input == "" || *output == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: ngnn5-convert -in source.nnue -out new.nnue")
		os.Exit(2)
	}
	if err := convert(*input, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
