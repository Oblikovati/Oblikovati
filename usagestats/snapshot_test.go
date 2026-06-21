// SPDX-License-Identifier: GPL-2.0-only

package usagestats

import (
	"encoding/json"
	"sort"
	"testing"
)

// TestSnapshotWireKeysMatchService pins the JSON field names. The stats.oblikovati.org
// service unmarshals into its own internal/report.Snapshot with these exact keys; if a key
// here drifts, the service silently drops the field. This guard plays the role the
// round-trip test plays for the bug-report Payload (no shared module across the two repos).
func TestSnapshotWireKeysMatchService(t *testing.T) {
	b, err := json.Marshal(Snapshot{})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := make([]string, 0, len(m))
	for k := range m {
		got = append(got, k)
	}
	sort.Strings(got)
	want := []string{
		"addIns", "appVersion", "arch", "cpu", "cpuCores", "gpu", "machineUUID",
		"os", "ramBytes", "reportedAt", "storageBytes", "vulkanVersion",
	}
	if len(got) != len(want) {
		t.Fatalf("snapshot keys = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("snapshot keys = %v, want %v", got, want)
		}
	}
}

func TestAddInWireKeys(t *testing.T) {
	b, _ := json.Marshal(AddIn{ID: "x", Version: "1"})
	if string(b) != `{"id":"x","version":"1"}` {
		t.Fatalf("AddIn JSON = %s, want {\"id\":\"x\",\"version\":\"1\"}", b)
	}
}
