// SPDX-License-Identifier: GPL-2.0-only
use crate::ViewportCamera;
use oblikovati_model::sketch::SketchPlane;
impl ViewportCamera {
    /// The same target and scale from the corner of the model axes.
    /// Example: `ViewportCamera::default().isometric()` for a newly opened view.
    pub fn isometric(self) -> Self {
        let delta: [f64; 3] = std::array::from_fn(|axis| self.eye[axis] - self.target[axis]);
        let distance = delta.iter().map(|value| value * value).sum::<f64>().sqrt();
        let along = distance / 3f64.sqrt();
        Self {
            eye: self.target.map(|value| value + along),
            up: [0., 1., 0.],
            ..self
        }
    }
    /// Face a sketch from the viewer's current side, preserving scale and standoff.
    /// Example: `view.set_camera(view.camera_state()?.facing_sketch(plane)?)?` on edit entry.
    pub fn facing_sketch(self, plane: SketchPlane) -> Result<Self, String> {
        let frame = plane.recipe();
        let zero_negative = dominant_normal_component(plane.normal()) < 0.;
        facing_plane(
            self,
            plane,
            normalized_up(sketch_up(frame.x_axis, frame.y_axis)),
            zero_negative,
        )
    }
    /// Face a selected planar reference with its Y axis up; example: `camera.facing_planar_reference(plane)?`.
    pub fn facing_planar_reference(self, plane: SketchPlane) -> Result<Self, String> {
        facing_plane(self, plane, plane.recipe().y_axis, false)
    }
}

fn facing_plane(
    camera: ViewportCamera,
    plane: SketchPlane,
    up: [f64; 3],
    zero_negative: bool,
) -> Result<ViewportCamera, String> {
    camera.validate()?;
    let origin = plane.recipe().origin;
    let normal = plane.normal();
    let standoff = plane_standoff(camera, origin, normal, zero_negative)?;
    let eye = std::array::from_fn(|axis| origin[axis] + normal[axis] * standoff);
    let facing = ViewportCamera {
        eye,
        target: origin,
        up,
        ..camera
    };
    facing.validate()?;
    Ok(facing)
}

/// Model Y stays up when the plane holds it, so a sketch on YZ reads as the Right view.
fn sketch_up(x_axis: [f64; 3], y_axis: [f64; 3]) -> [f64; 3] {
    const MODEL_Y: [f64; 3] = [0., 1., 0.];
    if x_axis == MODEL_Y { MODEL_Y } else { y_axis }
}

fn plane_standoff(
    camera: ViewportCamera,
    origin: [f64; 3],
    normal: [f64; 3],
    zero_negative: bool,
) -> Result<f64, String> {
    let delta: [f64; 3] = std::array::from_fn(|axis| camera.eye[axis] - camera.target[axis]);
    let distance = delta.iter().map(|value| value * value).sum::<f64>().sqrt();
    // Go scene.Camera.Facing at aa8297d1 uses a unit standoff below minDistance.
    let distance = if distance < 1e-3 { 1. } else { distance };
    let side: f64 = (0..3)
        .map(|axis| (camera.eye[axis] - origin[axis]) * normal[axis])
        .sum();
    if !side.is_finite() {
        return Err(format!(
            "sketch camera side {side}: expected a finite view-plane projection"
        ));
    }
    let negative = side < 0. || (side == 0. && zero_negative);
    Ok(if negative { -distance } else { distance })
}
// When the eye lies in the plane, choose the positive model-axis side for a stable view.
fn dominant_normal_component(normal: [f64; 3]) -> f64 {
    normal.into_iter().fold(0_f64, |largest, value| {
        if value.abs() > largest.abs() {
            value
        } else {
            largest
        }
    })
}
fn normalized_up(axis: [f64; 3]) -> [f64; 3] {
    let magnitude = axis.iter().map(|value| value * value).sum::<f64>().sqrt();
    axis.map(|value| value / magnitude)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn planar_reference_uses_its_y_axis_and_preserves_the_viewer_side() {
        let plane =
            SketchPlane::new([2., 3., 4.], [0., 1., 0.], [0., 0., 1.], Default::default()).unwrap();
        let camera = ViewportCamera {
            eye: [-8., 3., 4.],
            target: [2., 3., 4.],
            ..Default::default()
        };
        let result = camera.facing_planar_reference(plane).unwrap();
        assert_eq!(result.eye, [-8., 3., 4.]);
        assert_eq!(result.target, [2., 3., 4.]);
        assert_eq!(result.up, [0., 0., 1.]);
    }
    #[test]
    fn a_perspective_view_shows_the_same_height_at_the_target() {
        let camera = ViewportCamera::default();
        let half_angle = (camera.perspective_field_of_view() / 2.).to_radians();
        let shown = 2. * 20. * half_angle.tan();
        assert!((shown - camera.scale).abs() < 1e-9, "height {shown}");
    }
    #[test]
    fn facing_preserves_viewer_side_distance_scale_and_projection() {
        let plane =
            SketchPlane::new([2., 3., 4.], [1., 0., 0.], [0., 0., 1.], Default::default()).unwrap();
        let front = ViewportCamera {
            eye: [2., -7., 4.],
            target: [2., 3., 4.],
            ..Default::default()
        };
        let back = ViewportCamera {
            eye: [2., 13., 4.],
            ..front
        };
        for camera in [front, back] {
            let facing = camera.facing_sketch(plane).unwrap();
            assert_eq!(facing.eye, camera.eye);
            assert_eq!(facing.target, [2., 3., 4.]);
            assert_eq!(facing.up, [0., 0., 1.]);
            assert_eq!(
                (facing.scale, facing.near, facing.far),
                (camera.scale, camera.near, camera.far)
            );
        }
    }
    #[test]
    fn model_y_stays_up_when_the_sketch_plane_holds_it() {
        let right = ViewportCamera::default()
            .facing_sketch(SketchPlane::yz())
            .unwrap();
        assert_eq!(right.up, [0., 1., 0.]);
        let top = ViewportCamera::default()
            .facing_sketch(SketchPlane::xz())
            .unwrap();
        assert_eq!(top.up, [0., 0., 1.]);
        assert_eq!(top.eye, [0., 20., 0.]);
        let front = ViewportCamera::default()
            .facing_sketch(SketchPlane::xy())
            .unwrap();
        assert_eq!(front.up, [0., 1., 0.]);
    }
    #[test]
    fn facing_retains_go_small_distance_policy_and_refuses_overflow() {
        let camera = ViewportCamera {
            eye: [0., 0., 1e-4],
            ..Default::default()
        };
        assert_eq!(
            camera.facing_sketch(SketchPlane::xy()).unwrap().eye,
            [0., 0., 1.]
        );
        let invalid = ViewportCamera {
            eye: [f64::NAN; 3],
            ..camera
        };
        assert!(invalid.facing_sketch(SketchPlane::xy()).is_err());
        let plane = SketchPlane::new(
            [f64::MAX; 3],
            [1., 0., 0.],
            [0., 1., 0.],
            Default::default(),
        )
        .unwrap();
        assert!(camera.facing_sketch(plane).is_err());
    }
}
