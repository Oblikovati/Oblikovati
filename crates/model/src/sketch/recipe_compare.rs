// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarConstraint, Sketch2D, SketchConstraint, SketchPoint2D};
impl Sketch2D {
    /// Compare authored recipe bits for history and storage; example: `old.same_recipe(&candidate)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.same_dimension_settings(other)
            && self.same_dimension_placements(other)
            && self.centerlines == other.centerlines
            && self.construction_curves == other.construction_curves
            && self.entity_order == other.entity_order
            && self.lines == other.lines
            && self.arcs == other.arcs
            && self.circles == other.circles
            && same_ellipses(&self.ellipses, &other.ellipses)
            && same_points(&self.points, &other.points)
            && same_constraints(&self.constraints, &other.constraints)
    }
}
pub(crate) fn same_components<const N: usize>(first: [f64; N], second: [f64; N]) -> bool {
    first
        .into_iter()
        .zip(second)
        .all(|(first, second)| first.to_bits() == second.to_bits())
}
fn same_points(first: &[SketchPoint2D], second: &[SketchPoint2D]) -> bool {
    first.len() == second.len()
        && first.iter().zip(second).all(|(first, second)| {
            first.id == second.id && same_components(first.position, second.position)
        })
}
fn same_constraints(first: &[SketchConstraint], second: &[SketchConstraint]) -> bool {
    first.len() == second.len()
        && first.iter().zip(second).all(|(first, second)| {
            first.id == second.id && same_relation(&first.relation, &second.relation)
        })
}
fn same_relation(first: &PlanarConstraint, second: &PlanarConstraint) -> bool {
    match (first, second) {
        (
            PlanarConstraint::Ground { points: first },
            PlanarConstraint::Ground { points: second },
        ) => same_points(first, second),
        (
            PlanarConstraint::Fix {
                point: first_point,
                position: first,
            },
            PlanarConstraint::Fix {
                point: second_point,
                position: second,
            },
        ) => first_point == second_point && same_components(*first, *second),
        _ => first == second,
    }
}

fn same_ellipses(first: &[super::SketchEllipse2D], second: &[super::SketchEllipse2D]) -> bool {
    first.len() == second.len()
        && first.iter().zip(second).all(|(first, second)| {
            first.id() == second.id()
                && first.center() == second.center()
                && same_components(first.major_axis(), second.major_axis())
                && same_components(first.radii(), second.radii())
        })
}
