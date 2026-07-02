// SPDX-License-Identifier: GPL-2.0-only

package compdef

import "oblikovati.org/model/doc"

// testContentFactories mirrors model/contentset.Default for this package's tests —
// compdef cannot import contentset (it imports compdef back); the workspace gets the
// same live part/assembly content the composition root wires (#1617). Drawing content
// stays stubbed: these tests never open drawings.
func testContentFactories() doc.ContentFactories {
	return doc.ContentFactories{
		doc.Part:     func() doc.Content { return NewPartComponentDefinition() },
		doc.Assembly: func() doc.Content { return NewAssemblyComponentDefinition() },
	}
}
