// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarConstraint, Sketch2D, SketchConstraint, SketchPoint2D};
impl Sketch2D {
    /// Compare authored recipe bits for history and storage; example: `old.same_recipe(&candidate)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self.owned_relations == other.owned_relations
            && self.entity_formats == other.entity_formats
            && self.same_rectangular_patterns(other)
            && self.same_pattern_offsets(other)
            && self.pattern_radius_links == other.pattern_radius_links
            && self.pattern_point_links.len() == other.pattern_point_links.len()
            && self
                .pattern_point_links
                .iter()
                .zip(&other.pattern_point_links)
                .all(|(a, b)| {
                    a.source == b.source
                        && a.member == b.member
                        && same_components(a.offset, b.offset)
                })
            && self.same_dimension_settings(other)
            && self.same_dimension_placements(other)
            && self.centerlines == other.centerlines
            && self.center_points == other.center_points
            && self.construction_curves == other.construction_curves
            && self.entity_order == other.entity_order
            && self.defining_points == other.defining_points
            && self.lines == other.lines
            && self.arcs == other.arcs
            && self.circles == other.circles
            && same_ellipses(&self.ellipses, &other.ellipses)
            && self.same_spline_recipes(other)
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

impl Sketch2D {
    fn same_rectangular_patterns(&self, other: &Self) -> bool {
        self.rectangular_patterns.len() == other.rectangular_patterns.len()
            && self
                .rectangular_patterns
                .iter()
                .zip(&other.rectangular_patterns)
                .all(|(a, b)| {
                    a.id == b.id
                        && a.members == b.members
                        && a.definition.seed == b.definition.seed
                        && a.definition.counts == b.definition.counts
                        && a.definition.spacing == b.definition.spacing
                        && a.definition.fitted == b.definition.fitted
                        && a.definition.suppressed == b.definition.suppressed
                        && a.definition
                            .directions
                            .into_iter()
                            .zip(b.definition.directions)
                            .all(|(x, y)| same_components(x, y))
                })
    }
    fn same_pattern_offsets(&self, other: &Self) -> bool {
        self.pattern_offsets.len() == other.pattern_offsets.len()
            && self
                .pattern_offsets
                .iter()
                .zip(&other.pattern_offsets)
                .all(|(a, b)| {
                    a.member == b.member
                        && a.terms.len() == b.terms.len()
                        && a.terms.iter().zip(&b.terms).all(|(x, y)| {
                            x.parameter == y.parameter
                                && same_components(x.coefficient, y.coefficient)
                        })
                })
    }
}
