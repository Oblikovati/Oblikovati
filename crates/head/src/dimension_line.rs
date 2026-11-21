// SPDX-License-Identifier: GPL-2.0-only
//! Geometry for a linear dimension's displayed measurement line.
use oblikovati_model::sketch::{DistanceMeasure, PlanarConstraint, Sketch2D, SketchConstraintId};

pub(super) fn distance_line(
    geometry: &Sketch2D,
    id: SketchConstraintId,
    label: [f64; 2],
) -> Option<[[f64; 2]; 2]> {
    let relation = &geometry
        .constraints()
        .iter()
        .find(|row| row.id == id)?
        .relation;
    let PlanarConstraint::Distance {
        points, measure, ..
    } = relation
    else {
        return None;
    };
    let [a, b] = points.map(|id| geometry.point(id).ok().map(|point| point.position));
    line_at_label(a?, b?, *measure, label)
}
fn line_at_label(
    a: [f64; 2],
    b: [f64; 2],
    measure: DistanceMeasure,
    label: [f64; 2],
) -> Option<[[f64; 2]; 2]> {
    match measure {
        DistanceMeasure::Horizontal => Some([[a[0], label[1]], [b[0], label[1]]]),
        DistanceMeasure::Vertical => Some([[label[0], a[1]], [label[0], b[1]]]),
        DistanceMeasure::Aligned => aligned_line(a, b, label),
    }
}
fn aligned_line(a: [f64; 2], b: [f64; 2], label: [f64; 2]) -> Option<[[f64; 2]; 2]> {
    let vector = [b[0] - a[0], b[1] - a[1]];
    let length = vector[0].hypot(vector[1]);
    if !length.is_finite() || length == 0. {
        return None;
    }
    let normal = [-vector[1] / length, vector[0] / length];
    let offset = (label[0] - a[0]) * normal[0] + (label[1] - a[1]) * normal[1];
    Some([a, b].map(|point| [point[0] + offset * normal[0], point[1] + offset * normal[1]]))
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn component_and_aligned_lines_follow_the_label() {
        let a = [0., 0.];
        let b = [3., 4.];
        assert_eq!(
            line_at_label(a, b, DistanceMeasure::Horizontal, [1., 6.]),
            Some([[0., 6.], [3., 6.]])
        );
        assert_eq!(
            line_at_label(a, b, DistanceMeasure::Vertical, [6., 1.]),
            Some([[6., 0.], [6., 4.]])
        );
        assert_eq!(aligned_line(a, a, [1., 1.]), None);
        let [first, second] = aligned_line(a, b, [-4., 3.]).unwrap();
        assert_eq!(first, [-4., 3.]);
        assert_eq!(second, [-1., 7.]);
    }
}
