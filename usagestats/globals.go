// SPDX-License-Identifier: GPL-2.0-only

package usagestats

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"oblikovati.org/persistence/userpaths"
)

// globalsName is the per-user file holding cross-session machine identity. It lives directly
// under the Oblikovati home (~/oblikovati/globals or %AppData%\oblikovati\globals).
const globalsName = "globals"

// machineUUIDKey is the globals key storing the anonymous per-machine identifier.
const machineUUIDKey = "machineUUID"

// MachineUUID returns the machine's anonymous telemetry identifier, generating and persisting
// one on first call. The id is random (not derived from hardware or user identity); it only
// lets the service keep one current row per machine. A read/parse problem returns a fresh
// (unpersisted) id rather than an error, so telemetry identity never blocks startup.
//
// Example:
//
//	id, _ := usagestats.MachineUUID()
func MachineUUID() (string, error) {
	path, err := globalsPath()
	if err != nil {
		return "", err
	}
	if id := readGlobalsValue(path, machineUUIDKey); id != "" {
		return id, nil
	}
	id, err := newUUID()
	if err != nil {
		return "", err
	}
	if err := writeGlobalsValue(path, machineUUIDKey, id); err != nil {
		return "", err
	}
	return id, nil
}

// globalsPath is <oblikovati-home>/globals. OBK_USER_GLOBALS_FILE overrides it (tests).
func globalsPath() (string, error) {
	if p := os.Getenv("OBK_USER_GLOBALS_FILE"); p != "" {
		return p, nil
	}
	home, err := userpaths.OblikovatiHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, globalsName), nil
}

// readGlobalsValue returns the value for key in the key=value globals file, or "" if the file
// or key is absent (a missing globals file on first run is expected, not an error).
func readGlobalsValue(path, key string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(b), "\n") {
		k, v, ok := strings.Cut(strings.TrimSpace(line), "=")
		if ok && k == key {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// writeGlobalsValue appends key=value to the globals file, creating the directory and file as
// needed. It does not rewrite existing keys — globals are write-once identity, and MachineUUID
// only writes when the key was absent.
func writeGlobalsValue(path, key, value string) error {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("usagestats: create globals dir %q: %w", dir, err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("usagestats: open globals %q: %w", path, err)
	}
	defer func() { _ = f.Close() }()
	if _, err := fmt.Fprintf(f, "%s=%s\n", key, value); err != nil {
		return fmt.Errorf("usagestats: write globals %q: %w", path, err)
	}
	return nil
}

// newUUID returns a random RFC-4122 version-4 UUID string from crypto/rand.
func newUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("usagestats: generate machine uuid: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
