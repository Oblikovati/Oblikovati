// SPDX-License-Identifier: GPL-2.0-only

package doc

// Resource is one imported file embedded in a document's root resource table (ADR-0031):
// a self-contained copy of a file a recipe step consumed (a mesh, a STEP, a font), so the
// document needs nothing outside itself to reopen. Value is the file's raw bytes; Encoding
// records how they are stored on disk (verbatim text vs base64) so a reader decodes by it.
type Resource struct {
	Type     string // extensible type tag: ObjFile/StepFile/StlFile/TrueTypeFont/ThreeMFFile/...
	Encoding string // EncodingUTF8 (verbatim text) | EncodingBase64 (binary)
	Value    []byte // the file's bytes
	Origin   string // optional: the original filename (display/round-trip metadata only)
}

const (
	// EncodingUTF8 stores a resource's bytes verbatim as text (git-diffable). Its string value
	// matches yamlcodec's so the persistence bridge copies it straight through.
	EncodingUTF8 = "utf8"
	// EncodingBase64 stores a resource's bytes base64-encoded (binary files).
	EncodingBase64 = "base64"
)

// ResourceBearer is the optional interface content implements when it owns a resource table
// (ADR-0031). The store reads it on save and restores it on open, BEFORE applying the recipe
// (so a feature that re-derives geometry from a resource can read its bytes). Like
// [RecipeContent], it is optional: content without imported files need not implement it.
type ResourceBearer interface {
	Content
	// Resources returns the document's resource table, keyed by per-import UUID.
	Resources() map[string]Resource
	// SetResources replaces the table (called on open before the recipe is applied).
	SetResources(map[string]Resource)
}
