// SPDX-License-Identifier: GPL-2.0-only

// Package olecf is a minimal, read-only reader for the Microsoft Compound File
// Binary format (CFBF / OLE2), the container used by Autodesk Inventor .ipt files.
// It enumerates and reads named streams; it does not write. Reference: [MS-CFB].
package olecf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"unicode/utf16"
)

const (
	sigLen        = 8
	dirEntrySize  = 128
	endOfChain    = 0xFFFFFFFE
	freeSect      = 0xFFFFFFFF
	maxObjectType = 5
)

var signature = [sigLen]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}

// File is a parsed compound file exposing its streams by path.
type File struct {
	data          []byte
	sectorSize    int
	miniSectorLen int
	miniCutoff    uint32
	fat           []uint32
	miniFAT       []uint32
	entries       []dirEntry
	miniStream    []byte
	paths         map[string]int // stream path -> directory entry index
}

type dirEntry struct {
	name        string
	objectType  byte
	left, right int32
	child       int32
	startSector uint32
	size        uint64
	clsid       [16]byte
}

// Open parses the compound-file bytes. The slice is retained (not copied).
func Open(data []byte) (*File, error) {
	if len(data) < 512 || [sigLen]byte(data[:sigLen]) != signature {
		return nil, errors.New("olecf: not a compound file (bad signature)")
	}
	f := &File{data: data}
	if err := f.readHeaderAndFAT(); err != nil {
		return nil, err
	}
	if err := f.readDirectory(); err != nil {
		return nil, err
	}
	f.loadMiniStream()
	f.paths = map[string]int{}
	f.walk(f.entries[0].child, "")
	return f, nil
}

// Streams returns the "/"-joined path of every stream, sorted arbitrarily.
func (f *File) Streams() []string {
	out := make([]string, 0, len(f.paths))
	for p := range f.paths {
		out = append(out, p)
	}
	return out
}

// RootCLSID returns the CLSID of the root storage (identifies the document type).
func (f *File) RootCLSID() [16]byte { return f.entries[0].clsid }

// Read returns the decompressed bytes of the stream at path (e.g. "RSeStorage/RSeDb").
func (f *File) Read(path string) ([]byte, error) {
	idx, ok := f.paths[path]
	if !ok {
		return nil, fmt.Errorf("olecf: no stream %q", path)
	}
	e := f.entries[idx]
	if e.size >= uint64(f.miniCutoff) {
		return f.readChain(f.fat, f.sectorSize, e.startSector, f.data, e.size, true), nil
	}
	return f.readChain(f.miniFAT, f.miniSectorLen, e.startSector, f.miniStream, e.size, false), nil
}

func (f *File) readHeaderAndFAT() error {
	d := f.data
	f.sectorSize = 1 << binary.LittleEndian.Uint16(d[30:])
	f.miniSectorLen = 1 << binary.LittleEndian.Uint16(d[32:])
	f.miniCutoff = binary.LittleEndian.Uint32(d[56:])
	if f.sectorSize <= 0 || f.miniSectorLen <= 0 {
		return errors.New("olecf: invalid sector sizing")
	}
	fatSectors := f.collectFATSectorList()
	f.fat = make([]uint32, 0, len(fatSectors)*f.sectorSize/4)
	for _, s := range fatSectors {
		off := f.sectorOffset(s)
		for i := 0; i+4 <= f.sectorSize; i += 4 {
			f.fat = append(f.fat, binary.LittleEndian.Uint32(d[off+i:]))
		}
	}
	f.buildMiniFAT()
	return nil
}

// collectFATSectorList returns the sector indices holding the FAT, from the 109
// header DIFAT entries plus any chained DIFAT sectors.
func (f *File) collectFATSectorList() []uint32 {
	d := f.data
	var list []uint32
	for i := 0; i < 109; i++ {
		s := binary.LittleEndian.Uint32(d[76+i*4:])
		if s == freeSect {
			continue
		}
		list = append(list, s)
	}
	difatSect := binary.LittleEndian.Uint32(d[68:])
	for difatSect != endOfChain && difatSect != freeSect {
		off := f.sectorOffset(difatSect)
		n := f.sectorSize/4 - 1
		for i := 0; i < n; i++ {
			s := binary.LittleEndian.Uint32(d[off+i*4:])
			if s != freeSect {
				list = append(list, s)
			}
		}
		difatSect = binary.LittleEndian.Uint32(d[off+n*4:])
	}
	return list
}

func (f *File) buildMiniFAT() {
	sect := binary.LittleEndian.Uint32(f.data[60:])
	for sect != endOfChain && sect != freeSect {
		off := f.sectorOffset(sect)
		for i := 0; i+4 <= f.sectorSize; i += 4 {
			f.miniFAT = append(f.miniFAT, binary.LittleEndian.Uint32(f.data[off+i:]))
		}
		sect = f.next(f.fat, sect)
	}
}

func (f *File) readDirectory() error {
	dirSect := binary.LittleEndian.Uint32(f.data[48:])
	raw := f.readChain(f.fat, f.sectorSize, dirSect, f.data, 0, true)
	for off := 0; off+dirEntrySize <= len(raw); off += dirEntrySize {
		e, ok := parseDirEntry(raw[off : off+dirEntrySize])
		if ok {
			f.entries = append(f.entries, e)
		} else {
			f.entries = append(f.entries, dirEntry{objectType: 0, left: -1, right: -1, child: -1})
		}
	}
	if len(f.entries) == 0 || f.entries[0].objectType != maxObjectType {
		return errors.New("olecf: missing root directory entry")
	}
	return nil
}

func parseDirEntry(b []byte) (dirEntry, bool) {
	nameLen := int(binary.LittleEndian.Uint16(b[64:]))
	typ := b[66]
	if typ == 0 || nameLen < 2 {
		return dirEntry{}, false
	}
	u16 := make([]uint16, 0, nameLen/2)
	for i := 0; i+1 < nameLen; i += 2 {
		u16 = append(u16, binary.LittleEndian.Uint16(b[i:]))
	}
	name := strings.TrimRight(string(utf16.Decode(u16)), "\x00")
	e := dirEntry{
		name:        name,
		objectType:  typ,
		left:        int32(binary.LittleEndian.Uint32(b[68:])),
		right:       int32(binary.LittleEndian.Uint32(b[72:])),
		child:       int32(binary.LittleEndian.Uint32(b[76:])),
		startSector: binary.LittleEndian.Uint32(b[116:]),
		size:        binary.LittleEndian.Uint64(b[120:]),
	}
	copy(e.clsid[:], b[80:96])
	return e, true
}

func (f *File) loadMiniStream() {
	root := f.entries[0]
	f.miniStream = f.readChain(f.fat, f.sectorSize, root.startSector, f.data, root.size, true)
}

// walk traverses the red-black directory tree, recording stream paths. Storages
// (objectType 1) contribute a path prefix; streams (objectType 2) become entries.
func (f *File) walk(id int32, prefix string) {
	if id < 0 || int(id) >= len(f.entries) {
		return
	}
	e := f.entries[id]
	f.walk(e.left, prefix)
	name := prefix + e.name
	switch e.objectType {
	case 2:
		f.paths[name] = int(id)
	case 1:
		f.walk(e.child, name+"/")
	}
	f.walk(e.right, prefix)
}

// readChain follows a sector chain in fat and returns limit bytes (0 = whole chain).
// mainSectors selects the file-sector offset (header-shifted) vs a flat mini-stream offset.
func (f *File) readChain(fat []uint32, unit int, start uint32, store []byte, limit uint64, mainSectors bool) []byte {
	var out []byte
	sect := start
	for sect != endOfChain && sect != freeSect {
		off := int(sect) * unit
		if mainSectors {
			off = f.sectorOffset(sect)
		}
		if off+unit > len(store) {
			break
		}
		out = append(out, store[off:off+unit]...)
		sect = f.next(fat, sect)
	}
	if limit > 0 && uint64(len(out)) > limit {
		out = out[:limit]
	}
	return out
}

func (f *File) next(fat []uint32, sect uint32) uint32 {
	if int(sect) >= len(fat) {
		return endOfChain
	}
	return fat[sect]
}

// sectorOffset is the byte offset of FAT sector n; the header occupies one sector.
func (f *File) sectorOffset(n uint32) int { return (int(n) + 1) * f.sectorSize }
