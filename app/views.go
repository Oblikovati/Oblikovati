// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"fmt"
	stdmath "math"

	"oblikovati/api/types"
	"oblikovati/math"
	"oblikovati/model/doc"
	"oblikovati/persistence/viewstate"
	"oblikovati/scene"
)

// faceAlignTol is the minimum |cosine| between the view direction and a principal axis for
// a view to count as a "face" view (a ViewCube face snap) under ProjPerspectiveOrthoFaces.
const faceAlignTol = 0.999

// orthoForView resolves whether a view renders orthographically: always for
// ProjOrthographic, never for ProjPerspective, and for ProjPerspectiveOrthoFaces only when
// the view direction is aligned to a principal axis (a straight-on face view).
func orthoForView(v *doc.View) bool {
	switch v.Projection {
	case doc.ProjOrthographic:
		return true
	case doc.ProjPerspectiveOrthoFaces:
		return faceAligned(v.Eye, v.Target)
	default:
		return false
	}
}

// faceAligned reports whether the eye→target direction is (near) parallel to a principal
// axis (±X/±Y/±Z) — i.e. a straight-on face view.
func faceAligned(eye, target math.Point3) bool {
	d := eye.VectorTo(target)
	l := d.Length()
	if l < 1e-9 {
		return false
	}
	d = d.Scale(1 / l)
	m := stdmath.Abs(d.X)
	if a := stdmath.Abs(d.Y); a > m {
		m = a
	}
	if a := stdmath.Abs(d.Z); a > m {
		m = a
	}
	return m >= faceAlignTol
}

// ActiveViewProjection returns the active view's projection mode (perspective by default).
func (s *Session) ActiveViewProjection() doc.ProjectionMode {
	if v := s.ActiveView(); v != nil {
		return v.Projection
	}
	return doc.ProjPerspective
}

// SetActiveViewProjection sets the active view's projection mode (the ViewCube projection
// menu). An invalid mode is ignored.
func (s *Session) SetActiveViewProjection(m doc.ProjectionMode) {
	if v := s.ActiveView(); v != nil && m.IsValid() {
		v.Projection = m
	}
}

// SetViewStateStore installs the per-user store that persists each document's view/camera
// configuration outside the .obk (so a camera move never dirties the document). The head
// and CLI inject a file-backed store; without one, view state is purely in-session.
func (s *Session) SetViewStateStore(store viewstate.Store) { s.viewState = store }

// saveViewState writes a document's current view configuration to the per-user store,
// keyed by its file path. Called when the document is saved. A no-op without a store or a
// path (an unsaved document has no key yet).
func (s *Session) saveViewState(d *doc.Document) {
	if s.viewState == nil || d == nil || d.FullFileName() == "" {
		return
	}
	vs := d.Views()
	st := viewstate.ViewState{Active: vs.ActiveIndex(), Layout: int32(vs.Layout())}
	st.SplitX, st.SplitY = vs.Split()
	for _, v := range vs.All() {
		st.Views = append(st.Views, viewstate.ViewFrame{
			Name:       v.Name,
			Eye:        [3]float64{v.Eye.X, v.Eye.Y, v.Eye.Z},
			Target:     [3]float64{v.Target.X, v.Target.Y, v.Target.Z},
			Up:         [3]float64{v.Up.X, v.Up.Y, v.Up.Z},
			FOV:        v.FOV,
			Projection: int(v.Projection),
		})
	}
	_ = s.viewState.Save(d.FullFileName(), st) // best-effort; a settings-write failure must not block saving the model
}

// loadViewState restores a document's view configuration from the per-user store after it
// is opened. A no-op without a store or a stored entry (the document keeps its default view).
func (s *Session) loadViewState(d *doc.Document) {
	if s.viewState == nil || d == nil {
		return
	}
	st, ok, err := s.viewState.Load(d.FullFileName())
	if err != nil || !ok || len(st.Views) == 0 {
		return
	}
	views := make([]*doc.View, len(st.Views))
	for i, f := range st.Views {
		views[i] = &doc.View{
			Name:       f.Name,
			Eye:        math.P3(f.Eye[0], f.Eye[1], f.Eye[2]),
			Target:     math.P3(f.Target[0], f.Target[1], f.Target[2]),
			Up:         math.V3(f.Up[0], f.Up[1], f.Up[2]),
			FOV:        f.FOV,
			Framed:     true,
			Projection: doc.ProjectionMode(f.Projection),
		}
	}
	d.RestoreViews(views, st.Active, types.ViewLayout(st.Layout))
	d.Views().SetSplit(st.SplitX, st.SplitY)
}

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
	c.Orthographic = orthoForView(v)
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

// ShowViewCube reports whether the navigation cube is shown in viewports (default true).
func (s *Session) ShowViewCube() bool { return !s.viewCubeHidden }

// SetShowViewCube shows or hides the navigation cube (View-tab toggle).
func (s *Session) SetShowViewCube(show bool) { s.viewCubeHidden = !show }

// ActivateView makes view i of the active document the active view (so picking, sketch and
// commands target it). Used when the user interacts with a tile.
func (s *Session) ActivateView(i int) error {
	d := s.ActiveDocument()
	if d == nil {
		return ErrNoActiveDoc
	}
	return d.Views().Activate(i)
}
