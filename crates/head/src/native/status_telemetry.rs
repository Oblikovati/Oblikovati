// SPDX-License-Identifier: GPL-2.0-only
use super::NativeHeadWindow;
use crate::HeadWindow;
use oblikovati_gui::GuiCommandState;
use oblikovati_model::{sketch::SketchPlane, units::convert_value};
use oblikovati_viewport::{ViewportCamera, ViewportEvent, ViewportInput};

impl NativeHeadWindow {
    pub(super) fn publish_status_telemetry(
        &mut self,
        event: Option<&ViewportEvent>,
        commands: &mut GuiCommandState,
    ) {
        match event {
            Some(ViewportEvent::Input(
                ViewportInput::Motion { position, .. }
                | ViewportInput::Button { position, .. }
                | ViewportInput::Wheel { position, .. },
            )) => self.status_pointer = Some(*position),
            Some(ViewportEvent::Focus(false) | ViewportEvent::Mapped(false)) => {
                self.status_pointer = None;
            }
            _ => {}
        }
        use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART};
        if !commands
            .document_type
            .as_ref()
            .is_some_and(|kind| *kind == DOCUMENT_PART || *kind == DOCUMENT_ASSEMBLY)
        {
            return;
        }
        if let Ok(camera) = self.view_camera() {
            let percent = 100. * ViewportCamera::default().scale / camera.scale;
            if percent.is_finite() && percent > 0. {
                commands.viewport_zoom_percent = Some(percent.round().max(1.) as u64);
            }
        }
        commands.viewport_coordinates = self.status_coordinates(commands);
    }

    fn status_coordinates(&mut self, commands: &GuiCommandState) -> Option<[f64; 2]> {
        let [x, y] = self.status_pointer?;
        let [left, top, right, bottom] = self
            .gui
            .as_ref()?
            .document_area(self.view.drawable_extent().ok()?)
            .ok()?;
        if !(left..right).contains(&u32::try_from(x).ok()?)
            || !(top..bottom).contains(&u32::try_from(y).ok()?)
        {
            return None;
        }
        let units = commands.document_units.as_ref()?;
        let plane = commands.active_sketch_plane.unwrap_or_else(SketchPlane::xy);
        let [u, v] = self.sketch_point([x, y], plane).ok()??;
        let one_cm = convert_value(1., "cm", &units.length_unit).ok()?;
        // The projected plane is in the viewport's physical centimetres.
        Some([u * one_cm, v * one_cm])
    }
}
