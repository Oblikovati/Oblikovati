// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"fmt"
	"os"
)

// DataIO reads and writes arbitrary named data streams in a [Package]. It is the
// surface add-ins and the attribute layer (M03-F06) use to stash client data
// alongside the model, without knowing the package's internal layout. Modernizes
// COM's DataIO object.
type DataIO struct {
	pkg *Package
}

// NewDataIO binds a DataIO to a package.
func NewDataIO(pkg *Package) *DataIO {
	return &DataIO{pkg: pkg}
}

// WriteData stores data under the given stream name.
func (d *DataIO) WriteData(stream string, data []byte) {
	d.pkg.WriteStream(stream, data)
}

// ReadData returns the bytes stored under stream, or false if absent.
func (d *DataIO) ReadData(stream string) ([]byte, bool) {
	return d.pkg.ReadStream(stream)
}

// WriteDataToFile stores data into a single stream of the package at path,
// preserving any other streams already there, and saves atomically. The package
// is created if it does not yet exist.
func WriteDataToFile(path, stream string, data []byte) error {
	pkg, err := openOrNew(path)
	if err != nil {
		return err
	}
	NewDataIO(pkg).WriteData(stream, data)
	return pkg.Save(path)
}

// ReadDataFromFile reads one stream from the package at path.
func ReadDataFromFile(path, stream string) ([]byte, error) {
	pkg, err := OpenPackage(path)
	if err != nil {
		return nil, err
	}
	data, ok := NewDataIO(pkg).ReadData(stream)
	if !ok {
		return nil, fmt.Errorf("persistence: stream %q not found in %q", stream, path)
	}
	return data, nil
}

// openOrNew opens the package at path, or returns a fresh one if no file is there.
func openOrNew(path string) (*Package, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return NewPackage(), nil
	}
	return OpenPackage(path)
}
