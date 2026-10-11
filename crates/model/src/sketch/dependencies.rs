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
    /// Resolve an owned dimension's target identity and numeric category without measuring geometry.
    /// Example: `let (parameter, category) = sketch.dimension_source(dimension.id)?;`.
    pub fn dimension_source(
        &self,
        id: super::SketchConstraintId,
    ) -> Result<(u64, crate::quantity::Dimension), super::SketchError> {
        let (row, parameter) = self
            .dimensional_constraints()
            .find(|(row, _)| row.id == id)
            .ok_or_else(|| {
                super::SketchError::Definition(format!(
                    "sketch constraint {id:?}: expected an owned dimensional source"
                ))
            })?;
        let category = dimension_category(&row.relation);
        Ok((parameter, category))
    }
    /// Remove a dimensional relation while retaining its parameter for the document owner cascade.
    /// Example: `sketch.remove_dimensional_constraint(selected.id)?;` on an isolated candidate.
    pub fn remove_dimensional_constraint(
        &mut self,
        id: super::SketchConstraintId,
    ) -> Result<super::SketchConstraint, super::SketchError> {
        self.remove_authored_relation(id, true)
    }
    /// Inspect driving target IDs even when resolution fails; example: `sketch.parameter_dependencies()`.
    pub fn parameter_dependencies(&self) -> Vec<u64> {
        let mut dependencies = Vec::new();
        for (constraint, parameter) in self.dimensional_constraints() {
            if self.reference_dimensions.contains(&constraint.id) {
                continue;
            }
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

pub(super) fn dimension_category(relation: &PlanarConstraint) -> crate::quantity::Dimension {
    if matches!(
        relation,
        PlanarConstraint::Angle { .. } | PlanarConstraint::ThreePointAngle { .. }
    ) {
        return crate::quantity::Dimension::Angle;
    }
    crate::quantity::Dimension::Length
}
