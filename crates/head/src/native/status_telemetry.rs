// SPDX-License-Identifier: GPL-2.0-only
use super::NativeHeadWindow;
use crate::HeadWindow;
use oblikovati_gui::GuiCommandState;
use oblikovati_model::{sketch::SketchPlane, units::convert_value};
use oblikovati_viewport::{ViewportCamera, ViewportEvent, ViewportInput};

#[derive(Clone, Copy, PartialEq)]
struct StatusProjectionKey {
    pixel: [i32; 2],
    extent: [u32; 2],
    generations: [u64; 2],
    plane: SketchPlane,
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn status_projection_reuses_only_the_same_pointer_camera_surface_and_plane() {
        let mut projection = StatusProjection::default();
        let key = StatusProjectionKey {
            pixel: [400, 300],
            extent: [800, 600],
            generations: [1, 2],
            plane: SketchPlane::xy(),
        };
        assert_eq!(projection.coordinates(key, || None), None);
        assert_eq!(
            projection.coordinates(key, || Some([3., 4.])),
            Some([3., 4.])
        );
        assert_eq!(
            projection.coordinates(key, || panic!("repeated status read")),
            Some([3., 4.])
        );
        for changed in [
            StatusProjectionKey {
                pixel: [401, 300],
                ..key
            },
            StatusProjectionKey {
                generations: [2, 2],
                ..key
            },
            StatusProjectionKey {
                generations: [1, 3],
                ..key
            },
            StatusProjectionKey {
                extent: [800, 601],
                ..key
            },
            StatusProjectionKey {
                plane: SketchPlane::xz(),
                ..key
            },
        ] {
            assert_eq!(
                projection.coordinates(changed, || Some([5., 6.])),
                Some([5., 6.])
            );
        }
    }
}

#[derive(Default)]
pub(super) struct StatusProjection {
    previous: Option<(StatusProjectionKey, [f64; 2])>,
}

impl StatusProjection {
    fn coordinates(
        &mut self,
        key: StatusProjectionKey,
        project: impl FnOnce() -> Option<[f64; 2]>,
    ) -> Option<[f64; 2]> {
        if let Some((previous, coordinates)) = self.previous
            && previous == key
        {
            return Some(coordinates);
        }
        let coordinates = project()?;
        self.previous = Some((key, coordinates));
        Some(coordinates)
    }
}

impl NativeHeadWindow {
    pub(super) fn publish_status_telemetry(
        &mut self,
        event: Option<&ViewportEvent>,
        extent: [u32; 2],
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
        commands.viewport_coordinates = self.status_coordinates(extent, commands);
    }

    fn status_coordinates(
        &mut self,
        extent: [u32; 2],
        commands: &GuiCommandState,
    ) -> Option<[f64; 2]> {
        let [x, y] = self.status_pointer?;
        let [left, top, right, bottom] = self.gui.as_ref()?.document_area(extent).ok()?;
        if !(left..right).contains(&u32::try_from(x).ok()?)
            || !(top..bottom).contains(&u32::try_from(y).ok()?)
        {
            return None;
        }
        let units = commands.document_units.as_ref()?;
        let plane = commands.active_sketch_plane.unwrap_or_else(SketchPlane::xy);
        let key = StatusProjectionKey {
            pixel: [x, y],
            extent,
            generations: self.view.projection_generations(),
            plane,
        };
        let [u, v] = self
            .status_projection
            .coordinates(key, || self.view.sketch_point([x, y], plane).ok().flatten())?;
        let one_cm = convert_value(1., "cm", &units.length_unit).ok()?;
        // The projected plane is in the viewport's physical centimetres.
        Some([u * one_cm, v * one_cm])
    }
}
