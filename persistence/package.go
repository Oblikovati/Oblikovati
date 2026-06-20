// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"errors"

	"oblikovati.org/persistence/yamlcodec"
)

// errNoManifest reports a document with no manifest — not a valid document file
// (though still a valid container of arbitrary data sections, e.g. for DataIO).
var errNoManifest = errors.New("persistence: package has no manifest")

// StreamStat is the size/name of one data section. It modernizes COM's tagSTATSTG,
// kept to the two fields callers actually use here.
type StreamStat struct {
	Name string
	Size int64
}

// Package is an in-memory document: its manifest identity, the recipe section (the
// model, as opaque YAML bytes owned by model/compdef), and an ordered set of named
// binary data sections (DataIO / attribute scratch). On disk it is one YAML text
// file (ADR-0020). Stream bytes are copied in and out so a Package never shares
// backing arrays with its callers.
type Package struct {
	manifest    Manifest                      // identity; the zero value means "no manifest"
	model       []byte                        // recipe YAML bytes; nil ⇒ no model
	streams     map[string][]byte             // named binary data sections
	order       []string                      // data-section insertion order, for stable enumeration
	resources   map[string]yamlcodec.Resource // embedded imported files, keyed by UUID (ADR-0031)
	identity    *yamlcodec.FileIdentityRecord // file identity block (M03-F07, #159); nil pre-identity
	references  []yamlcodec.FileReferenceRecord
	attachments []yamlcodec.AttachmentRecord     // external-file attachments (M03-F08)
	interests   []yamlcodec.InterestRecord       // add-in data registry (M03-F10)
	attributes  []byte                           // add-in attribute sets, encoded (#155)
	display     *yamlcodec.DisplaySettingsRecord // per-document display settings (M16-F07 #643)
	sketch      *yamlcodec.SketchSettingsRecord  // per-document sketch settings (#147)
	bodyNames   map[string]string                // per-body display names by reference key (#1078)
}

// NewPackage returns an empty package.
func NewPackage() *Package {
	return &Package{streams: map[string][]byte{}}
}

// SetIdentity stores the file identity block (M03-F07, #159).
func (p *Package) SetIdentity(id *yamlcodec.FileIdentityRecord) { p.identity = id }

// Identity returns the file identity block, nil for a pre-identity file.
func (p *Package) Identity() *yamlcodec.FileIdentityRecord { return p.identity }

// SetFileReferences stores the as-saved file-to-file reference records (M03-F07).
func (p *Package) SetFileReferences(refs []yamlcodec.FileReferenceRecord) { p.references = refs }

// FileReferences returns the as-saved file-to-file reference records.
func (p *Package) FileReferences() []yamlcodec.FileReferenceRecord { return p.references }

// SetAttachments stores the external-file attachment records (M03-F08).
func (p *Package) SetAttachments(a []yamlcodec.AttachmentRecord) { p.attachments = a }

// Attachments returns the external-file attachment records.
func (p *Package) Attachments() []yamlcodec.AttachmentRecord { return p.attachments }

// SetDisplaySettings stores the per-document display settings record (M16-F07 #643).
func (p *Package) SetDisplaySettings(d *yamlcodec.DisplaySettingsRecord) { p.display = d }

// DisplaySettings returns the per-document display settings record (nil when unset).
func (p *Package) DisplaySettings() *yamlcodec.DisplaySettingsRecord { return p.display }

// SetSketchSettings stores the per-document sketch settings record (#147).
func (p *Package) SetSketchSettings(s *yamlcodec.SketchSettingsRecord) { p.sketch = s }

// SketchSettings returns the per-document sketch settings record (nil when unset).
func (p *Package) SketchSettings() *yamlcodec.SketchSettingsRecord { return p.sketch }

// SetBodyNames stores the per-body display-name map, keyed by body reference key (#1078).
func (p *Package) SetBodyNames(names map[string]string) { p.bodyNames = names }

// BodyNames returns the per-body display-name map (nil when no body was renamed).
func (p *Package) BodyNames() map[string]string { return p.bodyNames }

// SetInterests stores the add-in data-registry records (M03-F10).
func (p *Package) SetInterests(i []yamlcodec.InterestRecord) { p.interests = i }

// Interests returns the add-in data-registry records.
func (p *Package) Interests() []yamlcodec.InterestRecord { return p.interests }

// SetAttributes stores the encoded add-in attribute sets (#155).
func (p *Package) SetAttributes(a []byte) { p.attributes = a }

// Attributes returns the encoded add-in attribute sets, nil when none.
func (p *Package) Attributes() []byte { return p.attributes }

// SetResources stores the document's embedded resource table (ADR-0031), keyed by UUID.
func (p *Package) SetResources(r map[string]yamlcodec.Resource) { p.resources = r }

// Resources returns the document's embedded resource table, or nil if there are none.
func (p *Package) Resources() map[string]yamlcodec.Resource { return p.resources }

// WriteStream stores data under name, replacing any existing section of that name.
// The bytes are copied; later mutations by the caller do not affect the package.
func (p *Package) WriteStream(name string, data []byte) {
	if _, exists := p.streams[name]; !exists {
		p.order = append(p.order, name)
	}
	clone := make([]byte, len(data))
	copy(clone, data)
	p.streams[name] = clone
}

// ReadStream returns a copy of the named section's bytes, or false if absent.
func (p *Package) ReadStream(name string) ([]byte, bool) {
	data, ok := p.streams[name]
	if !ok {
		return nil, false
	}
	clone := make([]byte, len(data))
	copy(clone, data)
	return clone, true
}

// DeleteStream removes a data section, reporting whether it existed.
func (p *Package) DeleteStream(name string) bool {
	if _, ok := p.streams[name]; !ok {
		return false
	}
	delete(p.streams, name)
	for i, n := range p.order {
		if n == name {
			p.order = append(p.order[:i], p.order[i+1:]...)
			break
		}
	}
	return true
}

// Streams returns a stat for every data section, in write order.
func (p *Package) Streams() []StreamStat {
	stats := make([]StreamStat, 0, len(p.order))
	for _, name := range p.order {
		stats = append(stats, StreamStat{Name: name, Size: int64(len(p.streams[name]))})
	}
	return stats
}

// Stat returns the stat of one data section, or false if absent.
func (p *Package) Stat(name string) (StreamStat, bool) {
	data, ok := p.streams[name]
	if !ok {
		return StreamStat{}, false
	}
	return StreamStat{Name: name, Size: int64(len(data))}, true
}

// Manifest returns the document's manifest, or errNoManifest if none has been set
// (a manifest-less data container).
func (p *Package) Manifest() (Manifest, error) {
	if p.manifest == (Manifest{}) {
		return Manifest{}, errNoManifest
	}
	return p.manifest, nil
}

// SetManifest records the document's identity.
func (p *Package) SetManifest(m Manifest) error {
	p.manifest = m
	return nil
}

// ModelYAML returns a copy of the recipe section's YAML bytes, or false if the
// document has no model. The recipe schema is owned by model/compdef; persistence
// treats it as opaque bytes embedded natively in the file (ADR-0020).
func (p *Package) ModelYAML() ([]byte, bool) {
	if len(p.model) == 0 {
		return nil, false
	}
	out := make([]byte, len(p.model))
	copy(out, p.model)
	return out, true
}

// SetModelYAML stores the recipe section's YAML bytes (copied).
func (p *Package) SetModelYAML(b []byte) {
	p.model = append([]byte(nil), b...)
}
