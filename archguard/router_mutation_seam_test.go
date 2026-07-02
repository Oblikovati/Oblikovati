// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The router is a wire adapter (ADR-0018): handlers decode a DTO, call an
// app.Session verb, and encode the result. Reads may walk the model directly
// (queries carry no invariants to drift), but a handler that orchestrates its
// own mutation — recomputing and recording undo itself instead of delegating —
// re-creates the divergence audited as B1 (#1612): a rule enforced in one
// driver and forgotten in the other. This guard holds the amended rule from
// addin/router/router.go's package doc: files that still self-orchestrate are
// pinned in a shrink-only allowlist; a new one is a red build.

// routerSelfOrchestrationMarkers are the calls only an app.Session verb should
// make on a mutation path: the recompute sequencing and the undo record.
var routerSelfOrchestrationMarkers = regexp.MustCompile(
	`RecordActiveEdit\(|RecomputeAfterChange\(|RecomputeFeatures\(|\.Recompute\(\)`)

// pendingRouterSeamMigrations pins the router files that still orchestrate
// mutations in-handler, mapped to why they are pending. Shrink-only: migrate a
// file to Session verbs, delete its row. Never add a row for new code — write
// the Session verb instead (#1612).
var pendingRouterSeamMigrations = map[string]string{
	"bodies.go":               "body rename/delete not yet behind Session verbs",
	"assembly_derive.go":      "derive/export program edits not yet behind Session verbs",
	"assembly_replication.go": "replication rebuild not yet behind a Session verb",

	"document_end_of_part.go": "end-of-part marker move not yet behind a Session verb",
	"documents_update.go":     "document metadata update not yet behind a Session verb",
	"drawing_views.go":        "drawing view edits not yet behind Session verbs",
	"parameter_settings.go":   "settings/import sweep not yet behind Session verbs",
	"parameters.go":           "parameters.set fast path not yet behind a Session verb",
	"parameters_detail.go":    "update/tolerance/value-list edits not yet behind Session verbs (delete IS migrated)",
	"sheet_metal.go":          "sheet-metal rule edits not yet behind Session verbs",
	"work_planes.go":          "work-geometry edits not yet behind Session verbs",
}

// centralSeamFiles hold the router's own post-success machinery (commitMutation
// calls RecordActiveEdit for every MutatingMethod) — the seam itself, exempt.
var centralSeamFiles = map[string]bool{"router.go": true}

// TestRouterHandlersDelegateMutations fails when a router file outside the
// allowlist self-orchestrates a mutation, and when an allowlist row went stale.
func TestRouterHandlersDelegateMutations(t *testing.T) {
	dirty := routerFilesWithOrchestrationMarkers(t)
	for file := range dirty {
		if centralSeamFiles[file] || pendingRouterSeamMigrations[file] != "" {
			continue
		}
		t.Errorf("addin/router/%s orchestrates a mutation in-handler (recompute/undo-record marker found) — "+
			"delegate to an app.Session verb so the UI and the wire share one seam (#1612, audit B1). "+
			"Do not add it to pendingRouterSeamMigrations.", file)
	}
	for file, why := range pendingRouterSeamMigrations {
		if !dirty[file] {
			t.Errorf("pendingRouterSeamMigrations[%q] (%s) is stale — the file delegates now; delete the entry.", file, why)
		}
	}
}

// routerFilesWithOrchestrationMarkers scans addin/router's non-test sources for
// the self-orchestration markers, returning the offending file names.
func routerFilesWithOrchestrationMarkers(t *testing.T) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir("../addin/router")
	if err != nil {
		t.Fatalf("reading ../addin/router: %v", err)
	}
	dirty := map[string]bool{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join("../addin/router", name))
		if err != nil {
			t.Fatalf("reading router source %s: %v", name, err)
		}
		if routerSelfOrchestrationMarkers.Match(src) {
			dirty[name] = true
		}
	}
	return dirty
}
