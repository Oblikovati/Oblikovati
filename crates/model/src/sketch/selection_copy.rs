// SPDX-License-Identifier: GPL-2.0-only
mod adopt;
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
        let identities = CopyIdentities::reserve(self, &plan.sources, reserve)?;
        let mut candidate = self.clone();
        plan.append(&mut candidate, &identities)
            .map_err(SketchCopyError::Sketch)?;
        let report = plan.report(&identities).map_err(SketchCopyError::Sketch)?;
        *self = candidate;
        Ok(report)
    }
}
impl PreparedSelectionCopy {
    fn new(
        source: &Sketch2D,
        entities: &[SketchEntityId],
        operation: SketchCopyOperation,
    ) -> Result<Self, SketchError> {
        let selected = copy_selection(source, entities)?;
        let mut prepared = source.clone();
        let mut sources = prepared.transform_entities(&selected, operation.resolve(source)?)?;
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
