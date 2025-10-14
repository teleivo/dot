#!/bin/bash
# Sync test graphs from graphviz repository for visual testing

set -e

GRAPHVIZ_REPO="https://gitlab.com/graphviz/graphviz.git"
TARGET_DIR="samples-graphviz"
BRANCH="main"

cd "$(dirname "$0")"

if [ -d "$TARGET_DIR/.git" ]; then
    echo "Updating existing samples..."
    cd "$TARGET_DIR"
    git pull
else
    echo "Cloning graphviz samples..."
    git clone --filter=blob:none --no-checkout --sparse "$GRAPHVIZ_REPO" "$TARGET_DIR"
    cd "$TARGET_DIR"
    git sparse-checkout init
    git sparse-checkout set --no-cone '*.dot' '*.gv'
    git checkout "$BRANCH"
fi

echo "Done. Samples in $TARGET_DIR/"
