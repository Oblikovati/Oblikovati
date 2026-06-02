// SPDX-License-Identifier: GPL-2.0-only

package persistence

import "errors"

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
	manifest Manifest          // identity; the zero value means "no manifest"
	model    []byte            // recipe YAML bytes; nil ⇒ no model
	streams  map[string][]byte // named binary data sections
	order    []string          // data-section insertion order, for stable enumeration
}

// NewPackage returns an empty package.
func NewPackage() *Package {
	return &Package{streams: map[string][]byte{}}
}

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
