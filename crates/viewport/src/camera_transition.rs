// SPDX-License-Identifier: GPL-2.0-only
//! Interpolate the orientation as one rigid frame so opposite views keep a valid camera.
use crate::ViewportCamera;
use martensite_tkmath::gp::{Matrix3, Quaternion, QuaternionSLerp, Vector3, Xyz};

/// A pair of valid camera frames and their shortest rotation path.
pub struct ViewportCameraTransition {
    start: ViewportCamera,
    end: ViewportCamera,
    rotation: QuaternionSLerp,
    eye_distances: [f64; 2],
}

impl ViewportCameraTransition {
    /// Validate both endpoints. Example: `ViewportCameraTransition::new(front, top)?`.
    pub fn new(start: ViewportCamera, end: ViewportCamera) -> Result<Self, String> {
        let (from, from_distance) = orientation(start)?;
        let (to, to_distance) = orientation(end)?;
        Ok(Self {
            start,
            end,
            rotation: QuaternionSLerp::new(from, to),
            eye_distances: [from_distance, to_distance],
        })
    }
    /// Sample an eased camera. Example: `transition.at(0.5)?` returns the midpoint.
    pub fn at(&self, fraction: f64) -> Result<ViewportCamera, String> {
        if !(0. ..=1.).contains(&fraction) {
            return Err(format!(
                "camera transition progress {fraction:?}: expected a value in [0, 1]"
            ));
        }
        if fraction == 0. {
            return Ok(self.start);
        }
        if fraction == 1. {
            return Ok(self.end);
        }
        let eased = fraction * fraction * (3. - 2. * fraction);
        let blend = |from: f64, to: f64| from + (to - from) * eased;
        let rotation = self.rotation.interpolate(eased);
        let toward = rotation.multiply_vector(Vector3::new(0., 0., 1.));
        let up = rotation.multiply_vector(Vector3::new(0., 1., 0.));
        let distance = blend(self.eye_distances[0], self.eye_distances[1]);
        let target =
            std::array::from_fn(|axis| blend(self.start.target[axis], self.end.target[axis]));
        let eye =
            std::array::from_fn(|axis| target[axis] + toward.xyz().coordinates()[axis] * distance);
        Ok(ViewportCamera {
            eye,
            target,
            up: [up.x(), up.y(), up.z()],
            scale: blend(self.start.scale, self.end.scale),
            near: blend(self.start.near, self.end.near),
            far: blend(self.start.far, self.end.far),
            ..self.start
        })
    }
}

fn orientation(camera: ViewportCamera) -> Result<(Quaternion, f64), String> {
    camera.validate()?;
    let delta: [f64; 3] = std::array::from_fn(|axis| camera.eye[axis] - camera.target[axis]);
    let offset = Xyz::new(delta[0], delta[1], delta[2]);
    let distance = offset.modulus();
    let toward = offset.normalized().map_err(|cause| cause.to_string())?;
    let up = Xyz::new(camera.up[0], camera.up[1], camera.up[2]);
    let right = up
        .crossed(toward)
        .normalized()
        .map_err(|cause| cause.to_string())?;
    let frame = Matrix3::from_columns(right, toward.crossed(right), toward);
    Ok((Quaternion::from_matrix(frame), distance))
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn opposite_view_transition_keeps_a_valid_frame_and_exact_endpoints() {
        let start = ViewportCamera {
            eye: [0., 0., 10.],
            ..ViewportCamera::default()
        };
        let end = ViewportCamera {
            eye: [0., 0., -10.],
            ..start
        };
        let swing = ViewportCameraTransition::new(start, end).unwrap();
        assert_eq!(swing.at(0.).unwrap(), start);
        assert_eq!(swing.at(1.).unwrap(), end);
        for i in 1..10 {
            swing.at(f64::from(i) / 10.).unwrap().validate().unwrap();
        }
        let midway = swing.at(0.5).unwrap();
        assert!(midway.eye[2].abs() < 1e-9);
        assert!((midway.eye[0].abs() - 10.).abs() < 1e-9);
        assert!(swing.at(f64::NAN).is_err());
    }
}
