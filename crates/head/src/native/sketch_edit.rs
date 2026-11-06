// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl NativeHeadWindow {
    pub(super) fn synchronize_sketch_camera(
        &mut self,
        edit: Option<crate::HeadSketchEdit>,
    ) -> Result<bool, String> {
        let current = self
            .sketch_view
            .map(|(current, _)| (current.document, current.sketch));
        let requested = edit.map(|edit| (edit.document, edit.sketch));
        // Go reattaches the same sketch after restore without re-entering its camera transition.
        if current == requested {
            return Ok(false);
        }
        let Some(edit) = edit else {
            return self.restore_sketch_camera();
        };
        self.face_sketch_camera(edit)
    }
    /// Keep this window's view and restore the next one's; a new window starts at the default.
    pub(super) fn switch_view_slot(&mut self, slot: u64) -> Result<bool, String> {
        if slot == self.view_slot {
            return Ok(false);
        }
        let camera = self.view.camera_state()?;
        self.saved_views
            .insert(self.view_slot, (camera, self.sketch_view.take()));
        let (camera, sketch_view) = self.saved_views.remove(&slot).unwrap_or_else(|| {
            let first_view = oblikovati_viewport::ViewportCamera::default();
            let camera = if slot >= crate::FIRST_EXTRA_WINDOW {
                first_view.isometric()
            } else {
                first_view
            };
            (camera, None)
        });
        self.view.set_camera(camera)?;
        self.sketch_view = sketch_view;
        self.view_slot = slot;
        self.navigation = Default::default();
        Ok(true)
    }
    fn face_sketch_camera(&mut self, edit: crate::HeadSketchEdit) -> Result<bool, String> {
        let current = self.view.camera_state()?;
        let returning = self.sketch_view.map_or(current, |(_, returning)| returning);
        let basis = self
            .sketch_view
            .filter(|(old, _)| (old.document, old.sketch) != (edit.document, edit.sketch))
            .map_or(current, |(_, returning)| returning);
        self.view.set_camera(basis.facing_sketch(edit.plane)?)?;
        self.sketch_view = Some((edit, returning));
        self.navigation = Default::default();
        Ok(true)
    }
    fn restore_sketch_camera(&mut self) -> Result<bool, String> {
        let Some((_, returning)) = self.sketch_view else {
            return Ok(false);
        };
        self.view.set_camera(returning)?;
        self.sketch_view = None;
        self.navigation = Default::default();
        Ok(true)
    }
}
