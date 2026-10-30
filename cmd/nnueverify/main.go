// Command nnueverify checks a loss image and an NGN-v1 network.
package main

import (
	"fmt"
	"image/png"
	"os"

	"github.com/ehrlich-b/ngn/nnue"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: nnueverify loss.png network.nnue")
		os.Exit(2)
	}
	if err := verify(os.Args[1], os.Args[2]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("decoded loss PNG and loaded NGN-v1 network")
}

func verify(imagePath, modelPath string) error {
	imageFile, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer imageFile.Close()
	if _, err := png.Decode(imageFile); err != nil {
		return fmt.Errorf("loss PNG: %w", err)
	}
	modelFile, err := os.Open(modelPath)
	if err != nil {
		return err
	}
	defer modelFile.Close()
	if _, err := nnue.Load(modelFile); err != nil {
		return fmt.Errorf("network: %w", err)
	}
	return nil
}
