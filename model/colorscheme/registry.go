// SPDX-License-Identifier: GPL-2.0-only

package colorscheme

import (
	"fmt"

	"oblikovati.org/api/types"
)

// Registry is the application's set of color schemes with one active selection and an
// application-wide background type — the equivalent of the Color tab of the application
// options. It is not safe for concurrent use; the app touches it on the session goroutine.
type Registry struct {
	schemes    []Scheme
	activeName string
	background types.BackgroundTypeEnum
}

// NewRegistry builds the registry seeded with the built-in gallery, the first scheme active
// and its background type adopted as the application-wide default.
func NewRegistry() *Registry {
	schemes := builtinSchemes()
	return &Registry{schemes: schemes, activeName: schemes[0].Name, background: schemes[0].BackgroundType}
}

// Schemes returns the gallery in picker order (a copy of the backing slice is unnecessary —
// callers must not mutate the entries).
func (r *Registry) Schemes() []Scheme { return r.schemes }

// Active returns the currently active scheme.
func (r *Registry) Active() Scheme {
	s, _ := r.find(r.activeName)
	return s
}

// ActiveName returns the active scheme's name.
func (r *Registry) ActiveName() string { return r.activeName }

// SetActive makes the named scheme active and adopts its background type, returning an error
// that names the scheme (and lists the valid names) when it is absent.
func (r *Registry) SetActive(name string) error {
	s, ok := r.find(name)
	if !ok {
		return fmt.Errorf("colorscheme: no scheme named %q; have %v", name, r.names())
	}
	r.activeName = s.Name
	r.background = s.BackgroundType
	return nil
}

// BackgroundType returns the application-wide viewport background type.
func (r *Registry) BackgroundType() types.BackgroundTypeEnum { return r.background }

// SetBackgroundType overrides the application-wide viewport background type, erroring on an
// undefined value.
func (r *Registry) SetBackgroundType(t types.BackgroundTypeEnum) error {
	if !t.IsValid() {
		return fmt.Errorf("colorscheme: background type %d is not a defined BackgroundTypeEnum", int32(t))
	}
	r.background = t
	return nil
}

// find returns the scheme with the given name (case-sensitive) and whether it exists.
func (r *Registry) find(name string) (Scheme, bool) {
	for _, s := range r.schemes {
		if s.Name == name {
			return s, true
		}
	}
	return Scheme{}, false
}

// names lists the scheme names for error messages.
func (r *Registry) names() []string {
	out := make([]string, len(r.schemes))
	for i, s := range r.schemes {
		out[i] = s.Name
	}
	return out
}
