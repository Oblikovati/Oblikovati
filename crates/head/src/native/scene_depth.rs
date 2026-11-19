// SPDX-License-Identifier: GPL-2.0-only
//! Keep changed model geometry within the current camera depth range (#4044).
use oblikovati_viewport::{Viewport, ViewportCamera, ViewportNavigation, ViewportScene};

trait DepthTarget {
    fn depth_camera(&self) -> Result<ViewportCamera, String>;
    fn fit_depth(&mut self) -> Result<(), String>;
}
impl DepthTarget for Viewport {
    fn depth_camera(&self) -> Result<ViewportCamera, String> {
        self.camera_state()
    }
    fn fit_depth(&mut self) -> Result<(), String> {
        self.navigate(ViewportNavigation::FitDepth).map(|_| ())
    }
}

impl super::NativeHeadWindow {
    pub(super) fn submit_model_scene(
        &mut self,
        update: crate::HeadSceneUpdate,
    ) -> Result<(), String> {
        self.view.submit(&update.scene)?;
        self.refresh_presentation_bounds()?;
        self.scene_depth_bounds = SceneBounds::from_scene(&update.scene);
        self.current_scene = Some(update.scene);
        self.scene_status = Some(update.status);
        self.refresh_title()?;
        self.refresh_current_scene_depth()
    }

    /// Cache display-only bounds when scene geometry changes, not on each GUI frame.
    pub(super) fn refresh_presentation_bounds(&mut self) -> Result<(), String> {
        self.presentation_bounds = self.view.visible_presentation_bounds()?;
        #[cfg(test)]
        {
            self.presentation_bounds_refreshes += 1;
        }
        Ok(())
    }

    #[cfg(test)]
    pub(crate) fn presentation_bounds_refresh_count(&self) -> usize {
        self.presentation_bounds_refreshes
    }

    pub(super) fn restore_previous_scene_view(&mut self) -> Result<bool, String> {
        let changed = self.view.previous_view()?;
        if changed {
            self.refresh_current_scene_depth()?;
        }
        Ok(changed)
    }

    pub(super) fn refresh_current_scene_depth(&mut self) -> Result<(), String> {
        let (Some(scene), Some(bounds)) = (&self.current_scene, self.scene_depth_bounds) else {
            return Ok(());
        };
        refresh_with_bounds(&mut self.view, scene, bounds)
    }
}

#[derive(Clone, Copy)]
pub(super) struct SceneBounds {
    minimum: [f32; 3],
    maximum: [f32; 3],
}
impl SceneBounds {
    fn from_scene(scene: &ViewportScene) -> Option<Self> {
        let mut points = scene
            .triangles
            .iter()
            .flat_map(|triangle| triangle.positions);
        let first = points.next()?;
        let mut bounds = Self {
            minimum: first,
            maximum: first,
        };
        for point in points {
            for (axis, value) in point.into_iter().enumerate() {
                bounds.minimum[axis] = bounds.minimum[axis].min(value);
                bounds.maximum[axis] = bounds.maximum[axis].max(value);
            }
        }
        Some(bounds)
    }

    fn fits(self, camera: ViewportCamera) -> Result<bool, String> {
        let axis = view_axis(camera)?;
        validate_depth_range(camera)?;
        for mask in 0..8 {
            let point = std::array::from_fn(|i| {
                if mask & (1 << i) == 0 {
                    self.minimum[i]
                } else {
                    self.maximum[i]
                }
            });
            let depth = point_depth(point, camera.eye, axis);
            if !depth.is_finite() || depth <= camera.near || depth >= camera.far {
                return Ok(false);
            }
        }
        Ok(true)
    }
}

fn refresh_with_bounds(
    target: &mut impl DepthTarget,
    scene: &ViewportScene,
    bounds: SceneBounds,
) -> Result<(), String> {
    if bounds.fits(target.depth_camera()?)? {
        return Ok(());
    }
    refresh_if_clipped(target, scene)
}

fn refresh_if_clipped(target: &mut impl DepthTarget, scene: &ViewportScene) -> Result<(), String> {
    if scene.triangles.is_empty() || !scene_needs_depth_fit(scene, target.depth_camera()?)? {
        return Ok(());
    }
    target.fit_depth()?;
    let fitted = target.depth_camera()?;
    if scene_needs_depth_fit(scene, fitted)? {
        return Err(format!(
            "scene remains outside camera depth {:?}..{:?}: expected visible model geometry",
            fitted.near, fitted.far
        ));
    }
    Ok(())
}

fn scene_needs_depth_fit(scene: &ViewportScene, camera: ViewportCamera) -> Result<bool, String> {
    let axis = view_axis(camera)?;
    validate_depth_range(camera)?;
    let mut clipped = false;
    for point in scene
        .triangles
        .iter()
        .flat_map(|triangle| triangle.positions)
    {
        let depth = point_depth(point, camera.eye, axis);
        if !depth.is_finite() {
            return Err(format!(
                "scene point {point:?}: expected finite camera depth"
            ));
        }
        // A depth-only adjustment cannot place geometry behind the eye in view.
        if depth <= 0. {
            return Ok(false);
        }
        clipped |= depth <= camera.near || depth >= camera.far;
    }
    Ok(clipped)
}

fn validate_depth_range(camera: ViewportCamera) -> Result<(), String> {
    if camera.near.is_finite()
        && camera.far.is_finite()
        && camera.near > 0.
        && camera.far > camera.near
    {
        return Ok(());
    }
    Err(format!(
        "camera depth {:?}..{:?}: expected finite 0 < near < far",
        camera.near, camera.far
    ))
}

fn view_axis(camera: ViewportCamera) -> Result<[f64; 3], String> {
    let offset = std::array::from_fn::<_, 3, _>(|i| camera.target[i] - camera.eye[i]);
    let length = offset[0].hypot(offset[1]).hypot(offset[2]);
    if !length.is_finite() || length == 0. {
        return Err(format!(
            "camera eye {:?}, target {:?}: expected finite distinct points",
            camera.eye, camera.target
        ));
    }
    Ok(offset.map(|value| value / length))
}

fn point_depth(point: [f32; 3], eye: [f64; 3], axis: [f64; 3]) -> f64 {
    (0..3)
        .map(|i| (f64::from(point[i]) - eye[i]) * axis[i])
        .sum()
}

#[cfg(test)]
#[path = "../../tests/native_scene/depth_unit.rs"]
mod tests;
