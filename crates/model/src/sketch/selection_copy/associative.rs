// SPDX-License-Identifier: GPL-2.0-only
use super::{
    OwnedSketchGeometry, Sketch2D, SketchCopyError, SketchCopyOperation, SketchCopyReport,
    SketchEntityId, SketchError, copy_selection, entity_geometry,
};
use crate::{
    identity::RecipeIdError,
    sketch::{SketchPatternPointLink, SketchPatternRadiusLink},
};
impl Sketch2D {
    /// Copy and link instances with offsets driven by the supplied length parameters.
    /// Example: `sketch.copy_parametric_translations(&[circle], &[terms], parameters, &mut reserve)?;`.
    pub fn copy_parametric_translations(
        &mut self,
        entities: &[SketchEntityId],
        occurrences: &[Vec<crate::sketch::SketchPatternOffsetTerm>],
        parameters: &crate::parameter::ParameterCollection<u64>,
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<Vec<SketchCopyReport>, SketchCopyError> {
        self.validate_associative_selection(entities)
            .map_err(SketchCopyError::Sketch)?;
        let offsets = resolve_occurrences(entities[0], occurrences, parameters)
            .map_err(SketchCopyError::Sketch)?;
        let mut candidate = self.clone();
        let reports = candidate.copy_associative_translations(entities, &offsets, reserve)?;
        self.bind_copy_offsets(&mut candidate, &reports, occurrences)
            .map_err(SketchCopyError::Sketch)?;
        *self = candidate;
        Ok(reports)
    }
    fn bind_copy_offsets(
        &self,
        candidate: &mut Self,
        reports: &[SketchCopyReport],
        occurrences: &[Vec<crate::sketch::SketchPatternOffsetTerm>],
    ) -> Result<(), SketchError> {
        for (report, terms) in reports.iter().zip(occurrences) {
            for &(source, member) in &report.relations {
                if self.point(source).is_ok() {
                    candidate.set_pattern_offset(crate::sketch::SketchPatternOffset {
                        member,
                        terms: terms.clone(),
                    })?;
                }
            }
        }
        Ok(())
    }
    /// Copy translated selections and link every supported member to its seed atomically.
    /// Example: `sketch.copy_associative_translations(&[circle], &[[10.0, 0.0]], &mut reserve)?;`.
    pub fn copy_associative_translations(
        &mut self,
        entities: &[SketchEntityId],
        offsets: &[[f64; 2]],
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<Vec<SketchCopyReport>, SketchCopyError> {
        self.validate_associative_selection(entities)
            .map_err(SketchCopyError::Sketch)?;
        for offset in offsets {
            super::super::require_position(*offset).map_err(SketchCopyError::Sketch)?;
        }
        let operations: Vec<_> = offsets
            .iter()
            .copied()
            .map(SketchCopyOperation::Translate)
            .collect();
        let mut candidate = self.clone();
        let reports = candidate.copy_selected_entities_batch(entities, &operations, reserve)?;
        for (report, offset) in reports.iter().zip(offsets) {
            candidate
                .link_translation(report, *offset)
                .map_err(SketchCopyError::Sketch)?;
        }
        *self = candidate;
        Ok(reports)
    }
    fn validate_associative_selection(
        &self,
        entities: &[SketchEntityId],
    ) -> Result<(), SketchError> {
        let selected = copy_selection(self, entities)?;
        if selected.is_empty() {
            return Err(SketchError::Definition(format!(
                "pattern seed {entities:?}: expected at least one supported entity"
            )));
        }
        let catalogue = entity_geometry::catalogue(self);
        for id in selected {
            if matches!(catalogue.get(&id), Some(OwnedSketchGeometry::Ellipse(_))) {
                return Err(SketchError::Definition(format!(
                    "pattern seed {id:?}: expected geometry with supported associative variables; ellipse links are unavailable"
                )));
            }
        }
        Ok(())
    }
    fn link_translation(
        &mut self,
        report: &SketchCopyReport,
        offset: [f64; 2],
    ) -> Result<(), SketchError> {
        for &(source, member) in &report.relations {
            if self.point(source).is_ok() {
                self.add_pattern_point_link(SketchPatternPointLink {
                    source,
                    member,
                    offset,
                })?;
            } else if self.circle(source).is_ok() {
                self.add_pattern_radius_link(SketchPatternRadiusLink { source, member })?;
            }
        }
        Ok(())
    }
}

fn resolve_occurrences(
    member: SketchEntityId,
    occurrences: &[Vec<crate::sketch::SketchPatternOffsetTerm>],
    parameters: &crate::parameter::ParameterCollection<u64>,
) -> Result<Vec<[f64; 2]>, SketchError> {
    occurrences
        .iter()
        .map(|terms| {
            super::super::pattern_offset::validate_terms(terms)?;
            super::super::pattern_offset::resolve_offset(
                &crate::sketch::SketchPatternOffset {
                    member,
                    terms: terms.clone(),
                },
                parameters,
            )
        })
        .collect()
}
