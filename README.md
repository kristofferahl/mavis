# mavis

unconventional commit

<img alt="mavis - unconventional commit" src="demo.gif" width="800" />

## About

Mavis is an interactive terminal UI tool for creating structured Git commits. It provides a split-screen interface with customizable form inputs on the left and a live preview of your commit message on the right.

### Key Features

- **Interactive UI**: Split-screen with form inputs and live commit message preview
- **Customizable Templates**: Format commit messages using a flexible template system
- **Configurable Fields**: Define the structure of your commit messages
- **Multiple Themes**: Choose from themes like charm, dracula, catppuccin, and more
- **Configuration System**: YAML-based with support for conditional configurations
- **Environment Variable Overrides**: Customize behavior via environment variables

## Install

### Homebrew tap

```console
brew install kristofferahl/tap/mavis
```

### go install

```console
go install github.com/kristofferahl/mavis@latest
```

### Manual

Download binaries from [release page](https://github.com/kristofferahl/mavis/releases)

## Usage

Simply run `mavis` in your Git repository to start the interactive commit process:

```console
mavis
```

### Configuration

Mavis automatically creates a default configuration file at `~/.config/mavis/config.yaml` on first run. You can customize this file to change themes, fields, and commit message templates.

#### Environment Variables

- `MAVIS_THEME`: Override the theme (e.g., "charm", "dracula", "catppuccin")
- `MAVIS_CHIP`: Override the chip label shown in the UI

#### Debug Mode

Run with the debug flag to see additional information:

```console
mavis --debug
```
