// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::ViewportNavigation;
impl RenderView {
    pub(crate) fn camera_state(&self) -> ViewportCamera {
        let camera = self.view.camera();
        let eye = camera.eye();
        let target = camera.center();
        let up = camera.up();
        ViewportCamera {
            eye: [eye.x(), eye.y(), eye.z()],
            target: [target.x(), target.y(), target.z()],
            up: [up.x(), up.y(), up.z()],
            scale: camera.scale(),
            near: camera.z_near(),
            far: camera.z_far(),
            perspective: !camera.is_orthographic(),
        }
    }
    pub(crate) fn navigate(
        &self,
        command: ViewportNavigation,
        framing: Option<crate::fit_region::FitFraming>,
    ) -> Result<(), String> {
        let previous = self.view.camera();
        let candidate = Handle::new(previous.copy_members());
        self.view
            .set_camera(&candidate)
            .map_err(|cause| cause.to_string())?;
        if matches!(
            command,
            ViewportNavigation::Fit { .. } | ViewportNavigation::Home { .. }
        ) {
            candidate.set_tile(&Default::default());
        }
        let outcome = self
            .apply_navigation(command)
            .and_then(|()| self.apply_fit_region(framing))
            .and_then(|()| self.camera_state().validate());
        if let Err(cause) = outcome {
            self.view.set_camera(&previous).map_err(|rollback| {
                format!("navigation {command:?}: {cause}; camera restore: {rollback}")
            })?;
            return Err(format!(
                "navigation {command:?}: expected a valid camera: {cause}"
            ));
        }
        Ok(())
    }
    /// The same displayed bounds Fit uses, with the transient preview excluded.
    pub(crate) fn fit_visible_bounds(&self) -> Result<[[f64; 3]; 2], String> {
        self.visible_presentation_bounds()?
            .ok_or_else(|| "display fit bounds: expected nonempty finite bounds".into())
    }
    /// Presentation-only bounds; an empty scene has no bounds.
    pub(crate) fn visible_presentation_bounds(&self) -> Result<Option<[[f64; 3]; 2]>, String> {
        if self.structures.is_empty() && self.sketches.is_empty() {
            return Ok(None);
        }
        let (count, values) = self
            .preview
            .without_preview(|| self.view.min_max_world())
            .map_err(|cause| {
                format!("display fit bounds: expected visible scene bounds: {cause}")
            })?;
        if count == 0 {
            return Ok(None);
        }
        if values.iter().any(|value| !value.is_finite()) {
            return Err(format!(
                "display fit bounds {values:?}, structures {count}: expected finite bounds"
            ));
        }
        Ok(Some([
            [values[0], values[1], values[2]],
            [values[3], values[4], values[5]],
        ]))
    }
    fn apply_fit_region(
        &self,
        framing: Option<crate::fit_region::FitFraming>,
    ) -> Result<(), String> {
        let Some(framing) = framing else {
            return Ok(());
        };
        if self.structures.is_empty() && self.sketches.is_empty() {
            return Ok(());
        }
        let camera = self.view.camera();
        camera.set_scale(camera.scale() * framing.scale);
        let [width, height] = framing.extent.map(|side| side as i32);
        let [x, y] = framing.offset;
        camera.set_tile(&martensite_tkservice::graphic3d::Graphic3dCameraTile {
            total_size: martensite_tkernel::ncollection::Vec2::new(width, height),
            tile_size: martensite_tkernel::ncollection::Vec2::new(width, height),
            offset: martensite_tkernel::ncollection::Vec2::new(-x, y),
            is_top_down: false,
        });
        self.view.z_fit_all(1.).map_err(|cause| cause.to_string())
    }
    fn apply_navigation(&self, command: ViewportNavigation) -> Result<(), String> {
        let result = match command {
            ViewportNavigation::Fit { margin } => self.fit_displayed(margin),
            ViewportNavigation::Home { margin } => self.home_displayed(margin),
            ViewportNavigation::FitDepth => self.fit_displayed_depth(),
            ViewportNavigation::Pan([x, y]) => self.view.pan(x, -y, 1., true),
            ViewportNavigation::Orbit { yaw, pitch } => self.orbit_screen_axes(yaw, pitch),
            ViewportNavigation::Zoom { factor, anchor } => self.zoom_anchor(factor, anchor),
        };
        result.map_err(|cause| cause.to_string())
    }
    fn fit_displayed_depth(&self) -> Result<(), martensite_tkernel::standard::Failure> {
        if self.structures.is_empty() && self.sketches.is_empty() {
            return Ok(());
        }
        self.view.z_fit_all(1.)
    }
    fn fit_displayed(&self, margin: f64) -> Result<(), martensite_tkernel::standard::Failure> {
        self.preview.without_preview(|| self.fit_scene(margin))
    }
    fn fit_scene(&self, margin: f64) -> Result<(), martensite_tkernel::standard::Failure> {
        if self.structures.is_empty() && self.sketches.is_empty() {
            return Ok(());
        }
        let camera = self.view.camera();
        let distance = camera.distance();
        self.view.fit_all(margin, false)?;
        // Orthographic framing changes scale, not standoff. Preserve a valid pivot
        // distance for every input; kernel 193572dc FitMinMax sets it to box depth.
        camera.set_distance(distance);
        self.view.z_fit_all(1.)
    }
    fn home_displayed(&self, margin: f64) -> Result<(), martensite_tkernel::standard::Failure> {
        let camera = self.view.camera();
        // Preserve Go scene/camera.go Home: eye offset (+X,+Y,+Z), Y-up.
        camera.set_direction(&Direction3::new(-1., -1., -1.)?);
        camera.set_up(&Direction3::new(0., 1., 0.)?);
        self.fit_displayed(margin)
    }
    fn orbit_screen_axes(
        &self,
        yaw: f64,
        pitch: f64,
    ) -> Result<(), martensite_tkernel::standard::Failure> {
        // Preserve the Go yaw-then-pitch order and the current target as the orbit pivot.
        // Kernel 193572dc Rotate uses -Ax about up and Ay about right.
        self.view.rotate(-yaw, 0., 0., true)?;
        self.view.rotate(0., pitch, 0., true)
    }
    fn zoom_anchor(
        &self,
        factor: f64,
        anchor: Option<[i32; 2]>,
    ) -> Result<(), martensite_tkernel::standard::Failure> {
        let (x, y) = match anchor {
            Some([x, y]) => self.view.convert_to_view(x, y)?,
            None => (0., 0.),
        };
        self.view
            .panning(-x * (1. - factor), -y * (1. - factor), factor.recip(), true)
    }
}
