// SPDX-License-Identifier: GPL-2.0-only
use super::{
    PatternOccurrence, Sketch2D, SketchCopyError, SketchEntityId, SketchError,
    SketchPatternOffsetTerm, SketchRectangularPattern, SketchRectangularPatternDefinition,
    SketchRectangularPatternMember,
};
use crate::{
    identity::RecipeIdError,
    parameter::ParameterCollection,
    sketch::{SketchEntityDeletion, SketchPatternOffset, entity_geometry},
};
use std::collections::{BTreeMap, BTreeSet};

/// Edited pattern and the geometry and constraints removed by the edit.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchRectangularPatternEdit {
    pub pattern: SketchRectangularPattern,
    pub deletion: SketchEntityDeletion,
}
impl Sketch2D {
    /// Edit grid settings while preserving identities at retained indices.
    /// Example: `sketch.edit_rectangular_pattern(id, definition, parameters, &mut reserve)?;`.
    pub fn edit_rectangular_pattern(
        &mut self,
        id: SketchEntityId,
        definition: SketchRectangularPatternDefinition,
        parameters: &ParameterCollection<u64>,
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<SketchRectangularPatternEdit, SketchCopyError> {
        let position = self
            .rectangular_patterns
            .iter()
            .position(|pattern| pattern.id == id)
            .ok_or_else(|| {
                SketchCopyError::Sketch(SketchError::Definition(format!(
                    "rectangular pattern {id:?}: expected an existing pattern"
                )))
            })?;
        let old = self.rectangular_patterns[position].clone();
        if old.definition.seed != definition.seed {
            return Err(SketchCopyError::Sketch(SketchError::Definition(format!(
                "rectangular pattern {id:?} seed {:?}: expected unchanged seed {:?}",
                definition.seed, old.definition.seed
            ))));
        }
        definition
            .validate_parameters(parameters)
            .map_err(SketchCopyError::Sketch)?;
        let desired = definition.occurrences().map_err(SketchCopyError::Sketch)?;
        let mut candidate = self.clone();
        candidate.rectangular_patterns.remove(position);
        let deletion = candidate
            .remove_obsolete_occurrences(&old, &desired)
            .map_err(SketchCopyError::Sketch)?;
        let members =
            candidate.rebuild_pattern_members(&old, &definition, &desired, parameters, reserve)?;
        let pattern = SketchRectangularPattern {
            id,
            definition,
            members,
        };
        candidate
            .adopt_rectangular_pattern(pattern.clone())
            .map_err(SketchCopyError::Sketch)?;
        let added = candidate.rectangular_patterns.pop().ok_or_else(|| {
            SketchCopyError::Sketch(SketchError::Definition(format!(
                "pattern edit {id:?}: expected the validated replacement record"
            )))
        })?;
        candidate.rectangular_patterns.insert(position, added);
        *self = candidate;
        Ok(SketchRectangularPatternEdit { pattern, deletion })
    }
    fn remove_obsolete_occurrences(
        &mut self,
        old: &SketchRectangularPattern,
        desired: &[PatternOccurrence],
    ) -> Result<SketchEntityDeletion, SketchError> {
        let retained: BTreeSet<_> = desired.iter().map(|(index, _)| *index).collect();
        let removed: Vec<_> = old
            .members
            .iter()
            .filter(|member| !retained.contains(&member.index))
            .flat_map(|member| member.entities.iter().copied())
            .collect();
        self.delete_entities(&removed)
    }
    fn rebuild_pattern_members(
        &mut self,
        old: &SketchRectangularPattern,
        definition: &SketchRectangularPatternDefinition,
        desired: &[PatternOccurrence],
        parameters: &ParameterCollection<u64>,
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<Vec<SketchRectangularPatternMember>, SketchCopyError> {
        let retained: BTreeMap<_, _> = old
            .members
            .iter()
            .map(|member| (member.index, member))
            .collect();
        let mut members = Vec::with_capacity(desired.len());
        for (index, terms) in desired {
            if let Some(member) = retained.get(index) {
                self.update_member_offsets(&definition.seed, member, terms)
                    .map_err(SketchCopyError::Sketch)?;
                members.push((*member).clone());
            } else {
                let reports = self.copy_parametric_translations(
                    &definition.seed,
                    std::slice::from_ref(terms),
                    parameters,
                    reserve,
                )?;
                let report = reports.into_iter().next().ok_or_else(|| {
                    SketchCopyError::Sketch(SketchError::Definition(format!(
                        "pattern occurrence {index:?}: expected one copy report"
                    )))
                })?;
                members.push(SketchRectangularPatternMember {
                    index: *index,
                    entities: report.copies,
                });
            }
        }
        Ok(members)
    }
    fn update_member_offsets(
        &mut self,
        seed: &[SketchEntityId],
        member: &SketchRectangularPatternMember,
        terms: &[SketchPatternOffsetTerm],
    ) -> Result<(), SketchError> {
        let shapes = entity_geometry::catalogue(self);
        let mut points = BTreeSet::new();
        for (&source, &copy) in seed.iter().zip(&member.entities) {
            for (_, target) in shapes[&source]
                .points()
                .into_iter()
                .zip(shapes[&copy].points())
            {
                points.insert(target);
            }
        }
        for point in points {
            self.set_pattern_offset(SketchPatternOffset {
                member: point,
                terms: terms.to_vec(),
            })?;
        }
        Ok(())
    }
}
