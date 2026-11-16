// SPDX-License-Identifier: GPL-2.0-only
//! Conditional line segment and direction error bounds.
use super::ConditionalProjectedSketchRoot;
use crate::sketch::{SketchEntityId, interval_equations::enclosure::Enclosure};

/// Bounds for corresponding points and directions of one authored line segment.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct ConditionalSegmentError {
    pub line: SketchEntityId,
    pub endpoints: [SketchEntityId; 2],
    /// Distance between equal-parameter points, in document length units.
    pub displacement_upper: f64,
    /// Angle between oriented segment directions, in radians.
    pub angle_upper_radians: f64,
}
impl ConditionalProjectedSketchRoot<'_, '_> {
    /// Bound line displacement and angle using source-attributed endpoints.
    /// Example: `root.canonical_segment_errors()?`.
    /// Both segments use the same affine parameter and endpoint order.
    pub fn canonical_segment_errors(&self) -> Result<Vec<ConditionalSegmentError>, String> {
        let points = self.root.sketch().points();
        let errors = self.canonical_point_displacements()?;
        self.root
            .sketch()
            .lines()
            .iter()
            .map(|line| {
                let mut centers = [[0.; 2]; 2];
                let mut displacement = [0.; 2];
                for (axis, id) in line.endpoints.iter().enumerate() {
                    let index =
                        points
                            .iter()
                            .position(|point| point.id == *id)
                            .ok_or_else(|| {
                                format!(
                                    "line {:?} endpoint {id:?}: expected retained point",
                                    line.id
                                )
                            })?;
                    centers[axis] = points[index].position;
                    displacement[axis] = errors[index].upper;
                }
                let angle = direction_bound(centers, displacement)
                    .map_err(|cause| format!("line {:?}: {cause}", line.id))?;
                Ok(ConditionalSegmentError {
                    line: line.id,
                    endpoints: line.endpoints,
                    displacement_upper: displacement[0].max(displacement[1]),
                    angle_upper_radians: angle,
                })
            })
            .collect()
    }
}
fn point(value: f64) -> Result<Enclosure, String> {
    Enclosure::new([value, value])
}
fn axis_lower(a: f64, b: f64) -> Result<f64, String> {
    let [lo, hi] = point(b)?.difference(point(a)?)?.bounds();
    Ok(if lo > 0. {
        lo
    } else if hi < 0. {
        -hi
    } else {
        0.
    })
}
fn direction_bound(centers: [[f64; 2]; 2], errors: [f64; 2]) -> Result<f64, String> {
    let length_lower =
        axis_lower(centers[0][0], centers[1][0])?.max(axis_lower(centers[0][1], centers[1][1])?);
    let error = point(errors[0])?.sum(point(errors[1])?)?;
    let twice_error = error.product(point(2.)?)?.bounds()[1];
    if length_lower <= twice_error {
        return Err(format!(
            "direction may be degenerate or reversed: length lower {length_lower:?}, twice error {twice_error:?}"
        ));
    }
    // |unit(v)-unit(w)| <= 2|v-w|/|v| and angle <= 2|unit(v)-unit(w)|.
    Ok(error
        .product(point(4.)?)?
        .quotient(point(length_lower)?)?
        .bounds()[1])
}

#[cfg(test)]
mod tests {
    use super::*;
    use num_rational::BigRational;
    fn q(v: f64) -> BigRational {
        BigRational::from_float(v).unwrap()
    }
    #[test]
    fn angle_bound_covers_independent_rational_slope_oracle() {
        // For this test angle = atan(|dy/dx|) <= |dy/dx|.
        for (length, error) in [(10., 0.125), (1e100, 1e90), (1e-100, 1e-110)] {
            let bound = direction_bound([[0., 0.], [length, 0.]], [0., error]).unwrap();
            assert!(q(bound) >= q(error) / q(length));
        }
    }
    #[test]
    fn refuses_zero_length_and_possible_direction_reversal() {
        assert!(direction_bound([[0., 0.], [0., 0.]], [0., 0.]).is_err());
        assert!(direction_bound([[0., 0.], [1., 0.]], [0.25, 0.25]).is_err());
        assert!(direction_bound([[0., 0.], [1e-200, 0.]], [1e-200, 0.]).is_err());
        assert!(direction_bound([[0., 0.], [f64::INFINITY, 0.]], [0., 0.]).is_err());
    }
}
