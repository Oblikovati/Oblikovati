// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// OpenPackage reads the ZIP package at path into memory and runs the migration
// pipeline so the result is at [CurrentSchemaVersion]. It does not hold the file
// open — the package is fully buffered, which is what makes save atomic.
func OpenPackage(path string) (*Package, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("persistence: open %q: %w", path, err)
	}
	p, err := readZip(raw)
	if err != nil {
		return nil, fmt.Errorf("persistence: read %q: %w", path, err)
	}
	if err := Migrate(p); err != nil {
		return nil, fmt.Errorf("persistence: migrate %q: %w", path, err)
	}
	return p, nil
}

// Save writes the package to path atomically: it serializes the whole archive in
// memory, writes it to a sibling temp file, fsyncs, then renames over path. An
// interruption before the rename leaves any prior file at path untouched
// (architecture core/05: replaces COM compaction-on-save with a simpler invariant).
func (p *Package) Save(path string) error {
	data, err := p.marshalZip()
	if err != nil {
		return fmt.Errorf("persistence: marshal %q: %w", path, err)
	}
	return atomicWriteFile(path, data)
}

// marshalZip renders the package as a deflate-compressed ZIP archive with streams
// in deterministic order (see streamNames) for stable output.
func (p *Package) marshalZip() ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range p.streamNames() {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(p.streams[name]); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// readZip rebuilds a package from ZIP bytes, preserving entry order.
func readZip(raw []byte) (*Package, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}
	p := NewPackage()
	for _, entry := range zr.File {
		data, err := readZipEntry(entry)
		if err != nil {
			return nil, err
		}
		p.WriteStream(entry.Name, data)
	}
	return p, nil
}

func readZipEntry(entry *zip.File) ([]byte, error) {
	rc, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
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
