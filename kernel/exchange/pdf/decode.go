// SPDX-License-Identifier: GPL-2.0-only

package pdf

import (
	"fmt"

	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/drawing"
)

// Decode parses a vector PDF and returns the curve geometry of its FIRST page, mirroring
// dwg.Decode / dxf.Decode so a single-page CAD plot imports onto one sketch. Use
// DecodePages for every page. Coordinates are returned in millimetres (Units =
// drawing.INSMillimetres); the page user-space-to-millimetre scale (1 pt = 1/72 inch) is
// baked in. Per-page problems (unsupported operators, skipped text/images) are warnings,
// not errors.
//
// Example:
//
//	dr, warns, err := pdf.Decode(bytes)
//	for _, e := range dr.Entities { /* convert to sketch geometry */ }
func Decode(data []byte) (*drawing.Drawing, []string, error) {
	pages, warns, err := DecodePages(data)
	if err != nil {
		return nil, warns, err
	}
	return pages[0], warns, nil
}

// DecodePages parses a vector PDF and returns one drawing per page, in document order, plus
// the aggregated per-page warnings. It errors only on a whole-file failure (an unsupported
// xref-stream/object-stream PDF, or a structure with no pages); a page that decodes to no
// geometry yields an empty drawing rather than an error.
func DecodePages(data []byte) ([]*drawing.Drawing, []string, error) {
	return DecodePagesWithProgress(data, exchange.TranslationOptions{})
}

// DecodePagesWithProgress is [DecodePages] threaded through the shared progress/cancel seam
// (#1647): opts reports one tick per page and aborts the import when its ProgressFunc returns
// cancel (the returned error wraps [exchange.ErrCancelled]). DecodePages is this call with a zero
// options value.
func DecodePagesWithProgress(data []byte, opts exchange.TranslationOptions) ([]*drawing.Drawing, []string, error) {
	doc, err := newDocument(data)
	if err != nil {
		return nil, nil, err
	}
	pages, err := doc.pages()
	if err != nil {
		return nil, nil, err
	}
	drawings := make([]*drawing.Drawing, 0, len(pages))
	var warns []string
	for i, pg := range pages {
		if perr := opts.Report("pages", i, len(pages)); perr != nil {
			return drawings, warns, perr
		}
		ents, pageWarns := interpretPage(doc, pg)
		warns = append(warns, prefixPageWarnings(i+1, pageWarns)...)
		drawings = append(drawings, &drawing.Drawing{Units: drawing.INSMillimetres, Entities: ents})
	}
	return drawings, warns, nil
}

// interpretPage runs the content interpreter over one page and returns its entities and
// warnings. The initial CTM subtracts the media-box origin so content sits near (0,0).
func interpretPage(doc *document, pg page) ([]drawing.Entity, []string) {
	in := &interp{
		doc:       doc,
		resources: pg.resources,
		ctm:       translateMatrix(-pg.originX, -pg.originY),
	}
	in.run(pg.content)
	return in.out, in.warns
}

// prefixPageWarnings tags each warning with its page number so a multi-page import says
// where a problem occurred.
func prefixPageWarnings(pageNum int, warns []string) []string {
	if len(warns) == 0 {
		return nil
	}
	out := make([]string, len(warns))
	for i, w := range warns {
		out[i] = fmt.Sprintf("page %d: %s", pageNum, w)
	}
	return out
}
