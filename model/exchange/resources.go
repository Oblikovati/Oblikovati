// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"bytes"
	"path/filepath"
	"unicode/utf8"

	"oblikovati.org/api/types"
	"oblikovati.org/model/doc"
)

// resourceFor packages an imported file's bytes as an embedded document resource (ADR-0031):
// a type tag from the format, an encoding chosen from the bytes (text stays diffable, binary is
// base64'd), and the original filename as display-only origin metadata.
func resourceFor(format types.ExchangeFormat, path string, data []byte) doc.Resource {
	return doc.Resource{
		Type:     resourceType(format),
		Encoding: encodingFor(data),
		Value:    data,
		Origin:   filepath.Base(path),
	}
}

// resourceType maps an exchange format to its resource type tag (ADR-0031, extensible).
func resourceType(format types.ExchangeFormat) string {
	switch format {
	case types.FormatSTEP:
		return "StepFile"
	case types.FormatSTL:
		return "StlFile"
	case types.FormatOBJ:
		return "ObjFile"
	case types.Format3MF:
		return "ThreeMFFile"
	default:
		return "File"
	}
}

// encodingFor picks a resource's storage encoding from its bytes rather than its type — an STL
// may be ASCII or binary under one tag (ADR-0031 §3). Valid UTF-8 with no NUL byte is stored
// verbatim (git-diffable); anything else is base64.
func encodingFor(data []byte) string {
	if utf8.Valid(data) && !bytes.ContainsRune(data, 0) {
		return doc.EncodingUTF8
	}
	return doc.EncodingBase64
}
