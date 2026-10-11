// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarConstraint, Sketch2D};
impl Sketch2D {
    /// Inspect authored dimension IDs even when resolution fails; example: `sketch.parameter_dependencies()`.
    pub fn parameter_dependencies(&self) -> Vec<u64> {
        let mut dependencies = Vec::new();
        for parameter in self
            .constraints
            .iter()
            .filter_map(|constraint| dimension_parameter(&constraint.relation))
        {
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
