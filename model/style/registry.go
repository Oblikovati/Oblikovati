// SPDX-License-Identifier: GPL-2.0-only

package style

import (
	"fmt"
	"sort"

	"oblikovati.org/api/types"
)

// Registry is the document's style registry: the ordered color styles and the loaded style
// libraries. It is not safe for concurrent use; the app touches it on the session goroutine.
type Registry struct {
	styles    []ColorStyle
	libraries []Library
}

// NewRegistry builds the registry seeded with the built-in local styles.
func NewRegistry() *Registry { return &Registry{styles: builtinStyles()} }

// Styles returns the color styles in registration order (callers must not mutate the entries).
func (r *Registry) Styles() []ColorStyle { return r.styles }

// ByName returns the named style and whether it exists.
func (r *Registry) ByName(name string) (ColorStyle, bool) {
	for _, s := range r.styles {
		if s.Name == name {
			return s, true
		}
	}
	return ColorStyle{}, false
}

// Set creates or updates a style by name, reporting whether it was newly added (vs. updated).
// A blank name is rejected so a style is always addressable.
func (r *Registry) Set(s ColorStyle) (added bool, err error) {
	if s.Name == "" {
		return false, fmt.Errorf("style: a color style must have a non-empty name")
	}
	for i := range r.styles {
		if r.styles[i].Name == s.Name {
			r.styles[i] = s
			return false, nil
		}
	}
	r.styles = append(r.styles, s)
	return true, nil
}

// Delete removes a style by name, erroring when it is absent.
func (r *Registry) Delete(name string) error {
	for i := range r.styles {
		if r.styles[i].Name == name {
			r.styles = append(r.styles[:i], r.styles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("style: no color style named %q", name)
}

// Libraries returns the loaded style libraries in cascade order (lowest Order first).
func (r *Registry) Libraries() []Library {
	out := append([]Library(nil), r.libraries...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}

// Import folds a loaded library into the cascade: it is recorded and each of its styles is
// merged as a Library-located style (not overriding an existing Local style of the same name,
// which wins the cascade). It returns the names of the styles that were newly added.
func (r *Registry) Import(lib Library) []string {
	lib.Order = len(r.libraries)
	r.libraries = append(r.libraries, lib)
	var added []string
	for _, s := range lib.Styles {
		if _, exists := r.ByName(s.Name); exists {
			continue // a local style of this name shadows the library one
		}
		s.Location = types.LibraryStyleLocation
		r.styles = append(r.styles, s)
		added = append(added, s.Name)
	}
	return added
}
