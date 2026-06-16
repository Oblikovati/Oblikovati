// SPDX-License-Identifier: GPL-2.0-only

package doc

import "sort"

// NamedView is one saved named view: a label and the exact camera frame it restores
// (M16-F03 #404). The frame is a Fixed-Distance [ViewHome] (FitToView false) so a restore
// returns to the exact framing the user captured.
type NamedView struct {
	Name string
	Home ViewHome
}

// CaptureNamed saves frame under name, replacing any existing named view of that name.
func (vs *DocumentViews) CaptureNamed(name string, frame ViewHome) {
	if vs.named == nil {
		vs.named = map[string]ViewHome{}
	}
	vs.named[name] = frame
}

// NamedView returns the saved frame for name and whether it exists.
func (vs *DocumentViews) NamedView(name string) (ViewHome, bool) {
	h, ok := vs.named[name]
	return h, ok
}

// DeleteNamed removes a named view, reporting whether it existed.
func (vs *DocumentViews) DeleteNamed(name string) bool {
	if _, ok := vs.named[name]; !ok {
		return false
	}
	delete(vs.named, name)
	return true
}

// NamedViews returns every saved named view, sorted by name for a stable enumeration.
func (vs *DocumentViews) NamedViews() []NamedView {
	out := make([]NamedView, 0, len(vs.named))
	for name, h := range vs.named {
		out = append(out, NamedView{Name: name, Home: h})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
