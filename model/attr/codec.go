// SPDX-License-Identifier: GPL-2.0-only

package attr

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

// Encode serializes the whole attribute manager to the bytes persisted in a
// package's attributes.bin (architecture core/05). Layout, all little-endian:
//
//	[nKeys u32]{ key }            key   = [len u32][refkey bytes] [nSets u32]{ set }
//	                             set   = str(name) [nAttrs u32]{ attr }
//	                             attr  = str(name) [valueType u8] value
//	                             str   = [len u32][bytes]
//
// Keys, sets and attributes are emitted in insertion order for stable output.
func (m *AttributeManager) Encode() []byte {
	buf := binary.LittleEndian.AppendUint32(nil, uint32(len(m.order)))
	for _, k := range m.order {
		buf = appendBytes(buf, []byte(k))
		buf = encodeSets(buf, m.byKey[k])
	}
	return buf
}

func encodeSets(buf []byte, ss *AttributeSets) []byte {
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(ss.order)))
	for _, set := range ss.Sets() {
		buf = appendString(buf, set.name)
		buf = binary.LittleEndian.AppendUint32(buf, uint32(len(set.order)))
		for _, a := range set.Attributes() {
			buf = appendString(buf, a.name)
			buf = encodeValue(buf, a.value)
		}
	}
	return buf
}

// valueTypePersistByte maps a value-type tag to its 1-byte on-disk code, and back. The codes are the
// original compact 0..4 sequence — NOT the api ValueType's frozen Inventor numbers — so persisted
// metadata stays byte-identical now that ValueType is an alias of the int32 api enum (#1501). A byte
// outside this set decodes to an invalid tag the decoder rejects (the old behaviour).
var valueTypePersistByte = map[ValueType]byte{Boolean: 0, Integer: 1, Double: 2, String: 3, Bytes: 4}

var valueTypeFromPersistByte = map[byte]ValueType{0: Boolean, 1: Integer, 2: Double, 3: String, 4: Bytes}

func encodeValue(buf []byte, v Value) []byte {
	buf = append(buf, valueTypePersistByte[v.typ])
	switch v.typ {
	case Boolean:
		var b byte
		if v.b {
			b = 1
		}
		return append(buf, b)
	case Integer:
		return binary.LittleEndian.AppendUint64(buf, uint64(v.i))
	case Double:
		return binary.LittleEndian.AppendUint64(buf, math.Float64bits(v.f))
	case String:
		return appendString(buf, v.s)
	default: // Bytes
		return appendBytes(buf, v.raw)
	}
}

func appendString(buf []byte, s string) []byte { return appendBytes(buf, []byte(s)) }

func appendBytes(buf, b []byte) []byte {
	buf = binary.LittleEndian.AppendUint32(buf, uint32(len(b)))
	return append(buf, b...)
}

// DecodeAttributes reconstructs a manager from bytes produced by [AttributeManager.Encode].
func DecodeAttributes(data []byte) (*AttributeManager, error) {
	r := bytes.NewReader(data)
	nKeys, err := readU32(r)
	if err != nil {
		return nil, err
	}
	m := NewAttributeManager()
	for i := uint32(0); i < nKeys; i++ {
		if err := decodeAnchor(r, m); err != nil {
			return nil, err
		}
	}
	return m, nil
}

func decodeAnchor(r *bytes.Reader, m *AttributeManager) error {
	keyBytes, err := readBytes(r)
	if err != nil {
		return err
	}
	ss := newAttributeSets()
	m.byKey[string(keyBytes)] = ss
	m.order = append(m.order, string(keyBytes))
	nSets, err := readU32(r)
	if err != nil {
		return err
	}
	for i := uint32(0); i < nSets; i++ {
		if err := decodeSet(r, ss); err != nil {
			return err
		}
	}
	return nil
}

func decodeSet(r *bytes.Reader, ss *AttributeSets) error {
	name, err := readString(r)
	if err != nil {
		return err
	}
	set := ss.Set(name)
	nAttrs, err := readU32(r)
	if err != nil {
		return err
	}
	for i := uint32(0); i < nAttrs; i++ {
		attrName, err := readString(r)
		if err != nil {
			return err
		}
		v, err := decodeValue(r)
		if err != nil {
			return err
		}
		set.Put(attrName, v)
	}
	return nil
}

func decodeValue(r *bytes.Reader) (Value, error) {
	t, err := r.ReadByte()
	if err != nil {
		return Value{}, fmt.Errorf("attr: truncated value type: %w", err)
	}
	vt, ok := valueTypeFromPersistByte[t]
	if !ok {
		return Value{}, fmt.Errorf("attr: unknown value type code %d", t)
	}
	return decodeTypedValue(vt, r)
}

func decodeTypedValue(t ValueType, r *bytes.Reader) (Value, error) {
	switch t {
	case Boolean:
		b, err := r.ReadByte()
		return BoolValue(b == 1), wrapTrunc(err, "boolean")
	case Integer:
		u, err := readU64(r)
		return IntValue(int64(u)), err
	case Double:
		u, err := readU64(r)
		return FloatValue(math.Float64frombits(u)), err
	case String:
		s, err := readString(r)
		return StringValue(s), err
	case Bytes:
		b, err := readBytes(r)
		return BytesValue(b), err
	default:
		return Value{}, fmt.Errorf("attr: unknown value type %d", t)
	}
}

func readString(r *bytes.Reader) (string, error) {
	b, err := readBytes(r)
	return string(b), err
}

func readBytes(r *bytes.Reader) ([]byte, error) {
	n, err := readU32(r)
	if err != nil {
		return nil, err
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, fmt.Errorf("attr: truncated blob of %d bytes: %w", n, err)
	}
	return b, nil
}

func readU32(r *bytes.Reader) (uint32, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, fmt.Errorf("attr: truncated uint32: %w", err)
	}
	return binary.LittleEndian.Uint32(b[:]), nil
}

func readU64(r *bytes.Reader) (uint64, error) {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, fmt.Errorf("attr: truncated uint64: %w", err)
	}
	return binary.LittleEndian.Uint64(b[:]), nil
}

func wrapTrunc(err error, what string) error {
	if err != nil {
		return fmt.Errorf("attr: truncated %s value: %w", what, err)
	}
	return nil
}
