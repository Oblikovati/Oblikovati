// SPDX-License-Identifier: GPL-2.0-only

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// reloadPollInterval is how often the add-ins directory is scanned for a replaced
// library. A change must persist across one interval before it counts, so a
// half-written copy (make install writes ~13 MB) is never acted on.
const reloadPollInterval = 1 * time.Second

// Why auto-RESTART instead of in-process hot-reload: an add-in is a Go c-shared
// library, and a Go runtime cannot be unloaded (dlclose crashes its still-running
// sysmon/GC threads) nor safely loaded a second time alongside the host's runtime.
// Both were tried and both crash the host. The safe way to pick up a rebuilt add-in
// without a manual restart is to detect the change, exit cleanly, and let a
// supervisor (make run-watch) relaunch the process — which reloads the new .so at
// startup the normal way. Opt in with OBK_ADDIN_AUTORESTART=1 (set by run-watch);
// plain `make run` never auto-exits.

// fileSig identifies a file version cheaply (size + mtime).
type fileSig struct {
	size    int64
	modUnix int64
}

// watchAddIns polls the add-ins directory and, when a library is replaced (its
// signature changes and then settles for one interval), sets the changed flag the
// frame loop checks. Runs until stop closes h.watchDone.
func (h *addInHost) watchAddIns() {
	baseline := scanLibs(h.dir) // existing versions are the baseline; only later changes count
	pending := map[string]fileSig{}
	ticker := time.NewTicker(reloadPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-h.watchDone:
			return
		case <-ticker.C:
			for path, sig := range scanLibs(h.dir) {
				if baseline[path] == sig {
					continue
				}
				if pending[path] == sig { // stable for one interval → a real change
					fmt.Fprintf(os.Stderr, "add-in changed on disk (%s); exiting for supervisor restart\n", filepath.Base(path))
					atomic.StoreInt32(&h.changed, 1)
					return
				}
				pending[path] = sig
			}
		}
	}
}

// addInChanged reports whether a watched add-in library was replaced on disk, so the
// frame loop can exit for the supervisor to relaunch with the new version.
func (h *addInHost) addInChanged() bool { return atomic.LoadInt32(&h.changed) == 1 }

// scanLibs returns the signature of every shared library in dir, keyed by path.
func scanLibs(dir string) map[string]fileSig {
	out := map[string]fileSig{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || !isSharedLibName(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out[filepath.Join(dir, e.Name())] = fileSig{size: info.Size(), modUnix: info.ModTime().UnixNano()}
	}
	return out
}

func isSharedLibName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".so", ".dylib", ".dll":
		return true
	default:
		return false
	}
}
