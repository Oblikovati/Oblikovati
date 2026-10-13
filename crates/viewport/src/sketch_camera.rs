// SPDX-License-Identifier: GPL-2.0-only
use crate::ViewportCamera;
use oblikovati_model::sketch::SketchPlane;
impl ViewportCamera {
    /// Face a sketch from the viewer's current side, preserving scale and standoff.
    /// Example: `view.set_camera(view.camera_state()?.facing_sketch(plane)?)?` on edit entry.
    pub fn facing_sketch(self, plane: SketchPlane) -> Result<Self, String> {
        self.validate()?;
        let frame = plane.recipe();
        let normal = plane.normal();
        let standoff = sketch_standoff(self, frame.origin, normal)?;
        let eye = std::array::from_fn(|axis| frame.origin[axis] + normal[axis] * standoff);
        let up = normalized_up(frame.y_axis);
        let facing = Self {
            eye,
            target: frame.origin,
            up,
            ..self
        };
        facing.validate()?;
        Ok(facing)
    }
}

fn sketch_standoff(
    camera: ViewportCamera,
    origin: [f64; 3],
    normal: [f64; 3],
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
    Ok(if side < 0. { -distance } else { distance })
}
fn normalized_up(axis: [f64; 3]) -> [f64; 3] {
    let magnitude = axis.iter().map(|value| value * value).sum::<f64>().sqrt();
    axis.map(|value| value / magnitude)
}

#[cfg(test)]
mod tests {
    use super::*;
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
