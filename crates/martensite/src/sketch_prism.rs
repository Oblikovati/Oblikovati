// SPDX-License-Identifier: GPL-2.0-only
mod boundary;
use crate::{PrismProbeDefinition, PrismProbeSnapshot, retain_prism_on_plane};
pub use oblikovati_model::feature::sketch_prism::{SketchDirectedLine, SketchPrismSelection};
use oblikovati_model::feature::{sketch_prism::SketchPrismDefinition, work_geometry::DatumLength};
use oblikovati_model::{document_sketch::DocumentSketchId, history::DocumentSnapshot};
/// Consume current fully constrained geometry and placement before retaining a diagnostic prism.
/// Example: `retain_selected_sketch_prism(&snapshot, sketch_id, &selection, [0., 0., 2.])?`.
pub fn retain_selected_sketch_prism(
    snapshot: &DocumentSnapshot,
    sketch_id: DocumentSketchId,
    selection: &SketchPrismSelection,
    displacement: [f64; 3],
) -> Result<PrismProbeSnapshot, String> {
    let definition = SketchPrismDefinition::new(
        sketch_id,
        selection.clone(),
        displacement.map(DatumLength::Constant),
    )
    .map_err(|error| error.to_string())?;
    retain_sketch_prism_definition(snapshot, &definition)
}
/// Resolve authored scalar sources through the current document before kernel construction.
/// Example: `retain_sketch_prism_definition(&snapshot, &definition)?`.
pub fn retain_sketch_prism_definition(
    snapshot: &DocumentSnapshot,
    definition: &SketchPrismDefinition,
) -> Result<PrismProbeSnapshot, String> {
    let resolved = definition
        .resolve(snapshot)
        .map_err(|error| error.to_string())?;
    retain_resolved_sketch_prism(resolved)
}
/// Build a diagnostic result from the model's checked replay inputs.
/// Example: `retain_resolved_sketch_prism(input)?` does not read or replay the document again.
pub fn retain_resolved_sketch_prism(
    resolved: oblikovati_model::feature::sketch_prism::ResolvedSketchPrism<'_>,
) -> Result<PrismProbeSnapshot, String> {
    let sketch_id = resolved.sketch.sketch.id();
    let native = selected_definition(
        resolved.sketch.sketch.geometry(),
        resolved.selection,
        resolved.displacement,
    )
    .map_err(|error| format!("document sketch {:?}: {error}", sketch_id))?;
    retain_prism_on_plane(&native, resolved.sketch.plane)
        .map_err(|error| format!("document sketch {:?} prism: {error}", sketch_id))
}

fn selected_definition(
    geometry: &oblikovati_model::sketch::Sketch2D,
    selection: &SketchPrismSelection,
    displacement: [f64; 3],
) -> Result<PrismProbeDefinition, String> {
    let outer = boundary::selected_loop(geometry, &selection.outer)?;
    let holes = selection
        .holes
        .iter()
        .map(|lines| boundary::selected_loop(geometry, lines))
        .collect::<Result<Vec<_>, _>>()?;
    Ok(PrismProbeDefinition {
        outer,
        holes,
        displacement,
    })
}

/// Released kernel implementation of the model's selected-prism replay interface.
#[derive(Default)]
pub struct MartensitePrismEvaluator;
impl oblikovati_model::feature::part::SketchPrismEvaluator for MartensitePrismEvaluator {
    type Body = PrismProbeSnapshot;
    type Failure = String;
    fn evaluate_prism(
        &mut self,
        input: oblikovati_model::feature::sketch_prism::ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, Self::Failure> {
        retain_resolved_sketch_prism(input)
    }
}
