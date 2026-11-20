// SPDX-License-Identifier: GPL-2.0-only
mod edit;
use super::{
    Sketch2D, SketchCopyError, SketchEntityId, SketchError, SketchPatternOffsetTerm, SketchRecipe,
    SketchRectangularPatternLayout,
};
use crate::{identity::RecipeIdError, parameter::ParameterCollection};
pub use edit::SketchRectangularPatternEdit;
use std::collections::BTreeSet;
type PatternOccurrence = ([u32; 2], Vec<SketchPatternOffsetTerm>);

/// Authored settings for a rectangular pattern; directions contain their sign.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchRectangularPatternDefinition {
    pub seed: Vec<SketchEntityId>,
    pub counts: [u32; 2],
    pub directions: [[f64; 2]; 2],
    pub spacing: [u64; 2],
    pub fitted: bool,
    pub suppressed: BTreeSet<[u32; 2]>,
}
/// One occurrence retains its index and selected copied entities.
#[derive(Clone, Debug, PartialEq, Eq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchRectangularPatternMember {
    pub index: [u32; 2],
    pub entities: Vec<SketchEntityId>,
}
/// A stable owner for an associative rectangular pattern and its members.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct SketchRectangularPattern {
    pub id: SketchEntityId,
    pub definition: SketchRectangularPatternDefinition,
    pub members: Vec<SketchRectangularPatternMember>,
}
impl SketchRectangularPatternDefinition {
    fn validate_parameters(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<(), SketchError> {
        let member = *self.seed.first().ok_or_else(|| {
            SketchError::Definition(
                "rectangular pattern seed []: expected at least one entity".into(),
            )
        })?;
        let offset = super::SketchPatternOffset {
            member,
            terms: self
                .spacing
                .into_iter()
                .map(|parameter| SketchPatternOffsetTerm {
                    parameter,
                    coefficient: [0.0; 2],
                })
                .collect(),
        };
        super::pattern_offset::validate_terms(&offset.terms)?;
        super::pattern_offset::resolve_offset(&offset, parameters)?;
        Ok(())
    }
    fn layout(&self) -> SketchRectangularPatternLayout {
        SketchRectangularPatternLayout {
            counts: self.counts,
            vectors: self.directions,
            fitted: self.fitted,
            suppressed: self.suppressed.clone(),
        }
    }
    fn occurrences(&self) -> Result<Vec<PatternOccurrence>, SketchError> {
        let occurrences = self.layout().occurrences()?;
        if self.spacing.contains(&0) {
            return Err(SketchError::Definition(format!(
                "pattern spacing {:?}: expected two nonzero parameter IDs",
                self.spacing
            )));
        }
        for direction in self.directions {
            let length = direction[0].hypot(direction[1]);
            if !length.is_finite() || (length - 1.0).abs() > 1e-12 {
                return Err(SketchError::Definition(format!(
                    "pattern direction {direction:?}: expected a finite unit vector"
                )));
            }
        }
        Ok(occurrences
            .into_iter()
            .map(|item| {
                let factor = |axis: usize| {
                    if self.fitted && self.counts[axis] > 1 {
                        f64::from(item.index[axis]) / f64::from(self.counts[axis] - 1)
                    } else {
                        f64::from(item.index[axis])
                    }
                };
                let terms = (0..2)
                    .map(|axis| SketchPatternOffsetTerm {
                        parameter: self.spacing[axis],
                        coefficient: self.directions[axis].map(|v| v * factor(axis)),
                    })
                    .collect();
                (item.index, terms)
            })
            .collect())
    }
}
impl Sketch2D {
    /// Create one recorded rectangular pattern atomically; example: `sketch.create_rectangular_pattern(id, definition, parameters, &mut reserve)?;`.
    pub fn create_rectangular_pattern(
        &mut self,
        id: SketchEntityId,
        definition: SketchRectangularPatternDefinition,
        parameters: &ParameterCollection<u64>,
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<SketchRectangularPattern, SketchCopyError> {
        self.require_new_entity(id)
            .map_err(SketchCopyError::Sketch)?;
        definition
            .validate_parameters(parameters)
            .map_err(SketchCopyError::Sketch)?;
        let occurrences = definition.occurrences().map_err(SketchCopyError::Sketch)?;
        let terms: Vec<_> = occurrences.iter().map(|(_, terms)| terms.clone()).collect();
        let mut candidate = self.clone();
        let reports = candidate.copy_parametric_translations(
            &definition.seed,
            &terms,
            parameters,
            reserve,
        )?;
        let members = occurrences
            .into_iter()
            .zip(reports)
            .map(|((index, _), report)| SketchRectangularPatternMember {
                index,
                entities: report.copies,
            })
            .collect();
        let pattern = SketchRectangularPattern {
            id,
            definition,
            members,
        };
        candidate
            .adopt_rectangular_pattern(pattern.clone())
            .map_err(SketchCopyError::Sketch)?;
        *self = candidate;
        Ok(pattern)
    }
    /// Inspect recorded patterns in authored order; example: `sketch.rectangular_patterns()`.
    pub fn rectangular_patterns(&self) -> &[SketchRectangularPattern] {
        &self.rectangular_patterns
    }
    pub(super) fn validate_pattern_parameters(
        &self,
        parameters: &ParameterCollection<u64>,
    ) -> Result<(), SketchError> {
        self.rectangular_patterns
            .iter()
            .try_for_each(|pattern| pattern.definition.validate_parameters(parameters))
    }
    fn adopt_rectangular_pattern(
        &mut self,
        pattern: SketchRectangularPattern,
    ) -> Result<(), SketchError> {
        self.require_new_entity(pattern.id)?;
        let occurrences = pattern.definition.occurrences()?;
        if pattern.definition.seed.is_empty() || occurrences.len() != pattern.members.len() {
            return Err(pattern.invalid("a nonempty seed and one member per visible occurrence"));
        }
        let mut identities = BTreeSet::new();
        for id in &pattern.definition.seed {
            self.entity_kind(*id)?;
            if !identities.insert(*id) {
                return Err(pattern.invalid("unique seed and member entities"));
            }
        }
        for (member, (index, terms)) in pattern.members.iter().zip(occurrences) {
            if member.index != index || member.entities.len() != pattern.definition.seed.len() {
                return Err(pattern.invalid("members in grid order with one copy per seed entity"));
            }
            for id in &member.entities {
                self.entity_kind(*id)?;
                if !identities.insert(*id) {
                    return Err(pattern.invalid("unique seed and member entities"));
                }
            }
            self.validate_pattern_member(&pattern, member, &terms)?;
        }
        self.rectangular_patterns.push(pattern);
        Ok(())
    }
    fn validate_pattern_member(
        &self,
        pattern: &SketchRectangularPattern,
        member: &SketchRectangularPatternMember,
        terms: &[SketchPatternOffsetTerm],
    ) -> Result<(), SketchError> {
        let shapes = super::entity_geometry::catalogue(self);
        for (&source, &copy) in pattern.definition.seed.iter().zip(&member.entities) {
            let original = shapes[&source];
            let copied = shapes[&copy];
            if original.kind() != copied.kind() || original.points().len() != copied.points().len()
            {
                return Err(pattern.invalid(
                    "member geometry with the same kind and defining points as its seed",
                ));
            }
            for (source_point, member_point) in original.points().into_iter().zip(copied.points()) {
                if !self
                    .pattern_point_links
                    .iter()
                    .any(|link| link.source == source_point && link.member == member_point)
                    || !self
                        .pattern_offsets
                        .iter()
                        .any(|offset| offset.member == member_point && offset.terms == terms)
                {
                    return Err(
                        pattern.invalid("point links and offset terms matching each member")
                    );
                }
            }
            if self.circle(source).is_ok()
                && !self
                    .pattern_radius_links
                    .iter()
                    .any(|link| link.source == source && link.member == copy)
            {
                return Err(pattern.invalid("a radius link for every copied circle"));
            }
        }
        Ok(())
    }
    /// Remove a pattern and its copied geometry atomically, leaving its seed intact.
    /// Example: `sketch.remove_rectangular_pattern(pattern.id)?;`.
    pub fn remove_rectangular_pattern(
        &mut self,
        id: SketchEntityId,
    ) -> Result<super::SketchEntityDeletion, SketchError> {
        let index = self
            .rectangular_patterns
            .iter()
            .position(|pattern| pattern.id == id)
            .ok_or_else(|| {
                SketchError::Definition(format!(
                    "rectangular pattern {id:?}: expected an existing pattern"
                ))
            })?;
        let mut candidate = self.clone();
        let pattern = candidate.rectangular_patterns.remove(index);
        let members: Vec<_> = pattern
            .members
            .iter()
            .flat_map(|member| member.entities.iter().copied())
            .collect();
        let deleted = candidate.delete_entities(&members)?;
        *self = candidate;
        Ok(deleted)
    }
    pub(super) fn capture_rectangular_patterns(&self, recipe: &mut SketchRecipe) {
        if self.rectangular_patterns.is_empty() {
            return;
        }
        self.capture_point_ownership_metadata(recipe);
        recipe.version = 16;
        recipe.rectangular_patterns = Some(self.rectangular_patterns.clone());
    }
    pub(super) fn restore_rectangular_patterns(
        &mut self,
        recipe: &SketchRecipe,
    ) -> Result<(), SketchError> {
        match (recipe.version, &recipe.rectangular_patterns) {
            (1..=15, None) | (17..=super::SKETCH_RECIPE_VERSION, None) => Ok(()),
            (16..=super::SKETCH_RECIPE_VERSION, Some(patterns)) if !patterns.is_empty() => patterns
                .iter()
                .try_for_each(|pattern| self.adopt_rectangular_pattern(pattern.clone())),
            shape => Err(SketchError::Definition(format!(
                "rectangular pattern recipe {shape:?}: expected no patterns before version 16 or a nonempty pattern list in version 16"
            ))),
        }
    }
}
impl SketchRectangularPattern {
    fn invalid(&self, expected: &str) -> SketchError {
        SketchError::Definition(format!("rectangular pattern {self:?}: expected {expected}"))
    }
}
