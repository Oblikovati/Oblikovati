// SPDX-License-Identifier: GPL-2.0-only
//! Private, fully replayed feature-edit scene.
use super::*;
use oblikovati_model::{feature::FeatureId, history::DocumentSnapshot, identity::DocumentGuid};
use oblikovati_session::DocumentSession;
type PreviewEvaluation = oblikovati_model::feature::part::PartFeatureEvaluation<
    NamedPrismSnapshot,
    oblikovati_martensite::EvaluationFailure,
>;
impl DesktopPartScenes {
    pub(crate) fn preview_feature_edit<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<ViewportScene, String> {
        let policy = self
            .resolution
            .ok_or("feature preview: expected a certified resolution")?;
        let session = files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = files
            .workspace()
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let (evaluated, edited_key) = evaluate_preview(session, candidate, feature, guid, policy)?;
        render_preview(
            files,
            session,
            candidate,
            &evaluated,
            guid,
            &edited_key,
            document,
        )
    }
}
fn evaluate_preview<O: TransactionObserver>(
    session: &DocumentSession<O>,
    candidate: &DocumentSnapshot,
    feature: FeatureId,
    guid: DocumentGuid,
    policy: oblikovati_martensite::PrismResolutionPolicy,
) -> Result<(PreviewEvaluation, String), String> {
    let mut evaluator = oblikovati_martensite::CertifiedNamedPrismEvaluator::with_policy(
        session.history(),
        guid,
        policy,
    );
    let evaluated = candidate
        .evaluate_part_features(&mut evaluator)
        .map_err(|cause| cause.to_string())?;
    let edited = evaluated.body(feature).ok_or_else(|| {
        format!(
            "feature {feature:?} preview: expected a valid body; health {:?}",
            evaluated.health()
        )
    })?;
    let key = bodies::body_reference_text(edited.body_reference_key(session.history(), guid)?);
    Ok((evaluated, key))
}
fn render_preview<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    session: &DocumentSession<O>,
    candidate: &DocumentSnapshot,
    evaluated: &PreviewEvaluation,
    guid: DocumentGuid,
    edited_key: &str,
    document: u64,
) -> Result<ViewportScene, String> {
    let update = render_part_scene(evaluated, (document, session.generation()), |body| {
        let key = bodies::body_reference_text(body.body_reference_key(session.history(), guid)?);
        if !session.body_visible(&key) {
            return Ok(None);
        }
        if key == edited_key {
            return Ok(Some([1., 0.65, 0.1, 1.]));
        }
        Ok(Some(body_color(files, candidate, &key)?))
    })?;
    if !update.status.presentation_declines.is_empty() {
        return Err(format!(
            "feature preview: expected complete presentation: {:?}",
            update.status.presentation_declines
        ));
    }
    Ok(update.scene)
}
