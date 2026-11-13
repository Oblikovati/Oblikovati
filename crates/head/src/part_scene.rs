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
mod assembly;
pub(crate) mod extrusion;
pub(crate) mod resolution_policy;
/// Owned display state names its document and generation; kernel handles stay in result storage.
#[derive(Clone, Debug)]
pub struct HeadSceneStatus {
    pub source: Option<(u64, u64)>,
    pub visible_bodies: usize,
    pub health:
        Vec<FeatureHealthRecord<PartFeatureFailure<oblikovati_martensite::EvaluationFailure>>>,
    pub presentation_declines: Vec<PartPresentationDecline>,
    pub scene_errors: Vec<String>,
}
impl HeadSceneStatus {
    /// Show current solid and failure counts; example: `status.title()` for the native window.
    pub fn title(&self) -> String {
        let Some((document, generation)) = self.source else {
            return "Oblikovati".into();
        };
        let unavailable = self.unavailable_count();
        let title = format!(
            "Oblikovati - document {document} generation {generation} - {} solids, {unavailable} unavailable features",
            self.visible_bodies
        );
        self.scene_errors
            .first()
            .map_or(title.clone(), |error| format!("{title} - {error}"))
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
            + self.scene_errors.len()
    }
}
/// One complete replacement, including empty scenes after close or suppression.
pub struct HeadSceneUpdate {
    pub scene: ViewportScene,
    pub status: HeadSceneStatus,
}
#[derive(Default)]
pub(crate) struct DesktopPartScenes {
    source: Option<assembly::SceneSource>,
    scene_failure: Option<(u64, u64, String)>,
    presentation_invalidated: bool,
    resolution: Option<oblikovati_martensite::PrismResolutionPolicy>,
    slot: PartEvaluationSlot<NamedPrismSnapshot, oblikovati_martensite::EvaluationFailure>,
}
impl DesktopPartScenes {
    pub(crate) fn use_certified_resolution(
        &mut self,
        resolution: oblikovati_martensite::PrismResolutionPolicy,
    ) {
        self.resolution = Some(resolution);
        self.clear();
    }
    pub fn refresh<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
    ) -> Result<Option<HeadSceneUpdate>, String> {
        let source = match assembly::active_scene_source(files) {
            Ok(source) => source,
            Err(error) => return self.failed_assembly_scene(files, error),
        };
        if source == self.source && self.scene_failure.is_none() && !self.presentation_invalidated {
            return Ok(None);
        }
        let update = match &source {
            Some(source) if source.is_assembly => {
                match self.evaluate_assembly(files, source.document) {
                    Ok(scene) => scene,
                    Err(error) => return self.failed_assembly_scene(files, error),
                }
            }
            Some(source) => self.evaluate(files, source.document)?,
            None => self.empty(),
        };
        self.source = source;
        self.scene_failure = None;
        self.presentation_invalidated = false;
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
        let completed = complete_desktop_replay(files, document, self.resolution)?;
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
            let key =
                bodies::body_reference_text(body.body_reference_key(session.history(), guid)?);
            if !session.body_visible(&key) {
                return Ok(None);
            }
            if session.selected_browser_body().map(|(chosen, _)| chosen) == Some(key.as_str()) {
                return Ok(Some(CHOSEN_BODY_COLOR));
            }
            Ok(Some(body_color(files, session.snapshot(), &key)))
        })
    }
    fn empty(&mut self) -> HeadSceneUpdate {
        self.slot.clear();
        HeadSceneUpdate {
            scene: ViewportScene {
                background: [0., 0., 1.],
                triangles: Vec::new(),
            },
            status: HeadSceneStatus {
                source: None,
                visible_bodies: 0,
                health: Vec::new(),
                presentation_declines: Vec::new(),
                scene_errors: Vec::new(),
            },
        }
    }
    /// The outline of a feature's solid, face by face, or of one of its faces; empty for a
    /// feature with no body. Example: `scenes.feature_edges(session, feature, None)?`.
    pub(crate) fn feature_edges<O: TransactionObserver>(
        &self,
        session: &oblikovati_session::DocumentSession<O>,
        feature: oblikovati_model::feature::FeatureId,
        only: Option<oblikovati_martensite::PrismFaceRole>,
    ) -> Result<Vec<[[f64; 3]; 2]>, String> {
        let evaluation = self.current(session)?;
        let Some(body) = evaluation.body(feature) else {
            return Ok(Vec::new());
        };
        let mesh =
            oblikovati_martensite::PrismPresentationSource::presentation(body, DISPLAY_MESH)?;
        Ok(mesh
            .faces
            .iter()
            .filter(|face| only.is_none_or(|role| face.role == role))
            .flat_map(|face| face.boundary_segments())
            .collect())
    }
    /// The nearest face on a line of sight, over every solid of the part.
    /// Example: `scenes.face_on_line(files, document, window.view_line(pixel)?)?`.
    pub(crate) fn face_on_line<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<
        Option<(
            oblikovati_model::feature::FeatureId,
            oblikovati_martensite::PrismFaceRole,
        )>,
        String,
    > {
        let session = files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let evaluated = self.current(session)?;
        let mut nearest = None;
        for record in evaluated.health() {
            let Some(body) = evaluated.body(record.feature) else {
                continue;
            };
            if let Some((role, reach)) = body.face_on_line(line)?
                && nearest.is_none_or(|(_, _, best)| reach < best)
            {
                nearest = Some((record.feature, role, reach));
            }
        }
        Ok(nearest.map(|(feature, role, _)| (feature, role)))
    }
    pub(crate) fn invalidate_presentation(&mut self) {
        self.presentation_invalidated = true;
    }
    pub fn clear(&mut self) {
        self.slot.clear();
        self.source = None;
        self.scene_failure = None;
    }
}

fn complete_desktop_replay<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    resolution: Option<oblikovati_martensite::PrismResolutionPolicy>,
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
    let request = session.prepare_part_evaluation();
    match resolution {
        Some(resolution) => request.evaluate(
            &mut oblikovati_martensite::CertifiedNamedPrismEvaluator::with_policy(
                session.history(),
                guid,
                resolution,
            ),
        ),
        None => request.evaluate(&mut NamedPrismEvaluator::new(session.history(), guid)),
    }
    .map_err(|cause| cause.to_string())
}

/// How finely a body is meshed for display, in working units and radians.
pub(crate) const DISPLAY_MESH: PrismMeshOptions = oblikovati_drawing::DISPLAY_MESH;
/// The colour a body shows without an appearance of its own or of its part.
const PART_COLOR: [f32; 4] = [0., 1., 0., 1.];
/// A body whose browser row is chosen: scene colours are linear, so this draws as the accent
/// the browser's rows use.
const CHOSEN_BODY_COLOR: [f32; 4] = [0.05, 0.32, 1., 1.];
/// A body's own appearance, or its part's, by its base colour; an unknown id reads as no choice.
fn body_color<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    key: &str,
) -> [f32; 4] {
    use oblikovati_api::contract::Appearance;
    let reference = oblikovati_model::assets::AssignmentReference::new(key, vec![]);
    let id = reference.ok().and_then(|reference| {
        snapshot
            .assets
            .assignments
            .effective_appearance_id(&reference)
            .map(str::to_owned)
    });
    let Some(id) = id else {
        return PART_COLOR;
    };
    let appearance = snapshot
        .assets
        .appearances()
        .get(&id)
        .or_else(|| files.active_project()?.assets.appearances().get(&id))
        .or_else(|| files.builtin_catalog().ok()?.appearances().get(&id));
    appearance.map_or(PART_COLOR, |asset| {
        let color = asset.base().color;
        [color.r, color.g, color.b, 1.]
    })
}
pub(crate) fn render_part_scene(
    evaluated: &oblikovati_model::feature::part::PartFeatureEvaluation<
        NamedPrismSnapshot,
        oblikovati_martensite::EvaluationFailure,
    >,
    source: (u64, u64),
    style: impl FnMut(&NamedPrismSnapshot) -> Result<Option<[f32; 4]>, String>,
) -> Result<HeadSceneUpdate, String> {
    let presentation =
        ViewportScene::from_part_evaluation_styled(evaluated, DISPLAY_MESH, [0., 0., 1.], style)?;
    Ok(HeadSceneUpdate {
        scene: presentation.scene,
        status: HeadSceneStatus {
            source: Some(source),
            visible_bodies: presentation.visible_bodies,
            health: evaluated.health().to_vec(),
            presentation_declines: presentation.declines,
            scene_errors: Vec::new(),
        },
    })
}

mod references;

mod bodies;

mod measure;
mod physical;

mod topology;

#[cfg(test)]
#[path = "../tests/part_scene_certified.rs"]
mod certified_tests;

#[cfg(test)]
#[path = "../tests/part_scene_assembly.rs"]
mod assembly_tests;
