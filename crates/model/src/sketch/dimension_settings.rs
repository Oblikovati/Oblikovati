// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchConstraintId, SketchError, SketchRecipe};
/// Finite target bounds in the dimension's working units, applied only on drive.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchDimensionLimits {
    pub lower: f64,
    pub upper: f64,
}
/// Persisted bounds belong to a permanent dimensional relation.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchDimensionLimitRecipe {
    pub constraint: SketchConstraintId,
    pub limits: SketchDimensionLimits,
}
impl SketchDimensionLimits {
    /// Check finite ordered metric bounds; example: `limits.validate()?;` before adoption.
    pub fn validate(self) -> Result<(), SketchError> {
        if self.lower.is_finite() && self.upper.is_finite() && self.lower <= self.upper {
            return Ok(());
        }
        Err(SketchError::Definition(format!(
            "dimension limits {self:?}: expected finite ordered lower <= upper bounds"
        )))
    }
    /// Clamp a checked finite target while preserving in-range bits; example: `let target = limits.clamp(4.0);`.
    pub fn clamp(self, value: f64) -> f64 {
        if value < self.lower {
            return self.lower;
        }
        if value > self.upper {
            return self.upper;
        }
        value
    }
}
impl Sketch2D {
    /// Inspect whether a dimension measures without constraining geometry.
    /// Example: `sketch.dimension_is_reference(dimension.id)?;`.
    pub fn dimension_is_reference(&self, id: SketchConstraintId) -> Result<bool, SketchError> {
        self.dimension_source(id)?;
        Ok(self.reference_dimensions.contains(&id))
    }
    /// Change the role without changing the target expression or measured geometry.
    /// Example: `sketch.set_dimension_reference(dimension.id, true)?;`.
    pub fn set_dimension_reference(
        &mut self,
        id: SketchConstraintId,
        reference: bool,
    ) -> Result<(), SketchError> {
        self.dimension_source(id)?;
        if reference {
            self.reference_dimensions.insert(id);
        } else {
            self.reference_dimensions.remove(&id);
        }
        Ok(())
    }
    /// Inspect enabled target bounds; example: `sketch.dimension_limits(dimension.id)?;`.
    pub fn dimension_limits(
        &self,
        id: SketchConstraintId,
    ) -> Result<Option<SketchDimensionLimits>, SketchError> {
        self.dimension_source(id)?;
        Ok(self.dimension_drive_limits.get(&id).copied())
    }
    /// Set bounds without clamping an existing target; example: `sketch.set_dimension_limits(id, limits)?;`.
    pub fn set_dimension_limits(
        &mut self,
        id: SketchConstraintId,
        limits: SketchDimensionLimits,
    ) -> Result<(), SketchError> {
        self.dimension_source(id)?;
        limits.validate()?;
        self.dimension_drive_limits.insert(id, limits);
        Ok(())
    }
    pub(super) fn saved_reference_dimensions(&self) -> Option<Vec<SketchConstraintId>> {
        if self.reference_dimensions.is_empty() {
            return None;
        }
        Some(
            self.dimensional_constraints()
                .map(|(row, _)| row.id)
                .filter(|id| self.reference_dimensions.contains(id))
                .collect(),
        )
    }
    pub(super) fn saved_dimension_limits(&self) -> Option<Vec<SketchDimensionLimitRecipe>> {
        if self.dimension_drive_limits.is_empty() {
            return None;
        }
        Some(
            self.dimensional_constraints()
                .filter_map(|(row, _)| {
                    self.dimension_drive_limits.get(&row.id).map(|limits| {
                        SketchDimensionLimitRecipe {
                            constraint: row.id,
                            limits: *limits,
                        }
                    })
                })
                .collect(),
        )
    }
    pub(super) fn same_dimension_settings(&self, other: &Self) -> bool {
        self.reference_dimensions == other.reference_dimensions
            && self.dimension_drive_limits.len() == other.dimension_drive_limits.len()
            && self
                .dimension_drive_limits
                .iter()
                .zip(&other.dimension_drive_limits)
                .all(|((first_id, first), (second_id, second))| {
                    first_id == second_id
                        && super::same_components(
                            [first.lower, first.upper],
                            [second.lower, second.upper],
                        )
                })
    }
    pub(super) fn restore_dimension_settings(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        match (
            recipe.version,
            &recipe.reference_dimensions,
            &recipe.dimension_limits,
        ) {
            (1..=5, None, None) => Ok(()),
            (6..=13, Some(ids), Some(limits)) => self.adopt_dimension_settings(ids, limits),
            shape => Err(SketchError::Definition(format!(
                "dimension settings recipe {shape:?}: expected no settings in versions 1-5 or explicit owned reference IDs and limits in versions 6-12"
            ))),
        }
    }
    fn adopt_dimension_settings(
        &mut self,
        ids: &[SketchConstraintId],
        limits: &[SketchDimensionLimitRecipe],
    ) -> Result<(), SketchError> {
        let mut candidate = self.clone();
        for id in ids {
            candidate.set_dimension_reference(*id, true)?;
        }
        for row in limits {
            candidate.set_dimension_limits(row.constraint, row.limits)?;
        }
        let expected_ids = candidate.saved_reference_dimensions().unwrap_or_default();
        let expected_limits = candidate.saved_dimension_limits().unwrap_or_default();
        if ids != expected_ids || limits != expected_limits {
            return Err(SketchError::Definition(format!(
                "dimension settings {ids:?} {limits:?}: expected unique owned IDs in authored order {expected_ids:?} {expected_limits:?}"
            )));
        }
        self.reference_dimensions = candidate.reference_dimensions;
        self.dimension_drive_limits = candidate.dimension_drive_limits;
        Ok(())
    }
    pub(super) fn capture_dimension_settings_schema(&self, recipe: &mut SketchRecipe) {
        if recipe.reference_dimensions.is_none() && recipe.dimension_limits.is_none() {
            return;
        }
        recipe.version = 6;
        self.capture_dimension_schema_metadata(recipe);
    }
}
