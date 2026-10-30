package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/ehrlich-b/ngn/nnue"
)

func TestVerifyRejectsIncompleteAndCorruptArtifacts(t *testing.T) {
	var imageData bytes.Buffer
	if err := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	model, err := nnue.Marshal(&nnue.Tensors{})
	if err != nil {
		t.Fatal(err)
	}
	corrupt := append([]byte(nil), model...)
	corrupt[len(corrupt)-1] ^= 1
	for _, tc := range []struct {
		name         string
		image, model []byte
		valid        bool
	}{
		{"valid", imageData.Bytes(), model, true},
		{"png signature only", imageData.Bytes()[:8], model, false},
		{"truncated png", imageData.Bytes()[:imageData.Len()-1], model, false},
		{"model magic only", imageData.Bytes(), model[:8], false},
		{"corrupt model payload", imageData.Bytes(), corrupt, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			imagePath, modelPath := filepath.Join(dir, "loss.png"), filepath.Join(dir, "net.nnue")
			if err := os.WriteFile(imagePath, tc.image, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(modelPath, tc.model, 0600); err != nil {
				t.Fatal(err)
			}
			if err := verify(imagePath, modelPath); (err == nil) != tc.valid {
				t.Fatalf("verify returned %v, valid=%v", err, tc.valid)
			}
		})
	}
}
