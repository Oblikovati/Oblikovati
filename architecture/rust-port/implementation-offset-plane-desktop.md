# Desktop offset work plane

The 3D Model ribbon opens an Offset Plane editor for the active part. A user
can select an origin plane, an existing work plane, or a planar solid face.
The owner validates the expression in the document's length units before OK
is enabled. Zero and invalid expressions do not create a feature; Cancel,
Escape, and the close button discard the draft.

Acceptance calls the existing typed `work_planes.create` operation. It stores
the reference and offset expression in the part history, so normal undo, redo,
and save/reopen apply. A selected face is resolved to a current semantic key;
an ambiguous work-plane name is refused. A work plane can host a new sketch.

The pinned `solid-projection-fixture-02` scenario supplies a rectangular source
and selects XZ with a 50 mm offset. The recording did not capture the source
rectangle dimensions; the test chooses a 10 mm square. The native scenario
checks the evaluated work-plane origin and starts a sketch on it. The Go
reference is `app/offset_plane_tool.go` and `head/ui/offset_plane_dialog.go`.
This covers offset creation, not the full family of work-plane constructors or
general work-plane viewport rendering. See #3977.
