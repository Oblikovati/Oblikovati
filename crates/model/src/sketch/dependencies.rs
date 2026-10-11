// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarConstraint, Sketch2D};
impl Sketch2D {
    /// Enumerate owned dimensional relations with their backing parameter IDs in authored order.
    /// Example: `sketch.dimensional_constraints().map(|(constraint, parameter)| (constraint.id, parameter))`.
    pub fn dimensional_constraints(&self) -> impl Iterator<Item = (&super::SketchConstraint, u64)> {
        self.constraints.iter().filter_map(|constraint| {
            dimension_parameter(&constraint.relation).map(|parameter| (constraint, parameter))
        })
    }
    /// Inspect authored dimension IDs even when resolution fails; example: `sketch.parameter_dependencies()`.
    pub fn parameter_dependencies(&self) -> Vec<u64> {
        let mut dependencies = Vec::new();
        for (_, parameter) in self.dimensional_constraints() {
            if !dependencies.contains(&parameter) {
                dependencies.push(parameter);
            }
        }
        dependencies
    }
}
pub(super) fn dimension_parameter(relation: &PlanarConstraint) -> Option<u64> {
    match *relation {
        PlanarConstraint::Distance { parameter, .. }
        | PlanarConstraint::Radius { parameter, .. }
        | PlanarConstraint::Diameter { parameter, .. }
        | PlanarConstraint::Offset { parameter, .. }
        | PlanarConstraint::TangentDistance { parameter, .. }
        | PlanarConstraint::ArcLength { parameter, .. }
        | PlanarConstraint::Angle { parameter, .. }
        | PlanarConstraint::ThreePointAngle { parameter, .. } => Some(parameter),
        _ => None,
    }
}
