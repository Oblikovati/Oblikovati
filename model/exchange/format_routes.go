// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"fmt"
	"strings"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/exchange"
	"oblikovati.org/kernel/exchange/meshio"
	"oblikovati.org/kernel/exchange/step"
	"oblikovati.org/kernel/topo"
)

// Exchange format routing is one registry (#1631, audit I8): each format registers its
// extensions and its capabilities (a drawing decode entry, a body export entry) together,
// replacing the two hand-maintained switches in dispatch.go — the format dispatch and the
// FormatFromPath extension map — whose independent upkeep let a format be recognized from
// the menu yet refused at import (the #1416 drift class). The set is built by ONE visible
// construction (defaultFormatRoutes; no init() registration, matching the feature-codec
// registry's #1617 discipline), and registration panics on a duplicate format or extension.

// exportBodiesFunc writes bodies to a format's byte encoding: data, triangle count (0 for an
// exact B-rep write), warnings, error — the body-export capability of a formatRoute.
type exportBodiesFunc func(bodies []*topo.Body, res types.MeshResolution, opts exchange.TranslationOptions) ([]byte, int, []string, error)

// formatRoute is everything the dispatcher knows about one exchange format: the extensions
// it is recognized by, and its optional capabilities. A sketch (drawing) format carries its
// decoder; a body format carries its exporter. Both are registered in the same call, so
// extension recognition and dispatch cannot drift.
type formatRoute struct {
	format       types.ExchangeFormat
	extensions   []string
	decoder      DrawingDecoder   // non-nil exactly for sketch formats
	exportBodies exportBodiesFunc // non-nil for formats Export can write
}

// formatRouteSet indexes the routes by format and by extension — the seam FormatFromPath,
// Export and the drawing importers all consult.
type formatRouteSet struct {
	byFormat    map[types.ExchangeFormat]formatRoute
	byExtension map[string]types.ExchangeFormat
}

// register records one format's route, panicking on a duplicate format or extension and on
// a capability/format mismatch — programming errors caught at construction, not silent
// menu-vs-dispatch drift at runtime.
func (s *formatRouteSet) register(r formatRoute) {
	if _, dup := s.byFormat[r.format]; dup {
		panic(fmt.Sprintf("exchange: format %q registered twice", r.format))
	}
	if r.format.IsSketch() != (r.decoder != nil) {
		panic(fmt.Sprintf("exchange: format %q IsSketch=%v but decoder present=%v — a sketch format needs exactly one DrawingDecoder", r.format, r.format.IsSketch(), r.decoder != nil))
	}
	s.byFormat[r.format] = r
	s.registerExtensions(r)
}

// registerExtensions claims the route's extensions, panicking when one is already taken.
func (s *formatRouteSet) registerExtensions(r formatRoute) {
	for _, ext := range r.extensions {
		if owner, dup := s.byExtension[ext]; dup {
			panic(fmt.Sprintf("exchange: extension %q registered by both %q and %q", ext, owner, r.format))
		}
		s.byExtension[ext] = r.format
	}
}

// registerDrawing wraps a DrawingDecoder as a route — format, extensions and decode
// entry travel as one unit, the seam the next drawing importer registers through.
func (s *formatRouteSet) registerDrawing(d DrawingDecoder) {
	s.register(formatRoute{format: d.Format(), extensions: d.Extensions(), decoder: d})
}

// formatRoutes is the package's composition site: every production format, in one visible
// order. Adding an importer/exporter is one registration here plus its own package.
var formatRoutes = defaultFormatRoutes()

// defaultFormatRoutes assembles the full production route set: the mesh formats, STEP, and
// the drawing formats (DWG/DXF/PDF).
func defaultFormatRoutes() *formatRouteSet {
	s := &formatRouteSet{byFormat: map[types.ExchangeFormat]formatRoute{}, byExtension: map[string]types.ExchangeFormat{}}
	for _, f := range []types.ExchangeFormat{types.FormatSTL, types.FormatOBJ, types.Format3MF} {
		s.register(formatRoute{format: f, extensions: []string{"." + string(f)}, exportBodies: meshExportRoute(f)})
	}
	s.register(formatRoute{format: types.FormatSTEP, extensions: []string{".step", ".stp"}, exportBodies: stepExportRoute})
	s.registerDrawing(dwgDrawingDecoder{})
	s.registerDrawing(dxfDrawingDecoder{})
	s.registerDrawing(pdfDrawingDecoder{})
	return s
}

// meshExportRoute adapts the mesh translator to the export capability for one mesh format.
func meshExportRoute(f types.ExchangeFormat) exportBodiesFunc {
	return func(bodies []*topo.Body, res types.MeshResolution, opts exchange.TranslationOptions) ([]byte, int, []string, error) {
		data, tris, err := meshio.ExportBodies(f, bodies, res, opts)
		return data, tris, nil, err
	}
}

// stepExportRoute adapts the STEP writer (exact B-rep; resolution ignored) to the export
// capability.
func stepExportRoute(bodies []*topo.Body, _ types.MeshResolution, opts exchange.TranslationOptions) ([]byte, int, []string, error) {
	data, warns, err := step.Writer{}.ExportSolids(bodies, opts)
	return data, 0, warns, err
}

// drawingDecoderFor returns the registered decoder for a sketch format; ok is false for a
// format with no drawing decode entry.
func drawingDecoderFor(format types.ExchangeFormat) (DrawingDecoder, bool) {
	r, ok := formatRoutes.byFormat[format]
	if !ok || r.decoder == nil {
		return nil, false
	}
	return r.decoder, true
}

// formatForExtension resolves a lowercase dot-prefixed extension (".dwg") to its registered
// format — the one lookup behind FormatFromPath.
func formatForExtension(ext string) (types.ExchangeFormat, bool) {
	f, ok := formatRoutes.byExtension[strings.ToLower(ext)]
	return f, ok
}
