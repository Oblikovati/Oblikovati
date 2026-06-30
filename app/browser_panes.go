// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati.org/api/wire"
	"oblikovati.org/event"
)

// Add-in browser panes (M05-F03, #256): named trees an add-in declares through
// browser.setPane, shown by the head beside the built-in Model pane. The pane spec
// is declared bulk state ([wire.BrowserPaneSpec] is already pure data, so it IS the
// model — re-declaring it here would only duplicate the shape).

// BrowserPaneNodeActivated fires (After) when the user interacts with an add-in
// pane node; the events relay forwards it as a wire browser.node push event.
// Gesture is one of the BrowserGesture* constants.
type BrowserPaneNodeActivated struct {
	Pane     string
	Node     string
	Gesture  string
	MenuItem string // chosen context-menu item id (Gesture BrowserGestureMenu)
}

// EventID implements event.Event.
func (BrowserPaneNodeActivated) EventID() event.TypeID { return tidBrowserPaneNode }

// The gestures a browser pane node reports.
const (
	BrowserGestureSelect   = "select"
	BrowserGestureDouble   = "double"
	BrowserGestureExpand   = "expand"
	BrowserGestureCollapse = "collapse"
	BrowserGestureMenu     = "menu"
)

// AddInBrowserPanes stores the declared panes in creation order.
type AddInBrowserPanes struct {
	order []string
	panes map[string]wire.BrowserPaneSpec
}

// NewAddInBrowserPanes returns an empty pane store.
func NewAddInBrowserPanes() *AddInBrowserPanes {
	return &AddInBrowserPanes{panes: map[string]wire.BrowserPaneSpec{}}
}

// Set creates the pane or replaces its whole tree.
func (p *AddInBrowserPanes) Set(spec wire.BrowserPaneSpec) error {
	if spec.ID == "" || spec.Title == "" {
		return fmt.Errorf("app: browser pane needs id and title, got id=%q title=%q", spec.ID, spec.Title)
	}
	if _, exists := p.panes[spec.ID]; !exists {
		p.order = append(p.order, spec.ID)
	}
	p.panes[spec.ID] = spec
	return nil
}

// Delete removes a pane.
func (p *AddInBrowserPanes) Delete(id string) error {
	if _, ok := p.panes[id]; !ok {
		return fmt.Errorf("app: no browser pane %q", id)
	}
	delete(p.panes, id)
	for i, x := range p.order {
		if x == id {
			p.order = append(p.order[:i], p.order[i+1:]...)
			break
		}
	}
	return nil
}

// List returns the panes in creation order.
func (p *AddInBrowserPanes) List() []wire.BrowserPaneSpec {
	out := make([]wire.BrowserPaneSpec, len(p.order))
	for i, id := range p.order {
		out[i] = p.panes[id]
	}
	return out
}

// BrowserPanes returns the add-in browser-pane store.
func (s *Session) BrowserPanes() *AddInBrowserPanes { return s.browserPanes }

// ActivateBrowserPaneNode reports a user gesture on an add-in pane node — the head
// calls it from the browser UI; the event reaches the owning add-in through the
// events relay. Unknown panes error so a stale head click is loud, not silent.
func (s *Session) ActivateBrowserPaneNode(pane, node, gesture string) error {
	if _, ok := s.browserPanes.panes[pane]; !ok {
		return fmt.Errorf("app: no browser pane %q", pane)
	}
	switch gesture {
	case BrowserGestureSelect, BrowserGestureDouble, BrowserGestureExpand, BrowserGestureCollapse:
	default:
		return fmt.Errorf("app: unknown browser gesture %q (select/double/expand/collapse)", gesture)
	}
	event.Emit(s.bus, event.After, BrowserPaneNodeActivated{Pane: pane, Node: node, Gesture: gesture})
	return nil
}

// ActivateBrowserPaneNodeMenu reports that the user chose context-menu item menuItem on an
// add-in pane node — the head calls it from the node's right-click menu; the owning add-in
// receives it as a browser.node event with Gesture "menu" and the item id. Unknown panes error.
func (s *Session) ActivateBrowserPaneNodeMenu(pane, node, menuItem string) error {
	if _, ok := s.browserPanes.panes[pane]; !ok {
		return fmt.Errorf("app: no browser pane %q", pane)
	}
	event.Emit(s.bus, event.After, BrowserPaneNodeActivated{
		Pane: pane, Node: node, Gesture: BrowserGestureMenu, MenuItem: menuItem,
	})
	return nil
}
