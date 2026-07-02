// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoStaleSourcePathsInLivingDocs is the D4 guard (#1661): the GPL application moved from
// /source to the repo root, and LIVING documentation must not resurrect the old path — every
// stale reference sends a reader to a directory that does not exist. Point-in-time records are
// exempt when they say so: architecture/history and architecture/audits are archival by
// location, and an ADR carrying the repo-root-migration banner (or a historical-snapshot
// banner) is deliberately preserved wording.
func TestNoStaleSourcePathsInLivingDocs(t *testing.T) {
	var offenders []string
	err := filepath.WalkDir("../architecture", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case "history", "audits":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".md") {
			return nil
		}
		src, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		text := string(src)
		if strings.Contains(text, "repo-root migration") || strings.Contains(text, "Historical snapshot") {
			return nil // bannered point-in-time record: preserved wording is deliberate
		}
		for n, line := range strings.Split(text, "\n") {
			if strings.Contains(line, "/source") {
				offenders = append(offenders, filepath.Clean(path)+":"+itoaGuard(n+1)+": "+strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking architecture docs: %v", err)
	}
	if len(offenders) > 0 {
		t.Fatalf("stale /source path(s) in living docs — fix the path or banner the record (D4 #1661):\n%s",
			strings.Join(offenders, "\n"))
	}
}

// itoaGuard is a tiny dependency-free int→string.
func itoaGuard(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
