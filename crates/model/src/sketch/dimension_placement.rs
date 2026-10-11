// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchConstraintId, SketchError, SketchRecipe, geometry::require_position};
/// Authored annotation position in sketch-plane working length units.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchDimensionPlacement {
    pub constraint: SketchConstraintId,
    pub position: [f64; 2],
}
impl Sketch2D {
    /// Inspect placement for an owned dimensional relation; missing placement is explicit.
    /// Example: `let text = sketch.dimension_text_point(dimension.id)?;`.
    pub fn dimension_text_point(
        &self,
        id: SketchConstraintId,
    ) -> Result<Option<[f64; 2]>, SketchError> {
        self.require_dimension_placement_owner(id)?;
        Ok(self.dimension_text_points.get(&id).copied())
    }
    /// Move annotation metadata without changing geometry or equations.
    /// Example: `sketch.set_dimension_text_point(dimension.id, [2.0, 3.0])?;`.
    pub fn set_dimension_text_point(
        &mut self,
        id: SketchConstraintId,
        position: [f64; 2],
    ) -> Result<(), SketchError> {
        self.require_dimension_placement_owner(id)?;
        require_position(position)?;
        self.dimension_text_points.insert(id, position);
        Ok(())
    }
    fn require_dimension_placement_owner(&self, id: SketchConstraintId) -> Result<(), SketchError> {
        if self
            .dimensional_constraints()
            .any(|(constraint, _)| constraint.id == id)
        {
            return Ok(());
        }
        Err(SketchError::Definition(format!(
            "annotation constraint {id:?}: expected an owned dimensional relation"
        )))
    }
    pub(super) fn saved_dimension_placements(&self) -> Option<Vec<SketchDimensionPlacement>> {
        if self.dimension_text_points.is_empty() {
            return None;
        }
        Some(
            self.dimensional_constraints()
                .filter_map(|(constraint, _)| {
                    self.dimension_text_points
                        .get(&constraint.id)
                        .map(|position| SketchDimensionPlacement {
                            constraint: constraint.id,
                            position: *position,
                        })
                })
                .collect(),
        )
    }
    pub(super) fn same_dimension_placements(&self, other: &Self) -> bool {
        self.dimension_text_points.len() == other.dimension_text_points.len()
            && self
                .dimension_text_points
                .iter()
                .zip(&other.dimension_text_points)
                .all(|((first_id, first), (second_id, second))| {
                    first_id == second_id && super::same_components(*first, *second)
                })
    }
    pub(super) fn restore_dimension_placements(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        match (recipe.version, &recipe.dimension_placements) {
            (1..=4, None) => Ok(()),
            (5..=11, Some(placements)) => self.adopt_dimension_placements(placements),
            shape => Err(SketchError::Definition(format!(
                "dimension placement recipe {shape:?}: expected no metadata in versions 1-4 or explicit dimensional placements in versions 5-11"
            ))),
        }
    }
    fn adopt_dimension_placements(
        &mut self,
        placements: &[SketchDimensionPlacement],
    ) -> Result<(), SketchError> {
        let mut candidate = self.clone();
        for placement in placements {
            candidate.set_dimension_text_point(placement.constraint, placement.position)?;
        }
        let expected = candidate.saved_dimension_placements().unwrap_or_default();
        if placements != expected {
            return Err(SketchError::Definition(format!(
                "dimension placements {placements:?}: expected unique owned dimensional IDs in authored order {expected:?}"
            )));
        }
        self.dimension_text_points = candidate.dimension_text_points;
        Ok(())
    }
}
