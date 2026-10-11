// SPDX-License-Identifier: GPL-2.0-only
use super::sketch_constraints::require_sketch_namespace;
use crate::{
    document_sketch::{DocumentSketchId, DocumentSketches},
    identity::RecipeIdError,
    parameter::ParameterCollection,
};
use std::collections::BTreeMap;
mod transition;
/// Permanent per-sketch next ordinals; names start at d0 and never rewind with recipe state.
#[derive(Debug, Default)]
pub struct SketchDimensionNameReservations {
    next_ordinals: BTreeMap<u64, u64>,
}
impl SketchDimensionNameReservations {
    /// Restore saved names, or seed an older document from surviving owned dimension parameters.
    /// Example: `SketchDimensionNameReservations::restore(&sketches, &parameters, sketch_mark, None)?`.
    pub fn restore(
        sketches: &DocumentSketches,
        parameters: &ParameterCollection<u64>,
        sketch_mark: u64,
        saved: Option<&BTreeMap<u64, u64>>,
    ) -> Result<Self, RecipeIdError> {
        let mut restored = Self {
            next_ordinals: saved.cloned().unwrap_or_default(),
        };
        for &sketch in restored.next_ordinals.keys() {
            require_sketch_namespace(sketch, sketch_mark)?;
        }
        restored.restore_live_names(sketches, parameters, sketch_mark, saved.is_some())?;
        Ok(restored)
    }
    fn restore_live_names(
        &mut self,
        sketches: &DocumentSketches,
        parameters: &ParameterCollection<u64>,
        sketch_mark: u64,
        explicit: bool,
    ) -> Result<(), RecipeIdError> {
        for placed in sketches.sketches() {
            let sketch = placed.id().0;
            require_sketch_namespace(sketch, sketch_mark)?;
            self.restore_owner_names(placed.geometry(), parameters, sketch, explicit)?;
        }
        Ok(())
    }
    fn restore_owner_names(
        &mut self,
        geometry: &crate::sketch::Sketch2D,
        parameters: &ParameterCollection<u64>,
        sketch: u64,
        explicit: bool,
    ) -> Result<(), RecipeIdError> {
        let names = geometry
            .dimensional_constraints()
            .filter_map(|(_, parameter)| {
                parameters
                    .get(parameter)
                    .filter(|state| !state.renamed)
                    .map(|state| state.name.as_str())
            });
        for name in names {
            self.restore_live_name(sketch, name, explicit)?;
        }
        Ok(())
    }
    fn restore_live_name(
        &mut self,
        sketch: u64,
        name: &str,
        explicit: bool,
    ) -> Result<(), RecipeIdError> {
        let Some(ordinal) = canonical_dimension_ordinal(name) else {
            return Ok(());
        };
        let minimum = ordinal.checked_add(1).ok_or(RecipeIdError::Exhausted)?;
        let mark = self.next_ordinals.get(&sketch).copied().unwrap_or(0);
        if explicit && mark < minimum {
            return Err(RecipeIdError::DimensionNameMark {
                sketch,
                mark,
                minimum,
            });
        }
        self.next_ordinals.insert(sketch, mark.max(minimum));
        Ok(())
    }
    /// Reserve the next globally unused name, consuming colliding ordinals too (#3787).
    /// Example: `names.allocate(sketch, sketch_mark, &parameters)?` before parsing the new source.
    pub fn allocate(
        &mut self,
        sketch: DocumentSketchId,
        sketch_mark: u64,
        parameters: &ParameterCollection<u64>,
    ) -> Result<String, RecipeIdError> {
        require_sketch_namespace(sketch.0, sketch_mark)?;
        let next = self.next_ordinals.entry(sketch.0).or_default();
        loop {
            let name = format!("d{next}");
            *next = next.checked_add(1).ok_or(RecipeIdError::Exhausted)?;
            if parameters.find(&name).is_none() {
                return Ok(name);
            }
        }
    }
    /// Capture sorted nonzero next ordinals, retaining retired sketch namespaces.
    /// Example: `let marks = names.marks();` when saving a document copy.
    pub fn marks(&self) -> BTreeMap<u64, u64> {
        self.next_ordinals
            .iter()
            .filter_map(|(&sketch, &next)| (next != 0).then_some((sketch, next)))
            .collect()
    }
}
fn canonical_dimension_ordinal(name: &str) -> Option<u64> {
    let suffix = name.strip_prefix('d')?;
    let ordinal = suffix.parse::<u64>().ok()?;
    (name == format!("d{ordinal}")).then_some(ordinal)
}
