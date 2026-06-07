// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"

	"oblikovati/api/types"
	"oblikovati/model/doc"
	"oblikovati/scene"
)

// SetViewLayout sets how the active document tiles its views and ensures it has enough
// views to fill the layout, creating any missing ones from the current view's camera so
// the choice takes visible effect immediately (e.g. "Four Views" shows four tiles even
// from a single view). It never removes views — switching back to a smaller layout keeps
// the extras for later. View-tab Windows panel.
func (s *Session) SetViewLayout(l types.ViewLayout) error {
	d := s.ActiveDocument()
	if d == nil {
		return ErrNoActiveDoc
	}
	vs := d.Views()
	vs.SetLayout(l)
	for vs.Count() < l.Tiles() {
		cur := vs.Active()
		nv := doc.DefaultView(fmt.Sprintf("View %d", vs.Count()+1))
		nv.Eye, nv.Target, nv.Up, nv.FOV, nv.Framed = cur.Eye, cur.Target, cur.Up, cur.FOV, cur.Framed
		vs.Add(nv)
	}
	return nil
}

// NewView adds a view to the active document (copying the current view's camera) and makes
// it active — the View-tab "New View" command.
func (s *Session) NewView() error {
	_, err := s.AddView(0, "", true)
	return err
}

// CloseActiveView removes the active document's active view (the last view is kept).
func (s *Session) CloseActiveView() error {
	d := s.ActiveDocument()
	if d == nil {
		return ErrNoActiveDoc
	}
	vs := d.Views()
	return vs.Close(vs.ActiveIndex())
}

// DocumentByID resolves a document by its session id; id 0 means the active document. It
// is the entry point for the document-addressed view/camera API (a Document field of 0
// targets the active document).
func (s *Session) DocumentByID(id uint64) (*doc.Document, error) {
	if id == 0 {
		d := s.ActiveDocument()
		if d == nil {
			return nil, ErrNoActiveDoc
		}
		return d, nil
	}
	for _, d := range s.Workspace().Documents() {
		if uint64(d.ID()) == id {
			return d, nil
		}
	}
	return nil, fmt.Errorf("app: no document with id %d", id)
}

// AddView adds a new view to a document (id 0 = active) and makes it active, returning its
// index. With copyActiveCamera the new view starts at the document's current active-view
// camera; otherwise it gets a default view that the UI Home-fits the first time it is
// shown. An empty name is auto-numbered ("View N").
func (s *Session) AddView(docID uint64, name string, copyActiveCamera bool) (int, error) {
	d, err := s.DocumentByID(docID)
	if err != nil {
		return 0, err
	}
	vs := d.Views()
	if name == "" {
		name = fmt.Sprintf("View %d", vs.Count()+1)
	}
	nv := doc.DefaultView(name)
	if copyActiveCamera {
		cur := vs.Active()
		nv.Eye, nv.Target, nv.Up, nv.FOV, nv.Framed = cur.Eye, cur.Target, cur.Up, cur.FOV, cur.Framed
	}
	return vs.Add(nv), nil
}

// ViewCamera returns a document's active-view camera (id 0 = active document), sized to
// the current viewport.
func (s *Session) ViewCamera(docID uint64) (scene.Camera, error) {
	d, err := s.DocumentByID(docID)
	if err != nil {
		return scene.Camera{}, err
	}
	v := d.Views().Active()
	c := s.camera // carry the transient viewport pixel size
	c.Eye, c.Target, c.Up, c.FOV = v.Eye, v.Target, v.Up, v.FOV
	return c, nil
}

// SetViewCamera applies c to a document's active view (id 0 = active document). When the
// addressed document is the active one this routes through SetCamera (picker + cache
// sync); otherwise it writes the off-screen document's view frame directly.
func (s *Session) SetViewCamera(docID uint64, c scene.Camera) error {
	d, err := s.DocumentByID(docID)
	if err != nil {
		return err
	}
	if d == s.ActiveDocument() {
		s.SetCamera(c)
		return nil
	}
	v := d.Views().Active()
	v.Eye, v.Target, v.Up, v.FOV, v.Framed = c.Eye, c.Target, c.Up, c.FOV, true
	return nil
}

// ViewCameraAt returns the active document's view i camera, sized to the current viewport.
// ok is false when there is no active document or i is out of range. Used by the tiled
// viewport to render each tile from its own view's camera.
func (s *Session) ViewCameraAt(i int) (scene.Camera, bool) {
	d := s.ActiveDocument()
	if d == nil {
		return scene.Camera{}, false
	}
	vs := d.Views()
	if i < 0 || i >= vs.Count() {
		return scene.Camera{}, false
	}
	v := vs.All()[i]
	c := s.camera // carry the transient viewport pixel size
	c.Eye, c.Target, c.Up, c.FOV = v.Eye, v.Target, v.Up, v.FOV
	return c, true
}

// SetViewCameraAt writes c to the active document's view i (marking it framed). When i is
// the active view it routes through SetCamera (picker + cache sync); otherwise it writes
// the view frame directly. Out-of-range or no-active-document is a no-op.
func (s *Session) SetViewCameraAt(i int, c scene.Camera) {
	d := s.ActiveDocument()
	if d == nil {
		return
	}
	vs := d.Views()
	if i < 0 || i >= vs.Count() {
		return
	}
	if i == vs.ActiveIndex() {
		s.SetCamera(c)
		return
	}
	v := vs.All()[i]
	v.Eye, v.Target, v.Up, v.FOV, v.Framed = c.Eye, c.Target, c.Up, c.FOV, true
}

// ActivateView makes view i of the active document the active view (so picking, sketch and
// commands target it). Used when the user interacts with a tile.
func (s *Session) ActivateView(i int) error {
	d := s.ActiveDocument()
	if d == nil {
		return ErrNoActiveDoc
	}
	return d.Views().Activate(i)
}
