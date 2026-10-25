// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::{NamedPrismEvaluator, NamedPrismSnapshot, PrismMeshOptions};
use oblikovati_model::feature::{
    FeatureEvaluationStatus, FeatureHealthRecord, part::PartFeatureFailure,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{
    TransactionObserver, files::FileWorkspace, part_evaluation::PartEvaluationSlot,
    workspace::WorkspaceObserver,
};
use oblikovati_viewport::{PartPresentationDecline, ViewportScene};

/// Go Dark's `#1e2d50` viewport background, converted from sRGB to renderer-linear RGB.
const GO_DARK_VIEWPORT_BACKGROUND: [f64; 3] = [0.0132, 0.0262, 0.0803];
/// Go's neutral default material keeps Part solids legible without feature-specific appearances.
const GO_DEFAULT_SURFACE_COLOR: [f32; 4] = [0.7, 0.72, 0.75, 1.];
/// Owned display state names its document and generation; kernel handles stay in result storage.
#[derive(Clone, Debug)]
pub struct HeadSceneStatus {
    pub source: Option<(u64, u64)>,
    pub visible_bodies: usize,
    pub health:
        Vec<FeatureHealthRecord<PartFeatureFailure<oblikovati_martensite::EvaluationFailure>>>,
    pub presentation_declines: Vec<PartPresentationDecline>,
}
impl HeadSceneStatus {
    /// Show current solid and failure counts; example: `status.title()` for the native window.
    pub fn title(&self) -> String {
        let Some((document, generation)) = self.source else {
            return "Oblikovati".into();
        };
        let unavailable = self.unavailable_count();
        format!(
            "Oblikovati - document {document} generation {generation} - {} solids, {unavailable} unavailable features",
            self.visible_bodies
        )
    }
    fn unavailable_count(&self) -> usize {
        self.health
            .iter()
            .filter(|record| {
                matches!(
                    record.status,
                    FeatureEvaluationStatus::Failed(_)
                        | FeatureEvaluationStatus::Blocked { .. }
                        | FeatureEvaluationStatus::ConditionFailed(_)
                )
            })
            .count()
            + self.presentation_declines.len()
    }
}
/// One complete replacement, including empty scenes after close or suppression.
pub struct HeadSceneUpdate {
    pub scene: ViewportScene,
    pub status: HeadSceneStatus,
}
#[derive(Default)]
pub(crate) struct DesktopPartScenes {
    source: Option<(u64, u64, u64)>,
    slot: PartEvaluationSlot<NamedPrismSnapshot, oblikovati_martensite::EvaluationFailure>,
}
impl DesktopPartScenes {
    pub fn refresh<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
    ) -> Result<Option<HeadSceneUpdate>, String> {
        let source = active_scene_source(files)?;
        if source == self.source {
            return Ok(None);
        }
        let update = match source {
            Some((id, _, _)) => self.evaluate(files, id)?,
            None => self.empty(),
        };
        self.source = source;
        Ok(Some(update))
    }
    fn evaluate<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<HeadSceneUpdate, String> {
        self.ensure_current(files, document)?;
        self.present_current(files, document)
    }
    fn ensure_current<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<(), String> {
        if self
            .slot
            .latest()
            .is_some_and(|view| view.document != document)
        {
            self.slot.clear();
        }
        let session = files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if self.slot.current(session).is_ok() {
            return Ok(());
        }
        self.adopt_current(files, document)
    }
    fn adopt_current<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<(), String> {
        let completed = complete_desktop_replay(files, document)?;
        files
            .commands(document)
            .map_err(|cause| cause.to_string())?
            .adopt_part_evaluation(completed, &mut self.slot)
            .map_err(|cause| cause.to_string())?;
        Ok(())
    }
    /// Borrow only this session's current published bodies; example: `scenes.current(session)?`.
    pub(crate) fn current<'a, O: TransactionObserver>(
        &'a self,
        session: &'a oblikovati_session::DocumentSession<O>,
    ) -> Result<
        &'a oblikovati_model::feature::part::PartFeatureEvaluation<
            NamedPrismSnapshot,
            oblikovati_martensite::EvaluationFailure,
        >,
        String,
    > {
        self.slot
            .current(session)
            .map_err(|cause| cause.to_string())
    }
    fn present_current<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<HeadSceneUpdate, String> {
        let session = files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let evaluated = self.current(session)?;
        let guid = files
            .workspace()
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        render_part_scene(evaluated, (document, session.generation()), |body| {
            let key = body.body_reference_key(session.history(), guid)?;
            Ok(session.body_visible(&bodies::body_reference_text(key)))
        })
    }
    fn empty(&mut self) -> HeadSceneUpdate {
        self.slot.clear();
        HeadSceneUpdate {
            scene: ViewportScene {
                background: GO_DARK_VIEWPORT_BACKGROUND,
                triangles: Vec::new(),
            },
            status: HeadSceneStatus {
                source: None,
                visible_bodies: 0,
                health: Vec::new(),
                presentation_declines: Vec::new(),
            },
        }
    }
    pub fn clear(&mut self) {
        self.slot.clear();
        self.source = None;
    }
}

#[cfg(test)]
mod tests {
    use super::{DesktopPartScenes, GO_DARK_VIEWPORT_BACKGROUND};

    #[test]
    fn empty_scene_uses_the_go_dark_viewport_background() {
        let mut scenes = DesktopPartScenes::default();
        assert_eq!(scenes.empty().scene.background, GO_DARK_VIEWPORT_BACKGROUND);
    }
}

fn complete_desktop_replay<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
) -> Result<
    oblikovati_session::part_evaluation::CompletedPartEvaluation<
        NamedPrismSnapshot,
        oblikovati_martensite::EvaluationFailure,
    >,
    String,
> {
    let workspace = files.workspace();
    let guid = workspace
        .identity(document)
        .map_err(|cause| cause.to_string())?
        .internal_name;
    let session = workspace
        .session(document)
        .map_err(|cause| cause.to_string())?;
    session
        .prepare_part_evaluation()
        .evaluate(&mut NamedPrismEvaluator::new(session.history(), guid))
        .map_err(|cause| cause.to_string())
}

fn active_scene_source<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<Option<(u64, u64, u64)>, String> {
    files
        .workspace()
        .active_document()
        .map(|id| {
            files
                .workspace()
                .session(id)
                .map(|session| (id, session.generation(), session.body_visibility_revision()))
        })
        .transpose()
        .map_err(|cause| cause.to_string())
}

fn render_part_scene(
    evaluated: &oblikovati_model::feature::part::PartFeatureEvaluation<
        NamedPrismSnapshot,
        oblikovati_martensite::EvaluationFailure,
    >,
    source: (u64, u64),
    visible: impl FnMut(&NamedPrismSnapshot) -> Result<bool, String>,
) -> Result<HeadSceneUpdate, String> {
    let presentation = ViewportScene::from_part_evaluation_with_visibility(
        evaluated,
        PrismMeshOptions {
            chord_error: 0.05,
            angular_error: 0.5,
        },
        GO_DARK_VIEWPORT_BACKGROUND,
        GO_DEFAULT_SURFACE_COLOR,
        visible,
    )?;
    Ok(HeadSceneUpdate {
        scene: presentation.scene,
        status: HeadSceneStatus {
            source: Some(source),
            visible_bodies: presentation.visible_bodies,
            health: evaluated.health().to_vec(),
            presentation_declines: presentation.declines,
        },
    })
}

mod references;
mod topology;

mod bodies;

mod physical;

mod health;
