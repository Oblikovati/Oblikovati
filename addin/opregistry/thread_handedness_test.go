// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// Thread handedness over the wire (#1892). Until now the only way to ask for a left-hand thread
// was to spell it into the designation as "-LH", which a caller working from a thread table (where
// the designation is the table's key) has no way to do. The flag is the authorable form.

// lastThreadDef returns the definition of the part's most recently added thread.
func lastThreadDef(t *testing.T, s *app.Session) *feature.ThreadDefinition {
	t.Helper()
	fs := s.ActiveDocument().Content().(*compdef.PartComponentDefinition).Features()
	for i := fs.Count() - 1; i >= 0; i-- {
		if th, ok := fs.Item(i).Definition().(*feature.ThreadFeature); ok {
			return th.Definition()
		}
	}
	t.Fatal("no thread feature on the part")
	return nil
}

// TestThreadLeftHandedReachesTheDefinition: the geometry already honoured a left-hand spec, so
// what was missing was purely the way in. The test reads the flag back off the definition rather
// than trusting a request that merely succeeded.
func TestThreadLeftHandedReachesTheDefinition(t *testing.T) {
	s, _, face := extrudedSolid(t)
	if _, err := applyMap(t, s, "thread", map[string]any{
		"faceRef": face, "designation": "M8x1.25", "leftHanded": true,
	}); err != nil {
		t.Fatalf("thread with leftHanded: %v", err)
	}
	if !lastThreadDef(t, s).LeftHanded {
		t.Error("leftHanded did not reach the thread definition")
	}
}

// TestThreadDefaultsToRightHanded: the ordinary thread is right-handed, and the flag's absence
// must mean that rather than the zero value of a handedness name.
func TestThreadDefaultsToRightHanded(t *testing.T) {
	s, _, face := extrudedSolid(t)
	if _, err := applyMap(t, s, "thread", map[string]any{
		"faceRef": face, "designation": "M8x1.25",
	}); err != nil {
		t.Fatalf("thread: %v", err)
	}
	if lastThreadDef(t, s).LeftHanded {
		t.Error("a thread with no handedness given came out left-handed")
	}
}
