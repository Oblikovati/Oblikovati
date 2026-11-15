# Origin-plane identity evidence

Issue: #3928. This is conditional evidence, not feature qualification.

- The one-size projected rectangle on an explicitly identified XY plane has
  zero origin-projection bounds and passes the conditional root check.
- Fixed XY lookalikes remain unqualified. Recipe version 13 retains origin-plane
  identity; older versions reject that representation. Save/reopen and
  history checks preserve the identity. Editing to a fixed frame removes it;
  undo restores it and redo restores the fixed frame.
- Typed sketch creation checks XY, XZ, and YZ. A native GUI test selected
  **Start 2D Sketch** and **XY Plane** in a real window under Xvfb with Mesa
  lavapipe. The test injected pointer events into the frame loop; it did not
  use OS-level mouse or keyboard input. The test closed its window.
- A Go `persistence.OpenPackage` probe rejected a schema 3 document as newer
  than its schema 2 support before model restoration. Rust persistence tests
  confirm origin-plane sketches use document schema 3.

The tested native GUI command was:

```sh
VK_ICD_FILENAMES=/usr/share/vulkan/icd.d/lvp_icd.json xvfb-run -a \
  cargo test --locked -p oblikovati-head --lib \
  sketch_plane_label_remains_clickable_at_workspace_edge
```

The hardware ICD did not present to Xvfb. No full local gate, cross-platform
interval-arithmetic qualification, or session identity proof was completed.
