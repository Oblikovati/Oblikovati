// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarConstraint, Sketch2D, SketchConstraint, SketchPoint2D};
impl Sketch2D {
    /// Compare authored recipe bits for history and storage; example: `old.same_recipe(&candidate)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.lines == other.lines
            && self.arcs == other.arcs
            && self.circles == other.circles
            && same_points(&self.points, &other.points)
            && same_constraints(&self.constraints, &other.constraints)
    }
}
pub(super) fn same_components<const N: usize>(first: [f64; N], second: [f64; N]) -> bool {
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
