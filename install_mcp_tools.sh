#!/bin/bash
set -e

# Create central agent directory
AGENT_DIR="$HOME/.agent"
mkdir -p "$AGENT_DIR"
echo "Created $AGENT_DIR"

# 1. Get Shit Done (GSD)
if [ ! -d "$AGENT_DIR/get-shit-done" ]; then
    echo "Cloning Get Shit Done..."
    git clone https://github.com/gsd-build/get-shit-done.git "$AGENT_DIR/get-shit-done"
else
    echo "Get Shit Done already installed."
fi

# 2. Ralph
if [ ! -d "$AGENT_DIR/ralph" ]; then
    echo "Cloning Ralph..."
    git clone https://github.com/snarktank/ralph.git "$AGENT_DIR/ralph"
    chmod +x "$AGENT_DIR/ralph/scripts/ralph/ralph.sh" || true
else
    echo "Ralph already installed."
fi

# 3. Antigravity Awesome Skills
if [ ! -d "$AGENT_DIR/skills" ]; then
    echo "Cloning Antigravity Awesome Skills..."
    git clone https://github.com/sickn33/antigravity-awesome-skills.git "$AGENT_DIR/skills"
else
    echo "Antigravity Awesome Skills already installed."
fi

# 4. Hive
if [ ! -d "$AGENT_DIR/hive" ]; then
    echo "Cloning Hive..."
    git clone https://github.com/adenhq/hive.git "$AGENT_DIR/hive"
else
    echo "Hive already installed."
fi

# 5. Superpowers
if [ ! -d "$AGENT_DIR/superpowers" ]; then
    echo "Cloning Superpowers..."
    git clone https://github.com/obra/superpowers.git "$AGENT_DIR/superpowers"
else
    echo "Superpowers already installed."
fi

# Hive Setup
echo "Setting up Hive..."
cd "$AGENT_DIR/hive"
if command -v uv &> /dev/null; then
    echo "Syncing Hive dependencies..."
    uv sync
    echo "Installing Playwright browsers..."
    uv run python -m playwright install chromium
else
    echo "uv not found, skipping Hive dependency setup. Please install uv."
fi

# Create default Hive config
mkdir -p "$HOME/.hive"
if [ ! -f "$HOME/.hive/configuration.json" ]; then
    echo "Creating default Hive configuration..."
    cat > "$HOME/.hive/configuration.json" <<EOF
{
  "llm": {
    "provider": "anthropic",
    "model": "claude-3-5-sonnet-20240620",
    "max_tokens": 8192,
    "api_key_env_var": "ANTHROPIC_API_KEY"
  },
  "created_at": "$(date -u +"%Y-%m-%dT%H:%M:%S+00:00")"
}
EOF
fi

echo "Installation and setup complete."
