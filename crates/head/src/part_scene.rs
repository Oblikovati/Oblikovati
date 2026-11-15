// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_martensite::{NamedPrismSnapshot, PrismMeshOptions};
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
use assembly::AssemblyViewMode;
mod assembly_pick;
pub(crate) mod extrusion;
pub(crate) mod pick_candidates;
pub(crate) mod resolution_policy;
mod selection_bounds;
mod update;
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
    producer_digest: Option<Result<String, String>>,
    source: Option<assembly::SceneSource>,
    scene_failure: Option<(u64, u64, String)>,
    presentation_invalidated: bool,
    occurrence_selection: Option<oblikovati_gui::GuiOccurrencePath>,
    resolution: Option<oblikovati_martensite::PrismResolutionPolicy>,
    slot: PartEvaluationSlot<NamedPrismSnapshot, oblikovati_martensite::EvaluationFailure>,
}
impl DesktopPartScenes {
    pub(crate) fn assembly_producer_digest(&mut self) -> Result<String, String> {
        self.producer_digest
            .get_or_insert_with(crate::assembly_producer::current_executable_digest)
            .clone()
    }

    pub(crate) fn sketch_continuity_resolver<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<oblikovati_automation::PartSketchContinuity, String> {
        let workspace = files.workspace();
        let guid = workspace
            .identity(document)
            .map_err(|error| error.to_string())?
            .internal_name;
        let options = workspace
            .session(document)
            .map_err(|error| error.to_string())?
            .sketch_solve_options();
        Ok(oblikovati_automation::PartSketchContinuity::new(
            guid,
            self.resolution,
            options,
        ))
    }
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
        self.present_current(files, document, AssemblyViewMode::Live)
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
        oblikovati_automation::refresh_part_sketches(files, document, self.resolution)?;
        let completed =
            oblikovati_automation::complete_part_replay(files, document, self.resolution)?;
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
        mode: AssemblyViewMode,
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
            if mode == AssemblyViewMode::Live && !session.body_visible(&key) {
                return Ok(None);
            }
            if mode == AssemblyViewMode::Live
                && session.selected_browser_body().map(|(chosen, _)| chosen) == Some(key.as_str())
            {
                return Ok(Some(CHOSEN_BODY_COLOR));
            }
            Ok(Some(body_color(files, session.snapshot(), &key)?))
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
        let candidates = self.part_candidates(files, document, line)?;
        let filter = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::Face,
        ]);
        let ambient = oblikovati_session::selection::AmbientSelectionFilter::default();
        match candidates.resolve(&filter, &ambient, 0.)? {
            Some(pick_candidates::PickTarget::Face(feature, role)) => Ok(Some((feature, role))),
            None => Ok(None),
            Some(other) => Err(format!("part hit {other:?}: expected a face")),
        }
    }
    pub(crate) fn part_candidates<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        line: [[f64; 3]; 2],
    ) -> Result<pick_candidates::PickCandidates, String> {
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
        let mut candidates =
            pick_candidates::PickCandidates::new(pick_candidates::PickCoverage::NearestFacePerBody);
        for record in evaluated.health() {
            let Some(body) = evaluated.body(record.feature) else {
                continue;
            };
            let key =
                bodies::body_reference_text(body.body_reference_key(session.history(), guid)?);
            if !session.body_visible(&key) {
                continue;
            }
            if let Some((role, reach)) = body.face_on_line(line)? {
                candidates.push(
                    pick_candidates::PickTarget::Face(record.feature, role),
                    reach,
                )?;
                candidates.push(
                    pick_candidates::PickTarget::Body(record.feature, key),
                    reach,
                )?;
            }
        }
        Ok(candidates)
    }
    pub(crate) fn invalidate_presentation(&mut self) {
        self.presentation_invalidated = true;
    }
    pub(crate) fn set_occurrence_selection(
        &mut self,
        selected: Option<oblikovati_gui::GuiOccurrencePath>,
    ) {
        if self.occurrence_selection != selected {
            self.occurrence_selection = selected;
            self.invalidate_presentation();
        }
    }
    pub fn clear(&mut self) {
        self.slot.clear();
        self.source = None;
        self.scene_failure = None;
    }
}

/// How finely a body is meshed for display, in working units and radians.
pub(crate) const DISPLAY_MESH: PrismMeshOptions = oblikovati_drawing::DISPLAY_MESH;
/// A body whose browser row is chosen: scene colours are linear, so this draws as the accent
/// the browser's rows use.
const CHOSEN_BODY_COLOR: [f32; 4] = [0.05, 0.32, 1., 1.];
/// Resolve the body's effective appearance through the document, project, and catalog.
fn body_color<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    snapshot: &oblikovati_model::history::DocumentSnapshot,
    key: &str,
) -> Result<[f32; 4], String> {
    use oblikovati_model::assets::{AssignmentReference, ProjectAssetLibrary};
    let reference = AssignmentReference::new(key, vec![])?;
    let empty_project = ProjectAssetLibrary::default();
    let project = files
        .active_project()
        .map_or(&empty_project, |project| &project.assets);
    resolved_body_color(
        &snapshot.assets,
        project,
        files.builtin_catalog()?,
        &reference,
    )
}
fn resolved_body_color(
    assets: &oblikovati_model::assets::DocumentAssets,
    project: &oblikovati_model::assets::ProjectAssetLibrary,
    catalog: &oblikovati_model::assets::BuiltinAssetCatalog,
    reference: &oblikovati_model::assets::AssignmentReference,
) -> Result<[f32; 4], String> {
    use oblikovati_api::contract::Appearance;
    use oblikovati_model::assets::ScopedAssets;
    let appearances = ScopedAssets {
        document: assets.appearances(),
        project: project.appearances(),
        builtin: catalog.appearances(),
    };
    let materials = ScopedAssets {
        document: assets.materials(),
        project: project.materials(),
        builtin: catalog.materials(),
    };
    let color = assets
        .assignments
        .resolve_appearance(&appearances, &materials, reference, None)?
        .base()
        .color;
    Ok([color.r, color.g, color.b, 1.])
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

mod health;
