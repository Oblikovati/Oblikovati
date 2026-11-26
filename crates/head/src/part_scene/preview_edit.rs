// SPDX-License-Identifier: GPL-2.0-only
//! Private feature-edit scenes with generation-checked checkpoint reuse.
use super::*;
use oblikovati_model::{feature::FeatureId, history::DocumentSnapshot, identity::DocumentGuid};
use oblikovati_session::DocumentSession;
type PreviewEvaluation = oblikovati_model::feature::part::PartFeatureEvaluation<
    NamedSolid,
    oblikovati_martensite::EvaluationFailure,
>;

pub(crate) enum FeatureEditPreviewError {
    UnchangedCut(String),
    Failed(String),
}

impl From<String> for FeatureEditPreviewError {
    fn from(cause: String) -> Self {
        Self::Failed(cause)
    }
}

impl std::fmt::Display for FeatureEditPreviewError {
    fn fmt(&self, output: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::UnchangedCut(cause) | Self::Failed(cause) => output.write_str(cause),
        }
    }
}
impl DesktopPartScenes {
    #[cfg(test)]
    pub(crate) fn checkpoint_preview_metrics(&self) -> (u64, usize) {
        (self.replay.replay_count(), self.checkpoint_previews.get())
    }
    #[cfg(test)]
    pub(crate) fn extrusion_prefix_previews(&self) -> usize {
        self.extrusion_prefix_previews.get()
    }
    #[cfg(test)]
    pub(crate) fn note_extrusion_prefix_preview(&self) {
        self.extrusion_prefix_previews
            .set(self.extrusion_prefix_previews.get() + 1);
    }
    pub(crate) fn ensure_edit_checkpoints<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<(), String> {
        self.replay.ensure_checkpoints(files, document)
    }
    pub(crate) fn part_scene_without_feature_body<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        feature: FeatureId,
    ) -> Result<ViewportScene, String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = files
            .workspace()
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluated = self.current(session)?;
        let hidden = evaluated
            .body(feature)
            .map(|body| body.body_reference_key(session.history(), guid))
            .transpose()?;
        let update = render_part_scene(evaluated, (document, session.generation()), |body| {
            let reference = body.body_reference_key(session.history(), guid)?;
            if hidden.as_ref() == Some(&reference) {
                return Ok(None);
            }
            let key = bodies::body_reference_text(reference);
            if !session.body_visible(&key) {
                return Ok(None);
            }
            Ok(Some(body_color(files, session.snapshot(), &key)?))
        })?;
        if !update.status.presentation_declines.is_empty() {
            return Err(format!(
                "incomplete feature preview: {:?}",
                update.status.presentation_declines
            ));
        }
        Ok(update.scene)
    }
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
        self.project_feature_edit(files, document, candidate, feature)
            .map_err(|cause| cause.to_string())
    }

    pub(crate) fn project_feature_edit<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        candidate: &DocumentSnapshot,
        feature: FeatureId,
    ) -> Result<ViewportScene, FeatureEditPreviewError> {
        let policy = self
            .resolution
            .ok_or_else(|| "feature preview: expected a certified resolution".to_owned())?;
        let session = files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = files
            .workspace()
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let preview = files
            .workspace()
            .preview_owned_part(document, Some(policy), candidate)
            .map_err(|error| error.to_string())?;
        let evaluated = preview.evaluation;
        let used_checkpoint = preview.used_checkpoint;
        let edited = evaluated
            .body(feature)
            .ok_or_else(|| preview_failure(&evaluated, feature))?;
        let edited_key =
            bodies::body_reference_text(edited.body_reference_key(session.history(), guid)?);
        #[cfg(test)]
        if used_checkpoint {
            self.checkpoint_previews
                .set(self.checkpoint_previews.get() + 1);
        }
        #[cfg(not(test))]
        let _ = used_checkpoint;
        let preview_color = match candidate
            .work_geometry
            .get(feature)
            .map_err(|cause| cause.to_string())?
            .definition()
        {
            oblikovati_model::feature::part::PartFeatureDefinition::SketchSweep(_)
            | oblikovati_model::feature::part::PartFeatureDefinition::SketchLoft(_) => {
                NEW_SOLID_PREVIEW
            }
            _ => [1., 0.65, 0.1, 1.],
        };
        render_preview(
            files,
            session,
            candidate,
            &evaluated,
            guid,
            (&edited_key, preview_color),
            document,
        )
        .map_err(Into::into)
    }
}
fn preview_failure(evaluated: &PreviewEvaluation, feature: FeatureId) -> FeatureEditPreviewError {
    use oblikovati_model::feature::{FeatureEvaluationStatus, part::PartFeatureFailure};
    let records: Vec<_> = evaluated
        .health()
        .iter()
        .filter(|record| record.feature == feature)
        .collect();
    if let [record] = records.as_slice()
        && let FeatureEvaluationStatus::Failed(PartFeatureFailure::Kernel(cause)) = &record.status
    {
        if cause.kind() == oblikovati_martensite::EvaluationFailureKind::UnchangedCut {
            return FeatureEditPreviewError::UnchangedCut(cause.to_string());
        }
        return FeatureEditPreviewError::Failed(format!(
            "{cause}; feature {feature:?} preview: expected a valid body; health {:?}",
            evaluated.health()
        ));
    }
    FeatureEditPreviewError::Failed(format!(
        "feature {feature:?} preview: expected a valid body; health {:?}",
        evaluated.health()
    ))
}
fn render_preview<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    session: &DocumentSession<O>,
    candidate: &DocumentSnapshot,
    evaluated: &PreviewEvaluation,
    guid: DocumentGuid,
    (edited_key, preview_color): (&str, [f32; 4]),
    document: u64,
) -> Result<ViewportScene, String> {
    let update = render_part_scene(evaluated, (document, session.generation()), |body| {
        let key = bodies::body_reference_text(body.body_reference_key(session.history(), guid)?);
        if !session.body_visible(&key) {
            return Ok(None);
        }
        if key == edited_key {
            return Ok(Some(preview_color));
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
