// SPDX-License-Identifier: GPL-2.0-only

package drawing

import "oblikovati.org/model/doc"

// testContentFactories mirrors model/contentset.Default for this package's tests —
// drawing cannot import contentset (it imports drawing back). Part/assembly content
// stays stubbed: these tests only exercise drawing recipes.
func testContentFactories() doc.ContentFactories {
	return doc.ContentFactories{
		doc.Drawing: func() doc.Content { return NewContent() },
	}
}
