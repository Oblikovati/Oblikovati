// SPDX-License-Identifier: GPL-2.0-only

package pdf

import (
	"bytes"
	"compress/zlib"
	"fmt"
	"io"
)

// decodeStream returns a stream's decoded bytes. It handles the one filter CAD plot-to-PDF
// content streams use — FlateDecode with Predictor 1 (no predictor) — and reports an
// explicit error for any other filter or a >1 predictor, so an unsupported encoding
// surfaces as a per-page warning rather than as silent empty geometry.
func decodeStream(d *document, s streamObj) ([]byte, error) {
	filter, ok := streamFilter(d, s.dict)
	if !ok || filter == "" {
		return s.raw, nil // unfiltered content stream
	}
	if filter != "FlateDecode" {
		return nil, fmt.Errorf("pdf: unsupported stream filter %q (only FlateDecode)", filter)
	}
	if p := predictor(d, s.dict); p > 1 {
		return nil, fmt.Errorf("pdf: unsupported FlateDecode predictor %d (only 1)", p)
	}
	return inflate(s.raw)
}

// streamFilter returns the (single) filter name. A filter array longer than one element is
// rejected by returning a sentinel the caller treats as unsupported — content streams here
// carry at most one filter.
func streamFilter(d *document, dict dictObj) (string, bool) {
	switch f := d.resolve(dict["Filter"]).(type) {
	case nameObj:
		return string(f), true
	case arrayObj:
		if len(f) == 1 {
			if n, ok := d.nameOf(f[0]); ok {
				return n, true
			}
		}
		return "array", true // a multi-filter chain we don't support
	default:
		return "", false
	}
}

// predictor reads /DecodeParms /Predictor, defaulting to 1 (none).
func predictor(d *document, dict dictObj) int {
	parms, ok := d.dictOf(dict["DecodeParms"])
	if !ok {
		return 1
	}
	if n, ok := parms["Predictor"].(numberObj); ok {
		return int(n)
	}
	return 1
}

// inflate zlib-decompresses FlateDecode bytes.
func inflate(raw []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("pdf: flate header: %w", err)
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil && len(out) == 0 {
		return nil, fmt.Errorf("pdf: flate inflate: %w", err)
	}
	return out, nil // tolerate a truncated tail when some bytes decoded
}
