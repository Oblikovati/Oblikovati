// SPDX-License-Identifier: GPL-2.0-only
mod boundary;
use crate::{PrismProbeDefinition, PrismProbeSnapshot, retain_prism_on_plane};
use oblikovati_model::{
    document_sketch::DocumentSketchId, history::DocumentSnapshot, sketch::SketchEntityId,
};
use std::collections::BTreeSet;
/// One authored line in a directed boundary; reversal changes traversal, not its source identity.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct SketchDirectedLine {
    pub line: SketchEntityId,
    pub reversed: bool,
}
/// Explicit ordered boundaries over shared authored endpoints; this is not automatic region detection.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct SketchPrismSelection {
    pub outer: Vec<SketchDirectedLine>,
    pub holes: Vec<Vec<SketchDirectedLine>>,
}
/// Consume current fully constrained geometry and placement before retaining a diagnostic prism.
/// Example: `retain_selected_sketch_prism(&snapshot, sketch_id, &selection, [0., 0., 2.])?`.
pub fn retain_selected_sketch_prism(
    snapshot: &DocumentSnapshot,
    sketch_id: DocumentSketchId,
    selection: &SketchPrismSelection,
    displacement: [f64; 3],
) -> Result<PrismProbeSnapshot, String> {
    let resolved = snapshot
        .require_resolved_sketch(sketch_id)
        .map_err(|error| error.to_string())?;
    let definition = selected_definition(resolved.sketch.geometry(), selection, displacement)
        .map_err(|error| format!("document sketch {sketch_id:?}: {error}"))?;
    retain_prism_on_plane(&definition, resolved.plane)
        .map_err(|error| format!("document sketch {sketch_id:?} prism: {error}"))
}
fn selected_definition(
    geometry: &oblikovati_model::sketch::Sketch2D,
    selection: &SketchPrismSelection,
    displacement: [f64; 3],
) -> Result<PrismProbeDefinition, String> {
    let mut used = BTreeSet::new();
    let outer = boundary::selected_loop(geometry, &selection.outer, &mut used)?;
    let holes = selection
        .holes
        .iter()
        .map(|lines| boundary::selected_loop(geometry, lines, &mut used))
        .collect::<Result<Vec<_>, _>>()?;
    Ok(PrismProbeDefinition {
        outer,
        holes,
        displacement,
    })
}
