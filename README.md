# TokenMeter

[![Release](https://img.shields.io/github/v/release/otechmista/token-meter?sort=semver)](https://github.com/otechmista/token-meter/releases/tag/v0.1.0)
[![Release workflow](https://github.com/otechmista/token-meter/actions/workflows/release.yml/badge.svg)](https://github.com/otechmista/token-meter/actions/workflows/release.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/otechmista/token-meter)](https://goreportcard.com/report/github.com/otechmista/token-meter)
[![Go version](https://img.shields.io/github/go-mod/go-version/otechmista/token-meter)](https://github.com/otechmista/token-meter/blob/main/go.mod)
[![Platforms](https://img.shields.io/badge/platform-Windows%20%7C%20Linux-blue)](#install)

TokenMeter is a local CLI for measuring how much token weight a codebase carries. It scans a file or folder, tokenizes source files with the embedded `cl100k_base` tokenizer, and reports totals, hotspots, folder weight, and estimated model input/output cost.

It is designed for developers who want a quick, private way to understand how expensive a repository, feature folder, prompt bundle, or single file may be before sending it to an LLM.

## Highlights

- Runs fully locally and does not upload your code.
- Supports files and folders.
- Uses the embedded `cl100k_base` tokenizer.
- Shows total tokens, files, lines, folder totals, and heaviest files.
- Estimates input and output cost using configurable prices.
- Exports JSON for scripts and CI workflows.
- Ignores common binary assets, build outputs, dependency folders, and generated noise.
- Ships install scripts for Windows PowerShell and Linux.

## Install

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/otechmista/token-meter/main/install/install.ps1 | iex
```

### Linux/Ubuntu

```sh
curl -fsSL https://raw.githubusercontent.com/otechmista/token-meter/main/install/install.sh | sh
```

Or with `wget`:

```sh
wget -qO- https://raw.githubusercontent.com/otechmista/token-meter/main/install/install.sh | sh
```

Open a new terminal after installation, then run:

```sh
tkm .
```

To update TokenMeter, run the same install command again. The installer downloads the latest GitHub Release and replaces the previous binary.

## Quick Start

Measure the current directory:

```sh
tkm .
```

Show the 20 heaviest files:

```sh
tkm --top 20 .
```

Estimate cost with custom input and output prices:

```sh
tkm --input-price 1 --output-price 5 --output-tokens 1000 .
```

Export a machine-readable report:

```sh
tkm --json .
```

Measure a single file:

```sh
tkm main.go
```

Measure another project:

```sh
tkm /path/to/project
```

## Usage

```text
tkm [--json] [--top 10] [--input-price 1] [--output-price 5] [--output-tokens 0] [--encoding cl100k_base] <file|folder>
```

When running a locally built binary from this repository:

```sh
bin/tkm .
```

On Windows:

```powershell
.\bin\tkm.exe .
```

If you are inside the `bin` folder:

```powershell
.\tkm.exe .
```

## Options

| Option | Default | Description |
| --- | ---: | --- |
| `--top <n>` | `10` | Number of heaviest files to show. Use `0` to hide the list. |
| `--input-price <usd>` | `1` | Price in USD per 1 million input tokens. |
| `--output-price <usd>` | `5` | Price in USD per 1 million output tokens. |
| `--output-tokens <n>` | `0` | Output tokens to include in the cost estimate. |
| `--price <usd>` | `1` | Legacy shortcut for `--input-price`. |
| `--json` | `false` | Print the report as JSON. |
| `--encoding <name>` | `cl100k_base` | Tokenizer to use. Only `cl100k_base` is currently supported. |

## Report

TokenMeter reports:

- total tokens
- total files
- total lines
- tokens per file
- tokens per folder
- heaviest files
- estimated input, output, and total cost
- architecture weight

JSON output includes the same information with stable field names such as `tokens`, `top_files`, `folders`, `input_cost_usd`, `output_cost_usd`, and `total_cost_usd`.

## Terminal Colors

On Windows, output is colorless by default to avoid broken ANSI codes in PowerShell.

Force color:

```powershell
$env:TOKENMETER_COLOR=1
tkm .
```

Disable color on any system:

```sh
NO_COLOR=1 tkm .
```

## What Is Ignored

TokenMeter skips common files that are not useful for code token analysis:

- binaries
- images
- videos
- fonts
- compressed files
- folders such as `.git`, `node_modules`, `vendor`, `dist`, `build`, and `target`

## Build From Source

### Requirements

- Go 1.25.5 or newer
- Git
- Make optional

### Clone

```sh
git clone https://github.com/otechmista/token-meter.git
cd token-meter
```

### Build

With Make:

```sh
make build
```

Without Make:

```sh
go build -o bin/tkm .
```

On Windows:

```powershell
go build -o bin\tkm.exe .
```

The executable is created at:

```text
bin/tkm
```

On Windows:

```text
bin/tkm.exe
```

## Development

Run tests:

```sh
go test ./...
```

Run against the current repository:

```sh
go run . .
```

Generate local distribution binaries:

```sh
make dist
```

This creates:

```text
dist/tkm-windows-amd64.exe
dist/tkm-linux-amd64
```

## Release

Create and push a version tag:

```sh
git tag v0.1.0
git push origin v0.1.0
```

GitHub Actions will:

- run tests
- build `tkm-windows-amd64.exe` for Windows
- build `tkm-linux-amd64` for Linux
- generate `checksums.txt`
- publish the files to a GitHub Release

## How It Works

```text
path -> scan -> tokenize -> report
```

TokenMeter walks the target path, filters ignored files and directories, tokenizes text content with `cl100k_base`, aggregates file and folder totals, then prints a terminal or JSON report.
