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
