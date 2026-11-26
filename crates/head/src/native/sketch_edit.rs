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
        self.orientation_transition = None;
        let Some(edit) = edit else {
            return self.restore_sketch_camera();
        };
        self.enter_sketch_camera(edit)
    }
    /// Name the active document in the title bar; only a change touches the window.
    pub(super) fn note_document_title(
        &mut self,
        commands: &oblikovati_gui::GuiCommandState,
    ) -> Result<(), String> {
        let title = oblikovati_gui::active_window_name(commands);
        if title == self.document_title {
            return Ok(());
        }
        self.document_title = title;
        self.refresh_title()
    }
    /// Keep this window's view and restore the next one's; a new window starts at the default.
    pub(super) fn switch_view_slot(&mut self, slot: u64) -> Result<bool, String> {
        if slot == self.view_slot {
            return Ok(false);
        }
        let current = self.current_window_view()?;
        let next = self
            .saved_views
            .get(&slot)
            .cloned()
            .unwrap_or_else(|| super::SavedWindowView::initial(slot));
        self.view.set_camera(next.camera)?;
        self.orientation_transition = None;
        self.saved_views.insert(self.view_slot, current);
        self.saved_views.remove(&slot);
        self.view_projection = next.projection;
        self.sketch_view = next.sketch;
        self.wire_camera_stamp = next
            .authored
            .map(|authored| super::saved_view::WireCameraStamp {
                camera: self.view.camera_state().unwrap_or(next.camera),
                projection: next.projection,
                authored,
            });
        self.view_slot = slot;
        self.navigation = Default::default();
        Ok(true)
    }
    fn enter_sketch_camera(&mut self, edit: crate::HeadSketchEdit) -> Result<bool, String> {
        let current = self.view.camera_state()?;
        let returning = self.sketch_view.and_then(|(_, returning)| returning);
        let basis = returning.unwrap_or(current);
        if !edit.look_at_plane {
            self.sketch_view = Some((edit, None));
            return Ok(false);
        }
        self.view.set_camera(basis.facing_sketch(edit.plane)?)?;
        self.sketch_view = Some((edit, Some(basis)));
        self.navigation = Default::default();
        Ok(true)
    }
    fn restore_sketch_camera(&mut self) -> Result<bool, String> {
        let Some((_, returning)) = self.sketch_view else {
            return Ok(false);
        };
        if let Some(returning) = returning {
            self.view.set_camera(returning)?;
        }
        self.sketch_view = None;
        if returning.is_none() {
            return Ok(false);
        }
        self.navigation = Default::default();
        Ok(true)
    }
}
