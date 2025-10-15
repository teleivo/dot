#!/bin/bash
# Run all Graphviz DOT files through the visual test. Assumes you did ./sync-graphviz-samples.sh
# before.
set -euo pipefail

SAMPLES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/samples-graphviz"
ERROR_LOG="visual_test_error.log"

: > "$ERROR_LOG"

total=0
failed=0

while IFS= read -r -d '' dir; do
    total=$((total + 1))
    rel_dir="${dir#"$SAMPLES_DIR"/}"
    echo "Testing: $rel_dir"

    # Build test command with optional timeout
    test_cmd="DOTFMT_TEST_DIR=\"$dir\" go test -v"
    if [ -n "${DOTFMT_TEST_TIMEOUT:-}" ]; then
        test_cmd="$test_cmd -timeout $DOTFMT_TEST_TIMEOUT"
    fi
    test_cmd="$test_cmd ./cmd/dotfmt -run TestVisualOutput"

    if ! output=$(eval "$test_cmd" 2>&1); then
        if ! echo "$output" | grep -q "SKIP"; then
            failed=$((failed + 1))
            {
                echo "FAILED: $rel_dir (dir: $dir)"
                echo "$output"
                echo ""
            } >> "$ERROR_LOG"
        fi
    fi
done < <(find "$SAMPLES_DIR" -type d -print0)

echo "Tested: $total, Failed: $failed"
[ $failed -eq 0 ] && echo "All passed" || echo "Errors in $ERROR_LOG"
exit $failed
