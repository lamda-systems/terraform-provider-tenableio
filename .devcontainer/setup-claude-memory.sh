#!/usr/bin/env bash
# Point Claude Code's auto-memory at .claude/memory/ in this repository, so the
# memories are committed with the code and survive a container rebuild.
#
# The setting has to live in .claude/settings.local.json: Claude Code ignores
# autoMemoryDirectory in the checked-in .claude/settings.json, for security. The
# local file is gitignored, so this script recreates it on every container
# create. It needs an absolute path, and the checkout sits at a different path
# in a codespace than in the local devcontainer, so the path is resolved here.
#
# Idempotent: an existing settings.local.json keeps every other key.
set -euo pipefail

root="$(git -C "$(dirname "$0")" rev-parse --show-toplevel)"
memory="$root/.claude/memory"
settings="$root/.claude/settings.local.json"

mkdir -p "$memory"

python3 - "$settings" "$memory" <<'EOF'
import json, os, sys

path, memory = sys.argv[1], sys.argv[2]
settings = {}
if os.path.exists(path):
    with open(path) as f:
        settings = json.load(f)
settings["autoMemoryDirectory"] = memory
with open(path, "w") as f:
    json.dump(settings, f, indent=2)
    f.write("\n")
EOF

echo "Claude Code auto-memory: $memory"
