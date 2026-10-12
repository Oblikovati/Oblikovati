// SPDX-License-Identifier: GPL-2.0-only
use crate::{MartensitePrismEvaluator, PrismProbeSnapshot};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureEvaluation},
    history::DocumentSnapshot,
};
/// Consume active stored prism inputs through the common ordered replay.
/// Example: `retain_stored_sketch_prism(&snapshot, feature_id)?`; the full extrude contract remains open.
pub fn retain_stored_sketch_prism(
    snapshot: &DocumentSnapshot,
    feature_id: FeatureId,
) -> Result<PrismProbeSnapshot, String> {
    let evaluation = snapshot
        .evaluate_part_features(&mut MartensitePrismEvaluator)
        .map_err(|cause| format!("part feature {feature_id:?}: {cause}"))?;
    if evaluation.body(feature_id).is_none() {
        return Err(stored_body_decline(&evaluation, feature_id));
    }
    evaluation.into_body(feature_id).ok_or_else(|| {
        format!("part feature {feature_id:?}: expected transfer of an available isolated body",)
    })
}
fn stored_body_decline(
    evaluation: &PartFeatureEvaluation<PrismProbeSnapshot, String>,
    feature: FeatureId,
) -> String {
    let status = evaluation
        .health()
        .iter()
        .find(|record| record.feature == feature)
        .map(|record| &record.status);
    format!(
        "part feature {feature:?} replay health {status:?}: expected stored sketch prism inputs with a successful active result"
    )
}
