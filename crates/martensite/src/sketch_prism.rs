// SPDX-License-Identifier: GPL-2.0-only
mod boundary;
use crate::{
    EvaluationEntity as Entity, EvaluationFailure, EvaluationFailureKind as Kind,
    EvaluationPhase as Phase, PrismProbeDefinition, PrismProbeSnapshot,
};
pub use oblikovati_model::feature::sketch_prism::{SketchDirectedCurve, SketchPrismSelection};
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
    evaluate_resolved_prism(resolved).map_err(|cause| cause.to_string())
}
/// Preserve source context through the sole native construction path; example: `evaluate_resolved_prism(resolved)?`.
pub(crate) fn evaluate_resolved_prism(
    resolved: oblikovati_model::feature::sketch_prism::ResolvedSketchPrism<'_>,
) -> Result<PrismProbeSnapshot, EvaluationFailure> {
    let sketch_id = resolved.sketch.sketch.id();
    let native = selected_definition(
        resolved.sketch.sketch.geometry(),
        resolved.selection,
        resolved.displacement,
    )
    .map_err(|error| error.for_sketch(sketch_id, format!("document sketch {:?}", sketch_id)))?;
    crate::prism::build_prism(&native, resolved.sketch.plane).map_err(|error| {
        error.for_sketch(sketch_id, format!("document sketch {:?} prism", sketch_id))
    })
}

fn selected_definition(
    geometry: &oblikovati_model::sketch::Sketch2D,
    selection: &SketchPrismSelection,
    displacement: [f64; 3],
) -> Result<PrismProbeDefinition, EvaluationFailure> {
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
/// Completed session receipts retain the kernel's owner-thread restriction.
/// ```compile_fail
/// use oblikovati_martensite::PrismProbeSnapshot;
/// use oblikovati_session::part_evaluation::CompletedPartEvaluation;
/// fn require_send<T: Send>() {}
/// require_send::<CompletedPartEvaluation<PrismProbeSnapshot, oblikovati_martensite::EvaluationFailure>>();
/// ```
#[derive(Default)]
pub struct MartensitePrismEvaluator;
impl oblikovati_model::feature::part::SketchPrismEvaluator for MartensitePrismEvaluator {
    type Body = PrismProbeSnapshot;
    type Failure = EvaluationFailure;
    fn verify_body_reference(&self, _body: &Self::Body, key: &[u8]) -> Result<(), String> {
        Err(format!(
            "diagnostic prism reference {key:?}: expected a document-scoped NamedPrismEvaluator for body-set edits"
        ))
    }
    fn evaluate_prism(
        &mut self,
        feature: oblikovati_model::feature::FeatureId,
        input: oblikovati_model::feature::sketch_prism::ResolvedSketchPrism<'_>,
    ) -> Result<Self::Body, Self::Failure> {
        evaluate_resolved_prism(input).map_err(|cause| cause.for_feature(feature))
    }
}
