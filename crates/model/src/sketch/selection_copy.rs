// SPDX-License-Identifier: GPL-2.0-only
mod adopt;
mod associative;
mod identities;
mod operation;
mod report;
use super::{
    Sketch2D, SketchEntityId, SketchError,
    entity_geometry::{self, OwnedSketchGeometry},
};
use crate::identity::RecipeIdError;
use identities::CopyIdentities;
pub use operation::SketchCopyOperation;
pub use report::{SketchCopyError, SketchCopyReport};
use std::collections::BTreeSet;
struct PreparedSelectionCopy {
    source: Sketch2D,
    selected: Vec<SketchEntityId>,
    sources: Vec<SketchEntityId>,
}
impl Sketch2D {
    /// Copy another sketch's selection; example: `target.copy_transformed_selection(&source, &[line], transform, &mut reserve)?`.
    pub fn copy_transformed_selection(
        &mut self,
        source: &Sketch2D,
        entities: &[SketchEntityId],
        transform: super::SketchSelectionTransform,
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<SketchCopyReport, SketchCopyError> {
        let plan = PreparedSelectionCopy::transformed(source, entities, transform)
            .map_err(SketchCopyError::Sketch)?;
        let mut candidate = self.clone();
        let report = plan.append_reserved(&mut candidate, reserve)?;
        *self = candidate;
        Ok(report)
    }
    /// Capture selected geometry in its own identity namespace without changing this sketch.
    /// Example: `let geometry = sketch.capture_selection(&[line])?;` for a block definition.
    pub fn capture_selection(&self, entities: &[SketchEntityId]) -> Result<Sketch2D, SketchError> {
        let selected = copy_selection(self, entities)?;
        if selected.is_empty() {
            return Err(SketchError::Definition(format!(
                "capture selection {entities:?}: expected at least one entity"
            )));
        }
        let sources = capture_sources(self, &selected);
        let plan = PreparedSelectionCopy {
            source: self.clone(),
            selected,
            sources,
        };
        let mut target = Self::default();
        plan.append(&mut target, &CopyIdentities::local(&plan.sources))?;
        Ok(target)
    }
    /// Copy geometry with shared points and fresh identities; constraints stay on the source.
    /// Example: `sketch.copy_entities(&[line], [2.0, 0.0], &mut reserve)?;`.
    pub fn copy_entities(
        &mut self,
        entities: &[SketchEntityId],
        vector: [f64; 2],
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<SketchCopyReport, SketchCopyError> {
        self.copy_selected_entities(entities, SketchCopyOperation::Translate(vector), reserve)
    }
    /// Copy through one checked transform and identity path; example: `sketch.copy_selected_entities(&[arc], SketchCopyOperation::Mirror(line), &mut reserve)?;`.
    pub fn copy_selected_entities(
        &mut self,
        entities: &[SketchEntityId],
        operation: SketchCopyOperation,
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<SketchCopyReport, SketchCopyError> {
        let plan = PreparedSelectionCopy::new(self, entities, operation)
            .map_err(SketchCopyError::Sketch)?;
        let mut candidate = self.clone();
        let report = plan.append_reserved(&mut candidate, reserve)?;
        *self = candidate;
        Ok(report)
    }
    /// Copy each instance from the original selection and publish the whole batch together.
    /// Example: `sketch.copy_selected_entities_batch(&[line], &[operation], &mut reserve)?;`.
    pub fn copy_selected_entities_batch(
        &mut self,
        entities: &[SketchEntityId],
        operations: &[SketchCopyOperation],
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<Vec<SketchCopyReport>, SketchCopyError> {
        copy_selection(self, entities).map_err(SketchCopyError::Sketch)?;
        let mut candidate = self.clone();
        let mut reports = Vec::with_capacity(operations.len());
        for &operation in operations {
            let plan = PreparedSelectionCopy::new(self, entities, operation)
                .map_err(SketchCopyError::Sketch)?;
            reports.push(plan.append_reserved(&mut candidate, reserve)?);
        }
        *self = candidate;
        Ok(reports)
    }
}
fn capture_sources(source: &Sketch2D, selected: &[SketchEntityId]) -> Vec<SketchEntityId> {
    let catalogue = entity_geometry::catalogue(source);
    let is_point = |id: &SketchEntityId| source.points().iter().any(|point| point.id == *id);
    let mut points: BTreeSet<_> = selected
        .iter()
        .flat_map(|id| catalogue[id].ownership_dependencies())
        .collect();
    points.extend(selected.iter().copied().filter(is_point));
    let mut sources: Vec<_> = source
        .owned_entity_ids()
        .into_iter()
        .filter(|id| points.contains(id))
        .collect();
    sources.extend(selected.iter().copied().filter(|id| !is_point(id)));
    sources
}
impl PreparedSelectionCopy {
    fn append_reserved(
        &self,
        target: &mut Sketch2D,
        reserve: &mut impl FnMut() -> Result<SketchEntityId, RecipeIdError>,
    ) -> Result<SketchCopyReport, SketchCopyError> {
        let identities = CopyIdentities::reserve(target, &self.sources, reserve)?;
        self.append(target, &identities)
            .map_err(SketchCopyError::Sketch)?;
        self.report(&identities).map_err(SketchCopyError::Sketch)
    }
    fn new(
        source: &Sketch2D,
        entities: &[SketchEntityId],
        operation: SketchCopyOperation,
    ) -> Result<Self, SketchError> {
        Self::transformed(source, entities, operation.resolve(source)?)
    }
    fn transformed(
        source: &Sketch2D,
        entities: &[SketchEntityId],
        transform: super::SketchSelectionTransform,
    ) -> Result<Self, SketchError> {
        let selected = copy_selection(source, entities)?;
        let mut prepared = source.clone();
        let mut sources = prepared.transform_entities(&selected, transform)?;
        sources.extend(
            selected
                .iter()
                .copied()
                .filter(|id| !source.points().iter().any(|point| point.id == *id)),
        );
        Ok(Self {
            source: prepared,
            selected,
            sources,
        })
    }
}
fn copy_selection(
    source: &Sketch2D,
    entities: &[SketchEntityId],
) -> Result<Vec<SketchEntityId>, SketchError> {
    let catalogue = entity_geometry::catalogue(source);
    let mut seen = BTreeSet::new();
    let mut selected = Vec::new();
    for id in entities {
        match catalogue.get(id) {
            None | Some(OwnedSketchGeometry::OffsetSpline(_)) => {
                return Err(copy_selection_error(*id));
            }
            _ => {}
        }
        if seen.insert(*id) {
            selected.push(*id);
        }
    }
    Ok(selected)
}
fn copy_selection_error(id: SketchEntityId) -> SketchError {
    SketchError::Definition(format!(
        "copy entity {id:?}: expected an owned point, line, circle, arc, ellipse, or parent spline; an offset spline cannot be copied independently"
    ))
}
