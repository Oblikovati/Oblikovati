// SPDX-License-Identifier: GPL-2.0-only
//! Analytic work-axis line constructors from planes, points, and reference axes (#3918).
use super::{WorkAxisError, WorkAxisLine, require_axis_origin};
use crate::sketch::{SketchError, SketchPlane};

const PARALLEL_PLANE_EPSILON: f64 = 1e-12;
const POINT_ON_AXIS_EPSILON: f64 = 1e-9;

impl WorkAxisLine {
    /// Compute the exact intersection line of two non-parallel planes.
    /// Example: `WorkAxisLine::plane_intersection(SketchPlane::xy(), SketchPlane::xz())?`.
    pub fn plane_intersection(
        first: SketchPlane,
        second: SketchPlane,
    ) -> Result<Self, WorkAxisError> {
        let n1 = first.normal();
        let n2 = second.normal();
        let dir = cross(n1, n2);
        let denom = dot(dir, dir);
        if !denom.is_finite() || denom <= PARALLEL_PLANE_EPSILON {
            return Err(WorkAxisError::Direction {
                direction: dir,
                cause: SketchError::Definition(
                    "plane intersection: expected two non-parallel planes".into(),
                ),
            });
        }
        let d1 = dot(n1, first.recipe().origin);
        let d2 = dot(n2, second.recipe().origin);
        let term = sub(scale(n2, d1), scale(n1, d2));
        let origin = scale(cross(term, dir), 1.0 / denom);
        Self::new(origin, dir)
    }

    /// Compute the line passing through `point` along the normal of `plane`.
    /// Example: `WorkAxisLine::normal_to_plane_through_point(SketchPlane::xy(), [1., 2., 3.])?`.
    pub fn normal_to_plane_through_point(
        plane: SketchPlane,
        point: [f64; 3],
    ) -> Result<Self, WorkAxisError> {
        require_axis_origin(point)?;
        Self::new(point, plane.normal())
    }

    /// Compute the line passing through `point` parallel to `axis`, rejecting points on `axis`.
    /// Example: `WorkAxisLine::parallel_to_line_through_point(z_axis, [1., 0., 0.])?`.
    pub fn parallel_to_line_through_point(
        axis: WorkAxisLine,
        point: [f64; 3],
    ) -> Result<Self, WorkAxisError> {
        require_axis_origin(point)?;
        let offset = sub(point, axis.origin());
        let perp = cross(offset, axis.direction());
        let distance = dot(perp, perp).sqrt();
        if !distance.is_finite() || distance <= POINT_ON_AXIS_EPSILON {
            return Err(WorkAxisError::Origin(point));
        }
        Self::new(point, axis.direction())
    }
}

fn dot(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

fn cross(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}

fn sub(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
}

fn scale(v: [f64; 3], factor: f64) -> [f64; 3] {
    [v[0] * factor, v[1] * factor, v[2] * factor]
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn plane_intersection_resolves_origin_planes_and_offset_planes() {
        let xy = SketchPlane::xy();
        let xz = SketchPlane::xz();
        let axis = WorkAxisLine::plane_intersection(xy, xz).unwrap();
        assert!(axis.origin().iter().all(|c| c.abs() < 1e-12));
        assert!((axis.direction()[0].abs() - 1.0).abs() < 1e-12);
        assert!(axis.direction()[1].abs() < 1e-12);
        assert!(axis.direction()[2].abs() < 1e-12);

        let offset_xy = SketchPlane::from_directions(
            [0.0, 0.0, 4.0],
            [1.0, 0.0, 0.0],
            [0.0, 1.0, 0.0],
            Default::default(),
        )
        .unwrap();
        let offset_yz = SketchPlane::from_directions(
            [3.0, 0.0, 0.0],
            [0.0, 1.0, 0.0],
            [0.0, 0.0, 1.0],
            Default::default(),
        )
        .unwrap();
        let line = WorkAxisLine::plane_intersection(offset_xy, offset_yz).unwrap();
        assert!((line.origin()[0] - 3.0).abs() < 1e-12);
        assert!((line.origin()[2] - 4.0).abs() < 1e-12);
        assert!((line.direction()[1].abs() - 1.0).abs() < 1e-12);

        assert!(WorkAxisLine::plane_intersection(xy, offset_xy).is_err());
    }

    #[test]
    fn normal_to_plane_and_parallel_to_line_validate_inputs() {
        let normal =
            WorkAxisLine::normal_to_plane_through_point(SketchPlane::xy(), [2.0, -3.0, 5.0])
                .unwrap();
        assert_eq!(normal.origin(), [2.0, -3.0, 5.0]);
        assert_eq!(normal.direction(), [0.0, 0.0, 1.0]);

        let z_axis = WorkAxisLine::new([0.0; 3], [0.0, 0.0, 1.0]).unwrap();
        let parallel =
            WorkAxisLine::parallel_to_line_through_point(z_axis, [1.5, -2.0, 4.0]).unwrap();
        assert_eq!(parallel.origin(), [1.5, -2.0, 4.0]);
        assert_eq!(parallel.direction(), [0.0, 0.0, 1.0]);

        assert!(WorkAxisLine::parallel_to_line_through_point(z_axis, [0.0, 0.0, 9.0]).is_err());
    }
}
