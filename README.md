# mavis

unconventional commit

<img alt="mavis - unconventional commit" src="demo.gif" width="800" />

## About

Mavis is an interactive terminal UI tool for creating structured Git commits. It provides a split-screen interface with customizable form inputs on the left and a live preview of your commit message on the right.

### Key Features

- **Interactive UI**: Split-screen with form inputs and live commit message preview
- **AI-Powered Suggestions**: Generate commit message field defaults using OpenAI
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

### Keyboard Shortcuts

- **Ctrl+A / Ctrl+S**: Accept preview and commit
- **?**: Toggle help view
- **Esc / Ctrl+C**: Quit without committing

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

### AI-Powered Commits

Mavis can generate intelligent commit message suggestions using OpenAI's GPT models. When enabled, it analyzes your git diff and suggests appropriate values for your commit message fields.

#### Setup

1. **Get an OpenAI API key**: Sign up at [OpenAI](https://platform.openai.com/) and create an API key
2. **Set the environment variable**:
   ```bash
   export OPENAI_API_KEY="your-api-key-here"
   ```
3. **Run mavis with --ai or enable it in your config** - AI suggestions will be automatically generated based on your staged changes

#### Customization

You can customize the AI behavior in your configuration file:

```yaml
ai:
  enabled: true
  provider: "openai"
  custom_prompt: "Focus on business impact and use imperative mood."
  openai:
    model: "gpt-4.1-mini"
    max_completion_tokens: 500
    temperature: 0.2
```

#### How It Works

- Analyzes your staged git changes (diff)
- Considers the current branch name for better context
- Uses GPT-4 Mini for cost-effective, accurate suggestions
- Generates default values for all configured commit message fields
- Respects your existing field configuration and templates
- Works with any custom fields you've defined in your config
- Designed to support multiple AI providers (currently supports OpenAI)
