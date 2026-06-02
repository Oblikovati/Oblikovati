// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"errors"
	"sort"
)

// errNoManifest reports a package with no manifest stream — not a valid document
// package (though still a valid container of arbitrary streams, e.g. for DataIO).
var errNoManifest = errors.New("persistence: package has no manifest")

// StreamStat is the size/name of one stream. It modernizes COM's tagSTATSTG, kept
// to the two fields callers actually use here.
type StreamStat struct {
	Name string
	Size int64
}

// Package is an in-memory document container: an ordered set of named byte
// streams. Order is preserved so saved archives are stable (manifest first), which
// keeps diffs readable. Stream bytes are copied in and out so a Package never
// shares backing arrays with its callers.
type Package struct {
	streams map[string][]byte
	order   []string
}

// NewPackage returns an empty package.
func NewPackage() *Package {
	return &Package{streams: map[string][]byte{}}
}

// WriteStream stores data under name, replacing any existing stream of that name.
// The bytes are copied; later mutations by the caller do not affect the package.
func (p *Package) WriteStream(name string, data []byte) {
	if _, exists := p.streams[name]; !exists {
		p.order = append(p.order, name)
	}
	clone := make([]byte, len(data))
	copy(clone, data)
	p.streams[name] = clone
}

// ReadStream returns a copy of the named stream's bytes, or false if absent.
func (p *Package) ReadStream(name string) ([]byte, bool) {
	data, ok := p.streams[name]
	if !ok {
		return nil, false
	}
	clone := make([]byte, len(data))
	copy(clone, data)
	return clone, true
}

// DeleteStream removes a stream, reporting whether it existed.
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

// Streams returns a stat for every stream, in write order.
func (p *Package) Streams() []StreamStat {
	stats := make([]StreamStat, 0, len(p.order))
	for _, name := range p.order {
		stats = append(stats, StreamStat{Name: name, Size: int64(len(p.streams[name]))})
	}
	return stats
}

// Stat returns the stat of one stream, or false if absent.
func (p *Package) Stat(name string) (StreamStat, bool) {
	data, ok := p.streams[name]
	if !ok {
		return StreamStat{}, false
	}
	return StreamStat{Name: name, Size: int64(len(data))}, true
}

// Manifest decodes the package manifest, or returns errNoManifest if absent.
func (p *Package) Manifest() (Manifest, error) {
	data, ok := p.ReadStream(manifestStream)
	if !ok {
		return Manifest{}, errNoManifest
	}
	return decodeManifest(data)
}

// SetManifest encodes m into the manifest stream.
func (p *Package) SetManifest(m Manifest) error {
	data, err := m.encode()
	if err != nil {
		return err
	}
	p.WriteStream(manifestStream, data)
	return nil
}

// streamNames returns the stream names in a deterministic order (manifest first,
// then the rest sorted) for stable archive output regardless of write order.
func (p *Package) streamNames() []string {
	rest := make([]string, 0, len(p.order))
	hasManifest := false
	for _, n := range p.order {
		if n == manifestStream {
			hasManifest = true
			continue
		}
		rest = append(rest, n)
	}
	sort.Strings(rest)
	if hasManifest {
		return append([]string{manifestStream}, rest...)
	}
	return rest
}
