# Contributing to grít

## Prerequisites

- Go 1.21+
- GCC (required for CGO / SQLite)
  - Windows: MSYS2 — `pacman -S mingw-w64-ucrt-x86_64-gcc`
  - macOS: `xcode-select --install`
  - Linux: `sudo apt install build-essential`

## Build from source

```sh
# macOS / Linux
go build -o grit .

# Windows (MSYS2 gcc)
PATH="/c/msys64/ucrt64/bin:$PATH" go build -o grit.exe .
```

## Running locally

```sh
cd /tmp/test-repo && git init
/path/to/grit init
git add . && git commit -m "test"
```

## Project structure

```
cmd/        cobra commands (one file per command)
internal/
  analysis/ complexity scoring and naming heuristics
  config/   viper config and path helpers
  hooks/    git hook installer
  prompt/   bubbletea TUI prompts
  store/    SQLite schema, events, answers
```

## Submitting changes

1. Fork the repo and create a branch off `main`
2. Keep commits focused — one logical change per commit
3. Test manually: `grit init`, make a commit, verify the hook fires
4. Open a pull request with a clear description of what and why

## Reporting bugs

Open an issue at https://github.com/AlchemistReturns/grit/issues with:
- OS and Go version
- Steps to reproduce
- Expected vs. actual behavior

## License

By contributing you agree your changes are licensed under the [MIT License](LICENSE).
