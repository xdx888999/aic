# aic

[中文](./README.md) | [English](./README.en.md)

`aic` is a terminal-based AI CLI tool manager.

It helps you view the install status, current version, latest version, and update source of common AI CLI tools in one place, and lets you upgrade tools or open their config files from the same interface.

## Preview

English UI:

![aic English UI](./assets/aic-terminal.png)

Chinese UI:

![aic Chinese UI](./assets/aic-terminal-zh.png)

## Core Features

- Scan common AI CLI tools on the local machine
- Show install status, current version, latest version, and update source
- Hide uninstalled tools by default, with an option to show all candidates
- Upgrade the selected tool directly from the TUI
- Open the selected tool's config file directly

## Common Tools

The project currently focuses on common tools such as:

- `Claude Code`
- `Codex CLI`
- `Gemini CLI`
- `Aider`
- `Qwen Code`
- `Kimi Code`
- `Cursor`
- `Windsurf`
- `Trae Agent`

This README lists common tools only and does not claim full coverage.

## Command Entry

The intended command entry is:

```bash
aic
```

Whether you use it yourself or distribute it to others, the final entry point should be the `aic` command.

## Installation

### Option 1: Install with Homebrew

```bash
brew tap xdx888999/homebrew-tap && brew install xdx888999/homebrew-tap/aic
```

### Option 2: Download from GitHub Releases

Release page:

- <https://github.com/xdx888999/aic/releases>

Choose the archive for your platform:

- macOS Apple Silicon: `aic_darwin_arm64.tar.gz`
- macOS Intel: `aic_darwin_amd64.tar.gz`
- Linux amd64: `aic_linux_amd64.tar.gz`
- Linux arm64: `aic_linux_arm64.tar.gz`

After extracting it, move `aic` into your `PATH`, for example:

```bash
mv aic /usr/local/bin/aic
```

### Option 3: Build from Source

```bash
go build -o aic . && mv aic /usr/local/bin/aic
```

If you prefer a user-local directory, you can also place it in:

```bash
~/bin/aic
```

Make sure that directory is already in your `PATH`.

## Updating aic

### Homebrew users

```bash
brew update && brew upgrade aic
```

### Manual install users

- Download the new version from GitHub Releases
- Replace the existing `aic` binary after extraction

## Common Keys

- `↑ / ↓` or `j / k`: move cursor
- `u`: upgrade the current tool
- `c`: open the config file
- `a`: show or hide uninstalled tools
- `l`: switch language
- `r`: rescan
- `q`: quit

## Development

Run locally:

```bash
go run .
```

Run tests:

```bash
go test ./...
```

## Credit

`xdx_lab`

- X: <https://x.com/terry13O>
- GitHub: <https://github.com/xdx888999>
