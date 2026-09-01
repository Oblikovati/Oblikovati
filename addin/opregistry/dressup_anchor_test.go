// SPDX-License-Identifier: GPL-2.0-only

package opregistry

import (
	"testing"

	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
)

// TestFilletViaWireAPICapturesAnchors is the end-to-end proof of ADR-0043 P6b for the path that
// was the gap: authoring a fillet through the public wire operation — edge reference keys only,
// never the GUI's resolved EdgeHandles — must still capture each picked edge's mint-time anchor,
// so a wire/MCP-authored fillet carries the geometric-recovery witness. Before the fix this map
// was always empty for API-authored dress-ups and the geometric tier was silently unreachable.
func TestFilletViaWireAPICapturesAnchors(t *testing.T) {
	t.Parallel()
	s, edge, _ := extrudedSolid(t)

	if _, err := applyMap(t, s, "fillet", map[string]any{"edgeRefs": []string{edge}, "radius": "1 mm"}); err != nil {
		t.Fatalf("fillet via wire API: %v", err)
	}

	fdef := lastFilletDef(t, s.ActiveDocument().Content().(*compdef.PartComponentDefinition))
	if len(fdef.EdgeAnchors) != 1 {
		t.Fatalf("wire-API fillet captured %d mint-time anchors, want 1 (EdgeAnchors=%v)", len(fdef.EdgeAnchors), fdef.EdgeAnchors)
	}
}

// lastFilletDef returns the most recently added fillet definition in the part.
func lastFilletDef(t *testing.T, def *compdef.PartComponentDefinition) *feature.FilletDefinition {
	t.Helper()
	feats := def.Features()
	for i := feats.Count() - 1; i >= 0; i-- {
		if ff, ok := feats.Item(i).Definition().(*feature.FilletFeature); ok {
			return ff.Definition()
		}
	}
	t.Fatal("no fillet feature found in part")
	return nil
}
