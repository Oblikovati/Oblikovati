// SPDX-License-Identifier: GPL-2.0-only
use super::{NativeHeadWindow, SavedWindowView, saved_view::WireCameraStamp};
use oblikovati_api::wire::{CameraView, SetCameraArgs};

impl NativeHeadWindow {
    pub(super) fn current_window_view(&self) -> Result<SavedWindowView, String> {
        let camera = self.view.camera_state()?;
        let authored = self
            .wire_camera_stamp
            .as_ref()
            .filter(|stamp| stamp.matches(camera, self.view_projection))
            .map(|stamp| stamp.authored.clone());
        Ok(SavedWindowView {
            camera,
            projection: self.view_projection,
            authored,
            sketch: self.sketch_view,
        })
    }

    fn window_view(&self, slot: u64) -> Result<SavedWindowView, String> {
        if slot == self.view_slot {
            return self.current_window_view();
        }
        Ok(self
            .saved_views
            .get(&slot)
            .cloned()
            .unwrap_or_else(|| SavedWindowView::initial(slot)))
    }

    pub(super) fn camera_for_slot(&self, slot: u64) -> Result<CameraView, String> {
        Ok(self.window_view(slot)?.wire_frame())
    }

    pub(super) fn assign_camera(
        &mut self,
        slot: u64,
        args: &SetCameraArgs,
    ) -> Result<CameraView, String> {
        let before = self.window_view(slot)?;
        let (camera, projection, authored) =
            crate::camera_automation::prepare(before.camera, before.projection, args)?;
        if slot == self.view_slot {
            self.apply_wire_camera(before, camera)?;
            self.view_projection = projection;
            self.wire_camera_stamp = Some(WireCameraStamp {
                camera: self.view.camera_state()?,
                projection,
                authored: authored.clone(),
            });
        } else {
            self.saved_views.insert(
                slot,
                SavedWindowView {
                    camera,
                    projection,
                    authored: Some(authored.clone()),
                    ..before
                },
            );
        }
        Ok(authored)
    }

    fn apply_wire_camera(
        &mut self,
        before: SavedWindowView,
        camera: oblikovati_viewport::ViewportCamera,
    ) -> Result<(), String> {
        self.view.set_camera(camera)?;
        if let Err(cause) = self
            .view
            .navigate(oblikovati_viewport::ViewportNavigation::FitDepth)
        {
            self.view.set_camera(before.camera).map_err(|restore| {
                format!("camera depth refresh failed: {cause}; restoring camera failed: {restore}")
            })?;
            return Err(cause);
        }
        self.orientation_transition = None;
        Ok(())
    }
}
