// Package diagnostic provides parse error diagnostics for DOT graph files.
package diagnostic

import (
	"github.com/teleivo/dot"
	"github.com/teleivo/dot/lsp/internal/rpc"
)

// Compute returns diagnostics for parse errors in the given source.
func Compute(src []byte, uri rpc.DocumentURI, version int32) rpc.PublishDiagnosticsParams {
	ps := dot.NewParser(src)
	ps.Parse()

	params := rpc.PublishDiagnosticsParams{
		URI:     uri,
		Version: &version,
	}
	sev := rpc.SeverityError
	errs := ps.Errors()
	params.Diagnostics = make([]rpc.Diagnostic, len(errs))
	for i, err := range errs {
		params.Diagnostics[i] = rpc.Diagnostic{
			Range: rpc.Range{
				Start: rpc.Position{
					Line:      uint32(err.Pos.Line) - 1,
					Character: uint32(err.Pos.Column) - 1,
				},
				End: rpc.Position{
					Line:      uint32(err.Pos.Line) - 1,
					Character: uint32(err.Pos.Column) - 1,
				},
			},
			Severity: &sev,
			Message:  err.Msg,
		}
	}

	return params
}
