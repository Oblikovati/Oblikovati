// SPDX-License-Identifier: GPL-2.0-only
//! Validate restored part-view recipes before they are adopted (#4118).
use super::PartDesignViews;
use oblikovati_api::types::{
    PART_ANNOTATION_SCALE_AUTO, PART_ANNOTATION_SCALE_CUSTOM, PART_ANNOTATION_SCALE_FIXED,
    PartAnnotationScale, PartDesignViewCamera,
};
use std::collections::BTreeSet;

pub(super) fn validate_scale(scale: &PartAnnotationScale) -> Result<(), String> {
    let valid = (scale.mode == PART_ANNOTATION_SCALE_AUTO
        && scale.numerator == 0
        && scale.denominator == 0)
        || ((scale.mode == PART_ANNOTATION_SCALE_FIXED
            || scale.mode == PART_ANNOTATION_SCALE_CUSTOM)
            && scale.numerator > 0
            && scale.denominator > 0);
    if !valid {
        return Err(format!(
            "annotation scale {scale:?}: expected Auto or a positive ratio"
        ));
    }
    Ok(())
}

/// Check that a part camera can be displayed; example: `validate_camera(&camera)?`.
pub fn validate_camera(camera: &PartDesignViewCamera) -> Result<(), String> {
    if !valid_camera_scalars(camera) || !valid_camera_vectors(camera) {
        return Err(format!(
            "part view camera {camera:?}: expected a finite usable viewport frame and supported projection"
        ));
    }
    Ok(())
}

fn valid_camera_scalars(camera: &PartDesignViewCamera) -> bool {
    use oblikovati_api::types::{ORTHOGRAPHIC_PROJECTION, PERSPECTIVE_PROJECTION};
    [camera.scale, camera.near, camera.far]
        .iter()
        .all(|value| value.is_finite())
        && camera.scale > 0.
        && camera.near > 0.
        && camera.far > camera.near
        && (camera.projection == ORTHOGRAPHIC_PROJECTION
            || camera.projection == PERSPECTIVE_PROJECTION)
}

fn valid_camera_vectors(camera: &PartDesignViewCamera) -> bool {
    let eye = [camera.eye.x, camera.eye.y, camera.eye.z];
    let target = [camera.target.x, camera.target.y, camera.target.z];
    let up = [camera.up.x, camera.up.y, camera.up.z];
    if eye
        .iter()
        .chain(&target)
        .chain(&up)
        .any(|value| !value.is_finite())
    {
        return false;
    }
    let direction = [
        camera.target.x - camera.eye.x,
        camera.target.y - camera.eye.y,
        camera.target.z - camera.eye.z,
    ];
    [direction, up].iter().all(|vector| {
        let magnitude: f64 = vector.iter().map(|value| value * value).sum();
        magnitude.is_finite() && magnitude != 0.
    }) && camera_directions_independent(direction, up)
}

fn camera_directions_independent(direction: [f64; 3], up: [f64; 3]) -> bool {
    let direction_length = direction
        .iter()
        .map(|value| value * value)
        .sum::<f64>()
        .sqrt();
    let up_length = up.iter().map(|value| value * value).sum::<f64>().sqrt();
    let direction = direction.map(|value| value / direction_length);
    let up = up.map(|value| value / up_length);
    let cross = [
        direction[1] * up[2] - direction[2] * up[1],
        direction[2] * up[0] - direction[0] * up[2],
        direction[0] * up[1] - direction[1] * up[0],
    ];
    let magnitude: f64 = cross.iter().map(|value| value * value).sum();
    magnitude.is_finite() && magnitude != 0.
}

impl PartDesignViews {
    /// Reject corrupt recipes; example: `views.validate()?` before document adoption.
    pub fn validate(&self) -> Result<(), String> {
        let primary = self
            .views
            .first()
            .ok_or("part views []: expected Primary first")?;
        if primary.id != 1 || primary.name != "Primary" || !primary.built_in {
            return Err(format!(
                "primary view {primary:?}: expected built-in Primary ID 1"
            ));
        }
        if self.views.iter().filter(|view| view.active).count() != 1 {
            return Err(format!(
                "part views {:?}: expected exactly one active view",
                self.views
            ));
        }
        for id in 1..=5 {
            if !self.views.iter().any(|view| view.id == id && view.built_in) {
                return Err(format!(
                    "part views {:?}: expected built-in view ID {id}",
                    self.views
                ));
            }
        }
        self.validate_entries()
    }
    fn validate_entries(&self) -> Result<(), String> {
        let mut ids = BTreeSet::new();
        let mut names = BTreeSet::new();
        for view in &self.views {
            if view.id == 0
                || !ids.insert(view.id)
                || view.name.trim().is_empty()
                || view.name.contains('\0')
                || !names.insert(&view.name)
                || (view.id <= 5) != view.built_in
            {
                return Err(format!(
                    "part view {view:?}: expected unique identity and name with consistent built-in status"
                ));
            }
            if view.built_in {
                let expected = ["Primary", "Isometric", "Front", "Top", "Right"];
                let index = usize::try_from(view.id - 1).map_err(|cause| cause.to_string())?;
                if expected.get(index) != Some(&view.name.as_str())
                    || view.standard_orientation != super::standard_orientation(view.id)
                {
                    return Err(format!(
                        "built-in part view {view:?}: expected its standard name and orientation"
                    ));
                }
            }
            validate_scale(&view.annotation_scale)?;
            let standard = view.standard_orientation;
            if standard.0 != 0 && !(2..=5).any(|id| super::standard_orientation(id) == standard) {
                return Err(format!(
                    "part view orientation {standard:?}: expected a supported standard orientation or none"
                ));
            }
            if let Some(camera) = &view.camera {
                validate_camera(camera)?;
            }
        }
        Ok(())
    }
}
