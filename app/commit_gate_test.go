// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"
)

// TestCommitGateBlocksSickPartFeatureDraft is the #1626 regression for the activation
// seam: a PartFeatureTool whose draft previews sick must surface a non-empty
// CommitBlockedReason and s.OK() must refuse — before #1626 these tools reached the
// plain StartTool with no DraftFeature, silently skipping the gate (the #1521
// bypass-by-omission shape). Combine with the same body picked twice is a real sick
// configuration: the boolean rejects equal target/tool indices.
func TestCommitGateBlocksSickPartFeatureDraft(t *testing.T) {
	t.Parallel()
	s, block := newPartWithBlock(t, 4)
	tool := NewCombineTool()
	s.StartFeatureTool(tool)
	tool.Pick(s, BodyHandle{Body: block})
	tool.Pick(s, BodyHandle{Body: block}) // same body twice: target == tool → sick
	if !tool.CanCommit() {
		t.Fatal("combine should be nominally ready with two picks")
	}

	reason := s.CommitBlockedReason()
	if reason == "" {
		t.Fatal("a sick combine draft must yield a non-empty CommitBlockedReason (#1626)")
	}
	if !strings.Contains(reason, "combine") {
		t.Errorf("blocked reason %q should name the combine failure", reason)
	}

	before := activePartDef(t, s).Features().Count()
	if err := s.OK(); err == nil {
		t.Fatal("OK on a sick combine config should be refused")
	}
	if after := activePartDef(t, s).Features().Count(); after != before {
		t.Errorf("a sick config must not append a feature: count %d → %d", before, after)
	}
}
