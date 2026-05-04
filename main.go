package main

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/pkoukk/tiktoken-go"
)

const (
	maxReadBytes            = 10 << 20
	defaultInputPricePer1M  = 1.00
	defaultOutputPricePer1M = 5.00
	defaultOutputTokens     = 0
)

//go:embed data/cl100k_base.tiktoken
var cl100kBase string

var ignoredDirs = map[string]bool{
	".git": true, ".hg": true, ".svn": true,
	"node_modules": true, "vendor": true,
	"dist": true, "build": true, "target": true,
	".next": true, ".nuxt": true, ".cache": true,
	".idea": true, ".vscode": true,
	"coverage": true, "__pycache__": true,
}

var ignoredExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true, ".ico": true,
	".pdf": true, ".zip": true, ".gz": true, ".tar": true, ".rar": true, ".7z": true,
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".mp3": true, ".mp4": true, ".mov": true, ".avi": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true,
	".lock":     true,
	".tiktoken": true, ".gitignore": true, ".dockerignore": true,
}

type fileStat struct {
	Path   string `json:"path"`
	Lines  int    `json:"lines"`
	Tokens int    `json:"tokens"`
	Bytes  int64  `json:"bytes"`
}

type folderStat struct {
	Path   string `json:"path"`
	Files  int    `json:"files"`
	Lines  int    `json:"lines"`
	Tokens int    `json:"tokens"`
}

type report struct {
	Path                string       `json:"path"`
	Encoding            string       `json:"encoding"`
	Files               int          `json:"files"`
	Lines               int          `json:"lines"`
	Tokens              int          `json:"tokens"`
	OutputTokens        int          `json:"output_tokens"`
	InputPricePer1MUSD  float64      `json:"input_price_per_1m_usd"`
	OutputPricePer1MUSD float64      `json:"output_price_per_1m_usd"`
	InputCostUSD        float64      `json:"input_cost_usd"`
	OutputCostUSD       float64      `json:"output_cost_usd"`
	TotalCostUSD        float64      `json:"total_cost_usd"`
	ArchitectureWeight  string       `json:"architecture_weight"`
	TopFiles            []fileStat   `json:"top_files"`
	Folders             []folderStat `json:"folders"`
	FilesDetail         []fileStat   `json:"files_detail,omitempty"`
}

func main() {
	jsonOut := flag.Bool("json", false, "print JSON report")
	top := flag.Int("top", 10, "heavy files to show")
	price := flag.Float64("price", defaultInputPricePer1M, "same as --input-price")
	inputPrice := flag.Float64("input-price", defaultInputPricePer1M, "USD per 1M input tokens")
	outputPrice := flag.Float64("output-price", defaultOutputPricePer1M, "USD per 1M output tokens")
	outputTokens := flag.Int("output-tokens", defaultOutputTokens, "output tokens to include in cost")
	encoding := flag.String("encoding", "cl100k_base", "token encoding")
	flag.Parse()

	if flag.NArg() != 1 {
		fail("usage: token-meter [--json] [--top 10] [--input-price 1] [--output-price 5] [--output-tokens 0] <file|folder>")
	}
	if *top < 0 {
		fail("--top must be 0 or higher")
	}
	if *outputTokens < 0 {
		fail("--output-tokens must be 0 or higher")
	}
	if *inputPrice < 0 || *outputPrice < 0 || *price < 0 {
		fail("prices must be 0 or higher")
	}
	if *encoding != "cl100k_base" {
		fail(fmt.Sprintf("unsupported encoding %q: only \"cl100k_base\" is available", *encoding))
	}
	if isFlagSet("price") && !isFlagSet("input-price") {
		*inputPrice = *price
	}

	tiktoken.SetBpeLoader(localBPELoader{})

	rep, err := scan(flag.Arg(0), *encoding, *inputPrice, *outputPrice, *outputTokens, *top)
	if err != nil {
		fail(err.Error())
	}

	if *jsonOut {
		printJSON(rep)
		return
	}
	printReport(rep)
}

func scan(path, encodingName string, inputPrice, outputPrice float64, outputTokens, top int) (report, error) {
	root, err := filepath.Abs(path)
	if err != nil {
		return report{}, err
	}

	info, err := os.Stat(root)
	if err != nil {
		return report{}, err
	}

	enc, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		return report{}, fmt.Errorf("tokenizer %q failed: %w", encodingName, err)
	}

	var files []fileStat
	if info.IsDir() {
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if path != root && ignoredDirs[entry.Name()] {
					return filepath.SkipDir
				}
				return nil
			}
			stat, ok, err := countFile(root, path, enc)
			if err != nil || !ok {
				return err
			}
			files = append(files, stat)
			return nil
		})
	} else {
		var ok bool
		var stat fileStat
		stat, ok, err = countFile(filepath.Dir(root), root, enc)
		if ok {
			files = append(files, stat)
		}
	}
	if err != nil {
		return report{}, err
	}
	if len(files) == 0 {
		return report{}, errors.New("no readable text files found")
	}

	rep := buildReport(root, encodingName, inputPrice, outputPrice, outputTokens, top, files)
	return rep, nil
}

func countFile(root, path string, enc *tiktoken.Tiktoken) (fileStat, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileStat{}, false, err
	}
	if info.Size() > maxReadBytes || ignoredExt[strings.ToLower(filepath.Ext(path))] {
		return fileStat{}, false, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fileStat{}, false, err
	}
	if isBinary(data) {
		return fileStat{}, false, nil
	}

	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = path
	}
	text := string(data)
	return fileStat{
		Path:   filepath.ToSlash(rel),
		Lines:  countLines(text),
		Tokens: len(enc.Encode(text, nil, nil)),
		Bytes:  info.Size(),
	}, true, nil
}

func buildReport(root, encoding string, inputPrice, outputPrice float64, outputTokens, top int, files []fileStat) report {
	sort.Slice(files, func(i, j int) bool {
		if files[i].Tokens == files[j].Tokens {
			return files[i].Path < files[j].Path
		}
		return files[i].Tokens > files[j].Tokens
	})

	folders := map[string]*folderStat{}
	var lines, tokens int
	for _, file := range files {
		lines += file.Lines
		tokens += file.Tokens
		dir := filepath.ToSlash(filepath.Dir(file.Path))
		if dir == "." {
			dir = "/"
		}
		for {
			folder := folders[dir]
			if folder == nil {
				folder = &folderStat{Path: dir}
				folders[dir] = folder
			}
			folder.Files++
			folder.Lines += file.Lines
			folder.Tokens += file.Tokens
			if dir == "/" || !strings.Contains(dir, "/") {
				break
			}
			dir = filepath.ToSlash(filepath.Dir(dir))
		}
	}

	folderList := make([]folderStat, 0, len(folders))
	for _, folder := range folders {
		folderList = append(folderList, *folder)
	}
	sort.Slice(folderList, func(i, j int) bool {
		if folderList[i].Tokens == folderList[j].Tokens {
			return folderList[i].Path < folderList[j].Path
		}
		return folderList[i].Tokens > folderList[j].Tokens
	})

	topFiles := min(top, len(files))
	inputCost := cost(tokens, inputPrice)
	outputCost := cost(outputTokens, outputPrice)
	return report{
		Path:                root,
		Encoding:            encoding,
		Files:               len(files),
		Lines:               lines,
		Tokens:              tokens,
		OutputTokens:        outputTokens,
		InputPricePer1MUSD:  inputPrice,
		OutputPricePer1MUSD: outputPrice,
		InputCostUSD:        inputCost,
		OutputCostUSD:       outputCost,
		TotalCostUSD:        inputCost + outputCost,
		ArchitectureWeight:  architectureWeight(tokens, len(files), len(folderList)),
		TopFiles:            files[:topFiles],
		Folders:             folderList,
		FilesDetail:         files,
	}
}

func isBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	sample := data[:min(len(data), 8000)]
	if !utf8.Valid(sample) {
		return true
	}
	for _, b := range sample {
		if b == 0 {
			return true
		}
	}
	return false
}

func countLines(text string) int {
	if text == "" {
		return 0
	}
	lines := strings.Count(text, "\n")
	if !strings.HasSuffix(text, "\n") {
		lines++
	}
	return lines
}

func architectureWeight(tokens, files, folders int) string {
	switch {
	case tokens > 200_000 || files > 400 || folders > 80:
		return "heavy"
	case tokens > 60_000 || files > 120 || folders > 30:
		return "medium"
	default:
		return "light"
	}
}

func printReport(rep report) {
	c := colors()
	fmt.Printf("%sTokenMeter report%s\n", c.bold, c.reset)
	fmt.Printf("%s%s%s\n\n", c.dim, rep.Path, c.reset)

	printSection("Summary", c)
	printRow("Encoding", rep.Encoding, c)
	printRow("Files", fmt.Sprintf("%d", rep.Files), c)
	printRow("Lines", fmt.Sprintf("%d", rep.Lines), c)
	printRow("Tokens", fmt.Sprintf("%d", rep.Tokens), c)
	printRow("Architecture weight", rep.ArchitectureWeight, c)

	printSection("Cost estimate", c)
	printRow("Input tokens", fmt.Sprintf("%d", rep.Tokens), c)
	printRow("Output tokens", fmt.Sprintf("%d", rep.OutputTokens), c)
	printRow("Input price", fmt.Sprintf("$%.2f / 1M tokens", rep.InputPricePer1MUSD), c)
	printRow("Output price", fmt.Sprintf("$%.2f / 1M tokens", rep.OutputPricePer1MUSD), c)
	printRow("Input cost", fmt.Sprintf("$%.4f", rep.InputCostUSD), c)
	printRow("Output cost", fmt.Sprintf("$%.4f", rep.OutputCostUSD), c)
	printRow("Total cost", fmt.Sprintf("$%.4f", rep.TotalCostUSD), c)
	fmt.Printf("%sValues are estimates and may change with provider pricing, tokenizer, model, cache, and output size.%s\n", c.dim, c.reset)

	printSection("Heavy files", c)
	for _, file := range rep.TopFiles {
		fmt.Printf("  %s%8d%s tokens  %6d lines  %s\n", c.green, file.Tokens, c.reset, file.Lines, file.Path)
	}

	printSection("Heavy folders", c)
	for _, folder := range rep.Folders {
		fmt.Printf("  %s%8d%s tokens  %6d lines  %4d files  %s\n", c.green, folder.Tokens, c.reset, folder.Lines, folder.Files, folder.Path)
	}
}

func printJSON(rep report) {
	out, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fail(err.Error())
	}
	fmt.Println(string(out))
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "error:", message)
	os.Exit(1)
}

type terminalColors struct {
	reset string
	bold  string
	dim   string
	blue  string
	green string
}

func colors() terminalColors {
	if os.Getenv("NO_COLOR") != "" {
		return terminalColors{}
	}
	if runtime.GOOS == "windows" && os.Getenv("TOKENMETER_COLOR") == "" {
		return terminalColors{}
	}
	return terminalColors{
		reset: "\033[0m",
		bold:  "\033[1m",
		dim:   "\033[2m",
		blue:  "\033[34m",
		green: "\033[32m",
	}
}

func printSection(title string, c terminalColors) {
	fmt.Printf("\n%s%s%s\n", c.blue, title, c.reset)
}

func printRow(label, value string, c terminalColors) {
	fmt.Printf("  %-20s %s%s%s\n", label+":", c.bold, value, c.reset)
}

func cost(tokens int, pricePer1M float64) float64 {
	return float64(tokens) / 1_000_000 * pricePer1M
}

func isFlagSet(name string) bool {
	set := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}

type localBPELoader struct{}

func (localBPELoader) LoadTiktokenBpe(path string) (map[string]int, error) {
	if !strings.Contains(path, "cl100k_base.tiktoken") {
		return nil, fmt.Errorf("no local tokenizer data for %s", path)
	}
	ranks := make(map[string]int)
	for _, line := range strings.Split(cl100kBase, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("bad tokenizer line %q", line)
		}
		token, err := base64.StdEncoding.DecodeString(parts[0])
		if err != nil {
			return nil, err
		}
		rank, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, err
		}
		ranks[string(token)] = rank
	}
	return ranks, nil
}
