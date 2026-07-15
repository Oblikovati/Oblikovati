// SPDX-License-Identifier: GPL-2.0-only

// Package sldprt decodes a SolidWorks part/assembly file (a Microsoft Compound File,
// the same CFBF/OLE2 container as an Inventor .ipt) into the pieces a translator maps
// onto Oblikovati. It is the offline twin of the SolidWorks COM API: it reads .SLDPRT
// files directly and needs no running SolidWorks.
//
// Reverse-engineered against SolidWorks 2016 (internal build 3400). The feature tree,
// its sketches, and parameters live in the "Contents/Config-0" stream as a UTF-16
// node list; the solid B-rep is a Parasolid partition in "Contents/Config-0-Partition".
package sldprt

import (
	"errors"

	"oblikovati.org/model/exchange/translators/olecf"
)

// DocType classifies a SolidWorks compound document by its root-storage CLSID.
type DocType int

const (
	Unknown DocType = iota
	Part
	Assembly
	Drawing
)

func (t DocType) String() string {
	switch t {
	case Part:
		return "part"
	case Assembly:
		return "assembly"
	case Drawing:
		return "drawing"
	default:
		return "unknown"
	}
}

// Root-storage CLSIDs that identify a SolidWorks document kind. These are the raw on-disk
// bytes (confirmed from real files + the .prtdot/.asmdot/.drwdot templates); the three kinds
// share a family and differ only in byte[0]: part 0x30, drawing 0x34, assembly 0x36.
// Part {83A3DA30-27C5-11CE-BFD4-00400513BB57}.
var (
	clsidPart     = [16]byte{0x30, 0x3d, 0xa3, 0x83, 0xc5, 0x27, 0xce, 0x11, 0xbf, 0xd4, 0x00, 0x40, 0x05, 0x13, 0xbb, 0x57}
	clsidAssembly = [16]byte{0x36, 0x3d, 0xa3, 0x83, 0xc5, 0x27, 0xce, 0x11, 0xbf, 0xd4, 0x00, 0x40, 0x05, 0x13, 0xbb, 0x57}
	clsidDrawing  = [16]byte{0x34, 0x3d, 0xa3, 0x83, 0xc5, 0x27, 0xce, 0x11, 0xbf, 0xd4, 0x00, 0x40, 0x05, 0x13, 0xbb, 0x57}
)

// Document is an opened SolidWorks file: its container, kind, and version. It abstracts over the
// two on-disk containers — CFBF/OLE (format A) via cf, and the SolidWorks-2026 log-structured store
// (format B) via fb (stream name -> decoded bytes) — so callers decode one stream set either way.
type Document struct {
	cf      *olecf.File
	fb      map[string][]byte
	Type    DocType
	Version int // internal build version parsed from the _MO_VERSION_NNNN storage (e.g. 3400)
}

// Open parses a .SLDPRT / .SLDASM in either container format.
func Open(data []byte) (*Document, error) {
	if isFormatB(data) {
		return openFormatB(data)
	}
	cf, err := olecf.Open(data)
	if err != nil {
		return nil, err
	}
	return &Document{cf: cf, Type: docTypeOf(cf.RootCLSID()), Version: versionOf(cf.Streams())}, nil
}

func openFormatB(data []byte) (*Document, error) {
	streams, err := parseFormatB(data)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(streams))
	for n := range streams {
		names = append(names, n)
	}
	return &Document{fb: streams, Type: Part, Version: versionOf(names)}, nil
}

// Stream returns the raw bytes of a named stream (e.g. "Contents/Config-0"), from whichever
// container backs the document. In format B the bytes are already inflated (see formatb.go).
func (d *Document) Stream(path string) ([]byte, error) {
	if d.fb != nil {
		b, ok := d.fb[path]
		if !ok {
			return nil, errors.New("sldprt: stream not found: " + path)
		}
		return b, nil
	}
	if d.cf == nil {
		return nil, errors.New("sldprt: document not open")
	}
	return d.cf.Read(path)
}

// Streams lists every stream path in the document.
func (d *Document) Streams() []string {
	if d.fb != nil {
		names := make([]string, 0, len(d.fb))
		for n := range d.fb {
			names = append(names, n)
		}
		return names
	}
	return d.cf.Streams()
}

func docTypeOf(clsid [16]byte) DocType {
	switch clsid {
	case clsidPart:
		return Part
	case clsidAssembly:
		return Assembly
	case clsidDrawing:
		return Drawing
	default:
		return Unknown
	}
}
