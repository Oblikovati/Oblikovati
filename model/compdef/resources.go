// SPDX-License-Identifier: GPL-2.0-only

package compdef

import (
	"bytes"
	"crypto/rand"
	"fmt"

	"oblikovati/model/doc"
	"oblikovati/model/feature"
	"oblikovati/model/text"
)

// A part definition owns the document resource table and exposes it three ways: to persistence
// as a [doc.ResourceBearer] (save/restore the table), to its feature engine as a
// [feature.ResourceStore] (re-derive imported bodies from embedded bytes), and as a
// [text.FontResolver] (resolve a text/emboss font from an embedded or app-provided resource) —
// ADR-0031.
var (
	_ doc.ResourceBearer    = (*PartComponentDefinition)(nil)
	_ feature.ResourceStore = (*PartComponentDefinition)(nil)
	_ text.FontResolver     = (*PartComponentDefinition)(nil)
)

// AddResource embeds an imported file in the document, returning the freshly minted UUID a
// feature cites to find it. A byte-identical resource already present is reused (its existing
// UUID is returned), so importing the same file twice stores it once.
func (d *PartComponentDefinition) AddResource(r doc.Resource) string {
	if id, ok := d.findResource(r); ok {
		return id
	}
	id := newUUID()
	d.resources[id] = r
	return id
}

// findResource returns the UUID of an entry equal to r (same type/encoding/origin/value), so
// repeated imports of one file — or repeated use of one app-provided font (which has no bytes,
// only an origin) — dedup to a single resource.
func (d *PartComponentDefinition) findResource(r doc.Resource) (string, bool) {
	for id, existing := range d.resources {
		if existing.Type == r.Type && existing.Encoding == r.Encoding && existing.Origin == r.Origin && bytes.Equal(existing.Value, r.Value) {
			return id, true
		}
	}
	return "", false
}

// EmbedSystemFont stores an OS/host font file's bytes in the document as a TrueTypeFont resource
// (base64) and returns its UUID — the value a text/emboss entity cites so reopening needs no host
// font (ADR-0031). origin is the source filename (display only).
func (d *PartComponentDefinition) EmbedSystemFont(data []byte, origin string) string {
	return d.AddResource(doc.Resource{Type: "TrueTypeFont", Encoding: doc.EncodingBase64, Value: data, Origin: origin})
}

// UseEmbeddedFont records that the document uses an APP-PROVIDED font face (family), returning a
// TrueTypeFont resource UUID with NO bytes — the application supplies the face on open. Lets the
// resource table list every font a document uses uniformly, app-bundled or host-embedded.
func (d *PartComponentDefinition) UseEmbeddedFont(family string) string {
	return d.AddResource(doc.Resource{Type: "TrueTypeFont", Encoding: doc.EncodingEmbedded, Origin: family})
}

// Resolve turns a text/emboss font reference into a parsed font (text.FontResolver). A ref that
// is a TrueTypeFont resource UUID resolves from the document — embedded bytes, or the app's
// bundled face for an app-provided entry; anything else falls back to the default resolver
// (treating the ref as a family name), so plain family-named text still renders.
func (d *PartComponentDefinition) Resolve(ref string) (*text.Font, error) {
	r, ok := d.resources[ref]
	if !ok || r.Type != "TrueTypeFont" {
		return text.DefaultResolver().Resolve(ref)
	}
	data := r.Value
	if r.Encoding == doc.EncodingEmbedded {
		b, ok := text.EmbeddedFontBytes(r.Origin)
		if !ok {
			return nil, fmt.Errorf("compdef: app-provided font %q (resource %q) is not bundled", r.Origin, ref)
		}
		data = b
	}
	return text.Parse(data)
}

// Resource returns the resource stored under id, if present.
func (d *PartComponentDefinition) Resource(id string) (doc.Resource, bool) {
	r, ok := d.resources[id]
	return r, ok
}

// ResourceBytes returns the raw bytes of the resource under id (feature.ResourceStore).
func (d *PartComponentDefinition) ResourceBytes(id string) ([]byte, bool) {
	r, ok := d.resources[id]
	if !ok {
		return nil, false
	}
	return r.Value, true
}

// Resources returns the document's resource table (doc.ResourceBearer).
func (d *PartComponentDefinition) Resources() map[string]doc.Resource { return d.resources }

// SetResources replaces the resource table — called on open before the recipe is applied so
// imported-body restore can read the embedded bytes (doc.ResourceBearer).
func (d *PartComponentDefinition) SetResources(m map[string]doc.Resource) {
	if m == nil {
		m = map[string]doc.Resource{}
	}
	d.resources = m
}

// newUUID returns a random RFC-4122 version-4 UUID string (ADR-0031 resource keys). It uses
// crypto/rand so keys never collide across imports or machines.
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("compdef: UUID entropy: %v", err)) // crypto/rand failing is unrecoverable
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
