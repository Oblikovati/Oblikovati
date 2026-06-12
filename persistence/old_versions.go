// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Old-version retention (M03-F09, #610): when enabled, every save first copies
// the prior file into an OldVersions sibling directory — <name>.<n>.obk with
// .1 the most recent — pruned to the configured count. The prior file is
// COPIED, never moved, so an interrupted save still leaves the original
// untouched (the atomic-save guarantee is preserved).

// retainOldVersion archives the current file at path before it is overwritten,
// keeping at most keep prior versions. keep<=0 disables retention; a missing
// prior file (first save) is a no-op.
func retainOldVersion(path string, keep int) error {
	if keep <= 0 {
		return nil
	}
	prior, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	dir := filepath.Join(filepath.Dir(path), "OldVersions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := rotateOldVersions(dir, filepath.Base(path), keep); err != nil {
		return err
	}
	return os.WriteFile(oldVersionPath(dir, filepath.Base(path), 1), prior, 0o644)
}

// rotateOldVersions shifts <name>.<n>.obk up by one and prunes beyond keep.
func rotateOldVersions(dir, base string, keep int) error {
	_ = os.Remove(oldVersionPath(dir, base, keep)) // the oldest retained slot falls off
	for n := keep - 1; n >= 1; n-- {
		from, to := oldVersionPath(dir, base, n), oldVersionPath(dir, base, n+1)
		if _, err := os.Stat(from); err != nil {
			continue
		}
		if err := os.Rename(from, to); err != nil {
			return fmt.Errorf("rotate %q: %w", from, err)
		}
	}
	return nil
}

// oldVersionPath names one retained version: <dir>/<stem>.<n>.obk.
func oldVersionPath(dir, base string, n int) string {
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	return filepath.Join(dir, fmt.Sprintf("%s.%d%s", stem, n, filepath.Ext(base)))
}
