package main

import "testing"

func TestCountLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single line no newline", "hello", 1},
		{"single line with newline", "hello\n", 1},
		{"multiple lines", "a\nb\nc\n", 3},
		{"multiple lines no trailing newline", "a\nb\nc", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := countLines(tt.input); got != tt.want {
				t.Errorf("countLines(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsBinary(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  bool
	}{
		{"empty", []byte{}, false},
		{"ascii text", []byte("hello world"), false},
		{"utf8 text", []byte("olá mundo"), false},
		{"null byte", []byte("hello\x00world"), true},
		{"invalid utf8", []byte{0xc3, 0x28, 0x41}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBinary(tt.input); got != tt.want {
				t.Errorf("isBinary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestArchitectureWeight(t *testing.T) {
	tests := []struct {
		name    string
		tokens  int
		files   int
		folders int
		want    string
	}{
		{"light", 1_000, 10, 5, "light"},
		{"medium by tokens", 100_000, 10, 5, "medium"},
		{"medium by files", 1_000, 150, 5, "medium"},
		{"medium by folders", 1_000, 10, 40, "medium"},
		{"heavy by tokens", 250_000, 10, 5, "heavy"},
		{"heavy by files", 1_000, 500, 5, "heavy"},
		{"heavy by folders", 1_000, 10, 90, "heavy"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := architectureWeight(tt.tokens, tt.files, tt.folders); got != tt.want {
				t.Errorf("architectureWeight(%d, %d, %d) = %q, want %q", tt.tokens, tt.files, tt.folders, got, tt.want)
			}
		})
	}
}

func TestCost(t *testing.T) {
	tests := []struct {
		tokens int
		price  float64
		want   float64
	}{
		{0, 1.0, 0.0},
		{1_000_000, 1.0, 1.0},
		{500_000, 2.0, 1.0},
		{1_000_000, 5.0, 5.0},
	}
	for _, tt := range tests {
		if got := cost(tt.tokens, tt.price); got != tt.want {
			t.Errorf("cost(%d, %g) = %g, want %g", tt.tokens, tt.price, got, tt.want)
		}
	}
}

func TestBuildReport(t *testing.T) {
	files := []fileStat{
		{Path: "main.go", Lines: 100, Tokens: 500, Bytes: 2000},
		{Path: "util.go", Lines: 50, Tokens: 200, Bytes: 800},
		{Path: "cmd/run.go", Lines: 30, Tokens: 100, Bytes: 300},
	}
	rep := buildReport("/project", "cl100k_base", 1.0, 5.0, 0, 10, files)

	if rep.Files != 3 {
		t.Errorf("Files = %d, want 3", rep.Files)
	}
	if rep.Lines != 180 {
		t.Errorf("Lines = %d, want 180", rep.Lines)
	}
	if rep.Tokens != 800 {
		t.Errorf("Tokens = %d, want 800", rep.Tokens)
	}
	if rep.ArchitectureWeight != "light" {
		t.Errorf("ArchitectureWeight = %q, want \"light\"", rep.ArchitectureWeight)
	}
	if len(rep.TopFiles) != 3 {
		t.Errorf("len(TopFiles) = %d, want 3", len(rep.TopFiles))
	}
	if rep.TopFiles[0].Path != "main.go" {
		t.Errorf("TopFiles[0].Path = %q, want \"main.go\" (sorted by tokens desc)", rep.TopFiles[0].Path)
	}
	if rep.TotalCostUSD != rep.InputCostUSD+rep.OutputCostUSD {
		t.Errorf("TotalCostUSD mismatch: %g != %g + %g", rep.TotalCostUSD, rep.InputCostUSD, rep.OutputCostUSD)
	}
}
