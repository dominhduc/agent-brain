# Installing agent-brain

## For AI Agents (Copy-Paste Instructions)

If you're an AI coding agent and your user asked you to install agent-brain, follow these steps:

### 1. Clone and Build

```bash
git clone https://github.com/dominhduc/agent-brain.git /tmp/agent-brain
cd /tmp/agent-brain
make build
```

### 2. Install Binary

```bash
mkdir -p ~/.local/bin
cp bin/brain ~/.local/bin/brain
chmod +x ~/.local/bin/brain
```

### 3. Ensure `~/.local/bin` is in PATH

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc 2>/dev/null || true
export PATH="$HOME/.local/bin:$PATH"
```

### 4. Verify Installation

```bash
brain --help
```

### 5. Initialize in a Project

```bash
cd /path/to/project
brain init
```

### 6. Set API Key

```bash
brain config set llm.api_key sk-or-v1-...
```

Get your OpenRouter API key at [openrouter.ai](https://openrouter.ai).

### 7. Clean Up

```bash
rm -rf /tmp/agent-brain
```

## Requirements

- **Go 1.21+** (only needed for building from source)
- **Git** (must be installed and the project must be a git repo)
- **OpenRouter API key** (for the daemon to analyze commits)

## Troubleshooting

### `brain: command not found`

`~/.local/bin` is not in your PATH. Add it:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

### `GOPATH and GOROOT are the same directory`

Harmless warning. The build still works.

### `dubious ownership` error

```bash
git config --global --add safe.directory /path/to/project
```

### Build fails with Go version error

You need Go 1.21 or later. Check:

```bash
go version
```

## One-Liner Install (for humans)

```bash
git clone https://github.com/dominhduc/agent-brain.git /tmp/agent-brain && cd /tmp/agent-brain && make build && mkdir -p ~/.local/bin && cp bin/brain ~/.local/bin/ && chmod +x ~/.local/bin/brain && rm -rf /tmp/agent-brain && echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc && export PATH="$HOME/.local/bin:$PATH" && echo "Done! Run: brain --help"
```
