// SPDX-License-Identifier: GPL-2.0-only
use super::{Sketch2D, SketchEntityId, SketchError, SketchRecipe, require_position};
use crate::{parameter::ParameterCollection, quantity::Dimension};

/// One length parameter and its signed grid-direction multiplier.
#[derive(Clone, Copy, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchPatternOffsetTerm {
    pub parameter: u64,
    pub coefficient: [f64; 2],
}
/// The parameter-driven offset for one structurally linked member point.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchPatternOffset {
    pub member: SketchEntityId,
    pub terms: Vec<SketchPatternOffsetTerm>,
}
impl Sketch2D {
    /// Bind a member point to parameter-driven offset terms; example: `sketch.set_pattern_offset(offset)?;`.
    pub fn set_pattern_offset(&mut self, offset: SketchPatternOffset) -> Result<(), SketchError> {
        if self.rectangular_patterns.iter().any(|pattern| {
            let shapes = super::entity_geometry::catalogue(self);
            pattern
                .members
                .iter()
                .flat_map(|member| &member.entities)
                .any(|id| {
                    shapes
                        .get(id)
                        .is_some_and(|shape| shape.points().contains(&offset.member))
                })
        }) {
            return Err(offset.invalid("an offset not owned by a recorded pattern"));
        }
        if !self
            .pattern_point_links
            .iter()
            .any(|link| link.member == offset.member)
        {
            return Err(offset.invalid("a member with an existing pattern point link"));
        }
        validate_terms(&offset.terms)?;
        if let Some(existing) = self
            .pattern_offsets
            .iter_mut()
            .find(|existing| existing.member == offset.member)
        {
            *existing = offset;
        } else {
            self.pattern_offsets.push(offset);
        }
        Ok(())
    }
    /// Inspect stored pattern offset formulas; example: `sketch.pattern_offsets()`.
    pub fn pattern_offsets(&self) -> &[SketchPatternOffset] {
        &self.pattern_offsets
    }
    pub(super) fn capture_pattern_offsets(&self, recipe: &mut SketchRecipe) {
        if self.pattern_offsets.is_empty() {
            return;
        }
        recipe.version = 15;
        recipe.pattern_offsets = Some(self.pattern_offsets.clone());
    }
    pub(super) fn restore_pattern_offsets(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        match (recipe.version, &recipe.pattern_offsets) {
            (1..=14, None) | (16..=20, None) => Ok(()),
            (15..=20, Some(offsets)) if !offsets.is_empty() => {
                for offset in offsets {
                    if self
                        .pattern_offsets
                        .iter()
                        .any(|existing| existing.member == offset.member)
                    {
                        return Err(offset.invalid("one formula per member"));
                    }
                    self.set_pattern_offset(offset.clone())?;
                }
                Ok(())
            }
            shape => Err(SketchError::Definition(format!(
                "pattern offset recipe {shape:?}: expected no formulas before version 15 or a nonempty formula list in version 15"
            ))),
        }
    }
}
impl SketchPatternOffset {
    fn invalid(&self, expected: &str) -> SketchError {
        SketchError::Definition(format!("pattern offset {self:?}: expected {expected}"))
    }
}
pub(super) fn validate_terms(terms: &[SketchPatternOffsetTerm]) -> Result<(), SketchError> {
    if terms.is_empty() {
        return Err(SketchError::Definition(
            "pattern offset terms []: expected at least one length parameter".into(),
        ));
    }
    for term in terms {
        if term.parameter == 0 {
            return Err(SketchError::Definition(format!(
                "pattern offset term {term:?}: expected a nonzero parameter identity"
            )));
        }
        require_position(term.coefficient)?;
    }
    Ok(())
}
pub(super) fn resolve_offset(
    offset: &SketchPatternOffset,
    parameters: &ParameterCollection<u64>,
) -> Result<[f64; 2], SketchError> {
    let mut value = [0.0; 2];
    for term in &offset.terms {
        let quantity = parameters.model_value(term.parameter).map_err(|cause| {
            SketchError::PatternParameter {
                member: offset.member,
                parameter: term.parameter,
                cause: Box::new(cause),
            }
        })?;
        if quantity.dimension != Dimension::Length
            || !quantity.value.is_finite()
            || quantity.value < 0.0
        {
            return Err(offset.invalid(&format!(
                "parameter {} to be a finite nonnegative length, found {quantity:?}",
                term.parameter
            )));
        }
        for (coordinate, coefficient) in value.iter_mut().zip(term.coefficient) {
            *coordinate += coefficient * quantity.value;
        }
        require_position(value)?;
    }
    Ok(value)
}
