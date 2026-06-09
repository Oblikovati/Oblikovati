// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"fmt"
	"os"
	"path/filepath"

	"oblikovati/persistence/yamlcodec"
)

// OpenPackage reads the YAML document at path into memory and runs the migration
// pipeline so the result is at [CurrentSchemaVersion]. It does not hold the file
// open — the document is fully buffered, which is what makes save atomic. A legacy
// ZIP package is rejected with a clear message (ADR-0020).
func OpenPackage(path string) (*Package, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("persistence: open %q: %w", path, err)
	}
	p, err := decode(raw)
	if err != nil {
		return nil, fmt.Errorf("persistence: read %q: %w", path, err)
	}
	if err := Migrate(p); err != nil {
		return nil, fmt.Errorf("persistence: migrate %q: %w", path, err)
	}
	return p, nil
}

// Save writes the package to path atomically: it serializes the whole document in
// memory, writes it to a sibling temp file, fsyncs, then renames over path. An
// interruption before the rename leaves any prior file at path untouched
// (architecture core/05).
func (p *Package) Save(path string) error {
	data, err := p.marshal()
	if err != nil {
		return fmt.Errorf("persistence: marshal %q: %w", path, err)
	}
	return atomicWriteFile(path, data)
}

// marshal renders the package as a single YAML document (manifest + recipe + data
// sections). Map keys are emitted in sorted order by the YAML library, so output is
// stable for clean git diffs.
func (p *Package) marshal() ([]byte, error) {
	return yamlcodec.MarshalDocument(yamlcodec.Document{
		SchemaVersion: p.manifest.SchemaVersion,
		DocumentType:  p.manifest.DocumentType,
		DisplayName:   p.manifest.DisplayName,
		Model:         p.model,
		Data:          p.streams,
		Resources:     p.resources,
	})
}

// decode rebuilds a package from a document's YAML bytes.
func decode(raw []byte) (*Package, error) {
	doc, err := yamlcodec.UnmarshalDocument(raw)
	if err != nil {
		return nil, err
	}
	p := NewPackage()
	if doc.SchemaVersion != 0 || doc.DocumentType != 0 || doc.DisplayName != "" {
		p.manifest = Manifest{
			SchemaVersion: doc.SchemaVersion,
			DocumentType:  doc.DocumentType,
			DisplayName:   doc.DisplayName,
		}
	}
	p.model = doc.Model
	p.resources = doc.Resources
	for name, data := range doc.Data {
		p.WriteStream(name, data)
	}
	return p, nil
}

// atomicWriteFile writes data to path via a sibling temp file and a rename, so the
// file at path is only ever the fully-written previous or next version.
func atomicWriteFile(path string, data []byte) error {
	tmp, err := stage(path, data)
	if err != nil {
		return err
	}
	if err := commit(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// stage writes data to a fresh temp file beside path and returns its name. The
// original at path is not touched. On any error the temp is removed.
func stage(path string, data []byte) (string, error) {
	f, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return "", err
	}
	if err := writeAndSync(f, data); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

func writeAndSync(f *os.File, data []byte) error {
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Chmod(0o644); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

// commit renames the staged temp over the destination — the atomic step.
func commit(tmpPath, path string) error {
	return os.Rename(tmpPath, path)
}
