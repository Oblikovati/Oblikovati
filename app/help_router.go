// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	"os"
	"strings"
)

// Help routing (M05-F14, #621): add-ins register help sources (URL prefixes or
// local directories) and open topics through the host, so their help behaves like
// product help. An in-process interceptor may handle a request first (the
// HelpEvents veto seam for first-party code); everything else resolves against the
// source's base and opens via the platform URL opener.

// defaultHelpBase is the host's own documentation, the "" source.
const defaultHelpBase = "https://github.com/Oblikovati/Oblikovati/tree/develop/architecture"

// HelpInterceptor may handle a help request before the host routes it: return
// true to consume it (the OnHelp veto/override hook for in-process code).
type HelpInterceptor func(source, topic string) bool

// SetHelpInterceptor installs the before-help hook (nil removes it).
func (s *Session) SetHelpInterceptor(fn HelpInterceptor) { s.helpInterceptor = fn }

// RegisterHelpContext declares a help source. Base must be an http(s) URL prefix
// or an absolute local directory.
func (s *Session) RegisterHelpContext(source, base string) error {
	if source == "" || base == "" {
		return fmt.Errorf("app: help context needs source and base, got source=%q base=%q", source, base)
	}
	if !isHelpBase(base) {
		return fmt.Errorf("app: help base %q is neither an http(s) URL nor an absolute path", base)
	}
	s.helpSources[source] = base
	return nil
}

// HelpPath returns a source's base ("" resolves to the host documentation).
func (s *Session) HelpPath(source string) (string, error) {
	if source == "" {
		return defaultHelpBase, nil
	}
	base, ok := s.helpSources[source]
	if !ok {
		return "", fmt.Errorf("app: no help source %q (register it first)", source)
	}
	return base, nil
}

// DisplayHelpTopic opens a topic of a registered source: the interceptor first,
// then the resolved target through the platform opener. A local-file base opens
// as a file:// URL via the same seam.
func (s *Session) DisplayHelpTopic(source, topic string) error {
	if s.helpInterceptor != nil && s.helpInterceptor(source, topic) {
		return nil
	}
	base, err := s.HelpPath(source)
	if err != nil {
		return err
	}
	return s.openHelpTarget(base, topic)
}

// openHelpTarget resolves base+topic and routes it through the URL opener.
func (s *Session) openHelpTarget(base, topic string) error {
	target := base
	if topic != "" {
		target = strings.TrimRight(base, "/") + "/" + strings.TrimLeft(topic, "/")
	}
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "file://" + target
	}
	if s.urlOpener == nil {
		return fmt.Errorf("app: no URL opener configured to show help %q", target)
	}
	return s.urlOpener.OpenURL(target)
}

// isHelpBase accepts http(s) URLs and absolute paths.
func isHelpBase(base string) bool {
	return strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") ||
		strings.HasPrefix(base, "/")
}

// Locale returns the host locale as a BCP-47 tag, derived from the process
// environment (LC_ALL/LANG), falling back to en-US.
func Locale() string {
	for _, name := range []string{"LC_ALL", "LANG"} {
		if raw := os.Getenv(name); raw != "" && raw != "C" && raw != "POSIX" {
			return normalizeLocale(raw)
		}
	}
	return "en-US"
}

// normalizeLocale turns "en_US.UTF-8" into "en-US".
func normalizeLocale(raw string) string {
	if i := strings.IndexAny(raw, ".@"); i >= 0 {
		raw = raw[:i]
	}
	return strings.ReplaceAll(raw, "_", "-")
}
