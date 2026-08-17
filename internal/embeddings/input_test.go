package embeddings

import (
	"math"
	"strings"
	"testing"
)

func TestPrepareInputCleansMarkdown(t *testing.T) {
	longCode := strings.Repeat("x", 300)
	input := `---
title: ignored
---
Before ![useful diagram](diagram.png) and [read this](https://example.com).
<div>HTML <strong>content</strong></div>

` + "```mermaid\n" + longCode + "\n``` After"
	got := PrepareInput("Title", "Description", []string{"Go", "AI"}, input)
	for _, want := range []string{"Title Description Go, AI", "useful diagram", "read this", "HTML content", "After"} {
		if !strings.Contains(got, want) {
			t.Errorf("PrepareInput() = %q, missing %q", got, want)
		}
	}
	for _, unwanted := range []string{"diagram.png", "https://example.com", "<strong>", "title: ignored", strings.Repeat("x", 201)} {
		if strings.Contains(got, unwanted) {
			t.Errorf("PrepareInput() retained %q in %q", unwanted, got)
		}
	}
}

func TestHashInputIgnoresFormattingOnlyChanges(t *testing.T) {
	a := PrepareInput("Title", "Desc", []string{"tag"}, "Hello   [world](one)\n")
	b := PrepareInput("Title", "Desc", []string{"tag"}, "Hello world")
	if a != b {
		t.Fatalf("prepared inputs differ: %q != %q", a, b)
	}
	if HashInput(a, "model", 512) != HashInput(b, "model", 512) {
		t.Fatal("formatting-only change modified hash")
	}
	if HashInput(a, "model", 512) == HashInput(a, "other", 512) {
		t.Fatal("model must affect hash")
	}
	if HashInput(a, "model", 512) == HashInput(a, "model", 256) {
		t.Fatal("dimensions must affect hash")
	}
}

func TestQuantizeRoundTrip(t *testing.T) {
	input := make([]float32, 512)
	for i := range input {
		input[i] = float32(math.Sin(float64(i)*0.17) + 0.2*math.Cos(float64(i)))
	}
	encoded, scale, err := Quantize(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > 700 {
		t.Fatalf("encoded vector is %d bytes, want about 700 or less", len(encoded))
	}
	got, err := Dequantize(encoded, scale)
	if err != nil {
		t.Fatal(err)
	}
	want := Normalize(input)
	var squaredError float64
	for i := range want {
		delta := float64(want[i] - got[i])
		squaredError += delta * delta
	}
	rmse := math.Sqrt(squaredError / float64(len(want)))
	if rmse > 0.001 {
		t.Fatalf("quantization RMSE = %g, want <= 0.001", rmse)
	}
}

func TestMergeChunksAveragesAndNormalizes(t *testing.T) {
	got, err := MergeChunks([][]float32{{1, 0}, {0, 1}})
	if err != nil {
		t.Fatal(err)
	}
	want := float32(1 / math.Sqrt2)
	if math.Abs(float64(got[0]-want)) > 1e-6 || math.Abs(float64(got[1]-want)) > 1e-6 {
		t.Fatalf("MergeChunks() = %#v, want [%v %v]", got, want, want)
	}
}

func TestChunkTextUsesOverlap(t *testing.T) {
	chunks := ChunkText("one two three four five six seven", 15, 4)
	if len(chunks) < 2 {
		t.Fatalf("ChunkText() returned %d chunks", len(chunks))
	}
	if !strings.Contains(chunks[0], "three") || !strings.Contains(chunks[1], "three") {
		t.Fatalf("expected overlap around boundary, got %#v", chunks)
	}
}
