// SPDX-License-Identifier: GPL-2.0-only
use crate::PrismProbeSnapshot;
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureEvaluation},
    history::DocumentSnapshot,
};
/// Retain an unnamed display snapshot from a document-scoped complete replay.
/// Example: `retain_stored_sketch_prism(&snapshot, document, feature_id)?`.
/// Completed receipts retain the kernel's owner-thread restriction.
/// ```compile_fail
/// use oblikovati_martensite::PrismProbeSnapshot;
/// use oblikovati_session::part_evaluation::CompletedPartEvaluation;
/// fn require_send<T: Send>() {}
/// require_send::<CompletedPartEvaluation<PrismProbeSnapshot, oblikovati_martensite::EvaluationFailure>>();
/// ```
pub fn retain_stored_sketch_prism(
    snapshot: &DocumentSnapshot,
    document: oblikovati_model::identity::DocumentGuid,
    feature_id: FeatureId,
) -> Result<PrismProbeSnapshot, String> {
    let replay = oblikovati_model::history::SnapshotReplay::new(snapshot)
        .map_err(|error| error.to_string())?;
    let prism = crate::retain_named_sketch_prism(replay.history(), document, feature_id)?;
    Ok(prism.into_probe_snapshot())
}
pub(crate) fn stored_body_decline<Body, Failure: std::fmt::Debug>(
    evaluation: &PartFeatureEvaluation<Body, Failure>,
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
