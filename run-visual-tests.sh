#!/bin/bash
# Run all Graphviz DOT files through the visual test. Assumes you did ./sync-graphviz-samples.sh
# before.
set -euo pipefail

SAMPLES_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/samples-graphviz"
ERROR_LOG="visual-test-error.log"

: > "$ERROR_LOG"

total=0
failed=0

while IFS= read -r -d '' dir; do
    total=$((total + 1))
    rel_dir="${dir#"$SAMPLES_DIR"/}"
    printf "Testing: %s" "$rel_dir"

    # Build test command with optional timeout
    test_cmd="DOTX_TEST_DIR=\"$dir\" go test -C cmd/dotx -v"
    if [ -n "${DOTX_TEST_TIMEOUT:-}" ]; then
        test_cmd="$test_cmd -timeout $DOTX_TEST_TIMEOUT"
    fi
    test_cmd="$test_cmd -run TestVisualOutput"

    if ! output=$(eval "$test_cmd" 2>&1); then
        has_skip=$(echo "$output" | grep -q "SKIP" && echo "yes" || echo "no")

        if [ "$has_skip" = "yes" ]; then
            echo -e " \033[90m-\033[0m"
            {
                echo "SKIPPED: $rel_dir (dir: $dir)"
                echo "$output"
                echo ""
            } >> "$ERROR_LOG"
        else
            failed=$((failed + 1))
            echo -e " \033[31m✘\033[0m"
            {
                echo "FAILED: $rel_dir (dir: $dir)"
                echo "$output"
                echo ""
            } >> "$ERROR_LOG"
        fi
    else
        echo -e " \033[32m✔\033[0m"
    fi
done < <(find "$SAMPLES_DIR" -type d -print0)

echo "Tested: $total, Failed: $failed"
[ $failed -eq 0 ] && echo "All passed" || echo "Errors in $ERROR_LOG"
exit $failed
