// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"fmt"
	"strings"
)

// ParseDocumentType maps a kind name to its [DocumentType]. It is the inverse of
// [DocumentType.String] and the single shared parser for the in-process drivers
// (the add-in router's documents.create and the oblikovati-cli fixture commands),
// so the name↔kind table lives in exactly one place. The match is case-insensitive
// and trims surrounding whitespace.
//
//	t, err := doc.ParseDocumentType("Part") // t == doc.Part
func ParseDocumentType(name string) (DocumentType, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "part":
		return Part, nil
	case "assembly":
		return Assembly, nil
	case "drawing":
		return Drawing, nil
	case "presentation":
		return Presentation, nil
	default:
		return Unknown, fmt.Errorf("doc: unknown document type %q (want part|assembly|drawing|presentation)", name)
	}
}
