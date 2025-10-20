# Visual Test Failures Analysis

## Overview

The visual tests currently fail for several distinct reasons. Since HTML label support is not
implemented, many files fail due to HTML-related parsing issues. Other failures are due to Graphviz
bugs and visual output differences.

## Failure Categories

### 1. Formatting Failures - Silent Output Errors (8 files)

These files appear to "format successfully" but actually produce empty or incomplete output,
causing visual differences. The formatter fails silently without error messages.

**Failed Files:**

* `1367.dot` - Invalid UTF-8 byte (0x80) in identifier
* `b34.gv` - ISO-8859-1 encoding (Latin1 character ñ)
* `b56.gv` - ISO-8859-1 encoding (German umlauts ä, ö, ü, ß)
* `b60.gv` - ISO-8859-1 encoding (extended ASCII 0x80-0xFF)
* `b100.gv` - ISO-8859-1 encoding (764KB file)
* `Latin1.gv` - ISO-8859-1 encoding (various accented characters)
* `1845.dot` - Multiple graphs in single file
* `multi.gv` - Multiple graphs in single file

**Root Causes:**

Two distinct issues cause formatting failures:

**A. Multiple Graphs in Single File (2 files: `1845.dot`, `multi.gv`)**

* Files contain multiple graph definitions separated by closing braces
* The formatter only processes and outputs the first graph
* Subsequent graphs are silently ignored
* Example: `1845.dot` has two digraphs but only outputs `graph {}`

**B. Non-UTF-8 Encoding (6 files: `1367.dot`, `b34.gv`, `b56.gv`, `b60.gv`, `b100.gv`,
`Latin1.gv`)**

* Files are encoded in ISO-8859-1 (Latin1) or contain invalid UTF-8 bytes
* The scanner assumes UTF-8 input and fails when encountering non-UTF-8 byte sequences
* Common characters: ñ (0xF1), ä (0xE4), ö (0xF6), ü (0xFC), ß (0xDF)
* Result: Empty graph output `graph {}`
* Workaround: `cat file.gv | iconv -f ISO-8859-1 -t UTF-8 | dotfmt` works correctly

**Evidence:**

```bash
# Multiple graphs - only first is output
$ cat samples-graphviz/tests/1845.dot
digraph { fear -> anger; }
digraph { anger -> hate; }

$ go run cmd/dotfmt/main.go samples-graphviz/tests/1845.dot
graph {}

# Non-UTF-8 encoding - empty output
$ file samples-graphviz/tests/graphs/b34.gv
samples-graphviz/tests/graphs/b34.gv: ISO-8859 text

$ go run cmd/dotfmt/main.go samples-graphviz/tests/graphs/b34.gv
graph {}

# With conversion - works correctly
$ cat samples-graphviz/tests/graphs/b34.gv | iconv -f ISO-8859-1 -t UTF-8 | \
  go run cmd/dotfmt/main.go
digraph grafo {
    graph [charset=latin1]
    "0" [fontname=verdana,height=0.1,width=0.1,shape=box,label="seres"]
    ...
}
```

**Status:** These are formatter bugs requiring fixes:

1. **Input encoding:** Add encoding detection/conversion or report clear errors for non-UTF-8
   input
2. **Multiple graphs:** Either support formatting all graphs or report an error (currently fails
   silently)
3. **Error reporting:** Exit with non-zero status and display error messages instead of producing
   empty output

### 2. Graphviz Assertion Failures (15 files)

Several test files cause the Graphviz `dot` tool itself to crash with assertion failures. These
are not `dotfmt` bugs but issues with the reference implementation.

**Failed Files:**

* `121.dot`: Assertion 'ED_to_virt(e) == NULL' failed (class2.c:148)
* `1408.dot`: Assertion failure
* `1447_1.dot`: Assertion failure
* `1453.dot`: Assertion failure
* `14.dot`: Assertion failure
* `1514.dot`: Assertion failure
* `1622_0.dot`: Assertion 'delx >= 0' failed (htmltable.c:1761)
* `1622_1.dot`: Assertion failure (htmltable.c)
* `1622_2.dot`: Assertion failure
* `1622_3.dot`: Assertion failure
* `1880.dot`: Assertion failure
* `1902.dot`: Assertion failure
* `1949.dot`: Assertion 'bez->eflag' failed (compound.c:429)
* `56.dot`: Assertion failure (also has format error - see category 1)
* `1308_1.dot`: Assertion failure

**Status:** These are Graphviz bugs, not formatter issues. Tests should be skipped or marked as
expected failures.

### 3. Format Errors - HTML Labels Not Supported (44 files)

Files using HTML-like labels (between `<<` and `>>`) fail because HTML label support is not
implemented in `dotfmt`.

**Error Message Pattern:**

```
unquoted string identifiers can contain alphabetic ([a-zA-Z\200-\377]) characters,
underscores ('_') or digits([0-9]), but not begin with a digit
```

**Failed Files:**

* HTML-specific files (13): `html.dot`, `html.gv`, `html2.gv`, `html_dot.gv`, `html2_dot.gv`,
  `tee.gv`, `table.gv`, `ports.gv`, `ports_dot.gv`, `sides.gv`, `url.gv`, `rd_rules.gv`,
  `sq_rules.gv`
* Gradient files with HTML (4): `grdlinear.gv`, `grdradial.gv`, `grdradial_angle.gv`,
  `grdfillcolor.gv`
* Font name files (8): `AvantGarde.gv`, `Bookman.gv`, `Helvetica.gv`, `NewCenturySchlbk.gv`,
  `Palatino.gv`, `Times.gv`, `ZapfChancery.gv`, `ZapfDingbats.gv`
* Validation test files (9): `inv_inv.gv`, `inv_nul.gv`, `inv_val.gv`, `nul_inv.gv`,
  `nul_nul.gv`, `nul_val.gv`, `val_inv.gv`, `val_nul.gv`, `val_val.gv`
* Regression test files (10): `1425.dot`, `1425_1.dot`, `1472.dot`, `1898.dot`, `2159.dot`,
  `2242.dot`, `2295.dot`, `2497.dot`, `2538.dot`, `2592.dot`

**Note:** 56.dot appears in both FORMAT_ERROR and ASSERTION categories (it has both issues).

**Root Cause:**

HTML labels use special syntax with tags like `<table>`, `<tr>`, `<td>`, `<font>`, etc. The
scanner tries to parse HTML tags as regular identifiers and fails when encountering characters
that aren't valid in unquoted strings.

**Status:** HTML label support is not implemented in `dotfmt` yet. This is a known limitation.

## Summary Statistics

* **Total failures:** 67 unique files (some appear in multiple categories)
* **HTML-related format errors:** 44 files (known limitation)
* **Graphviz crashes:** 15 files (not formatter bugs)
* **Formatting failures:** 8 files (6 encoding issues + 2 multiple graph issues)

## Recommendations

1. **Fix encoding handling:** The 6 files with non-UTF-8 encoding need proper handling:
   * Option A: Auto-detect ISO-8859-1/Latin1 and convert to UTF-8 before processing
   * Option B: Report clear error for non-UTF-8 input with conversion suggestion
   * Option C: Handle input as raw bytes (match Graphviz's permissive behavior)
2. **Fix multiple graph handling:** The 2 files with multiple graphs need proper handling:
   * Option A: Format all graphs in the file
   * Option B: Report error when multiple graphs detected
   * Option C: Document current limitation clearly
3. **Improve error reporting:** All silent failures should report errors and exit with non-zero
   status
4. **Skip known-bad files:** Files that crash Graphviz should be excluded from visual tests or
   marked as expected failures
5. **HTML labels:** This is a known limitation. Consider adding HTML label support or documenting
   that HTML labels are not supported

