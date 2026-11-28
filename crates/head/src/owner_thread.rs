// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, cylindrical/conical face picking, cosmetic preview, and commit for Part Modify Thread (#3918).
#[path = "owner_thread_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedThread, build_thread_definition, prepare_thread_definition, replace_thread,
};
use oblikovati_gui::{
    GuiThreadAction, GuiThreadEditSeed, GuiThreadPick, GuiThreadPreview, GuiThreadPreviewState,
    GuiThreadRequest,
};
use oblikovati_martensite::{EvaluationFailure, NamedSolid, ThreadableFaceInfo};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureEvaluation, thread::ThreadDefinition},
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::DocumentGuid,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum ThreadCommit {
    Create(Box<PreparedThread>),
    Edit(FeatureId, Box<ThreadDefinition>),
}

enum ThreadOutcome {
    Ready(ThreadCommit),
    Incomplete,
}

pub(in crate::owner) struct ThreadPreview {
    request: GuiThreadRequest,
    result: Result<ThreadOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct ThreadOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) pick: Option<GuiThreadPick>,
    pub(in crate::owner) preview: Option<ThreadPreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
    evaluation: Option<&PartFeatureEvaluation<NamedSolid, EvaluationFailure>>,
    history: &DocumentEditHistory,
    guid: DocumentGuid,
) -> BTreeMap<FeatureId, GuiThreadEditSeed> {
    seed::edit_seeds(snapshot, evaluation, history, guid)
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn part_has_threadable_faces(&self, document: u64) -> bool {
        let workspace = self.files.workspace();
        let Ok(session) = workspace.session(document) else {
            return false;
        };
        let Ok(identity) = workspace.identity(document) else {
            return false;
        };
        let Ok(evaluation) = self.part_scenes.current(session) else {
            return false;
        };
        evaluation
            .ordered_bodies()
            .any(|body| body.has_threadable_faces(session.history(), identity.internal_name))
    }

    pub(in crate::owner) fn gui_thread_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiThreadPreviewState> {
        let preview = self.create_features.thread.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiThreadPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(ThreadOutcome::Ready(_)) => GuiThreadPreview::Ready,
                    Ok(ThreadOutcome::Incomplete) => GuiThreadPreview::Incomplete,
                    Err(cause) => GuiThreadPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_thread_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiThreadPick> {
        self.create_features
            .thread
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn pick_thread_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.thread.active {
            return Ok(None);
        }
        let editing = self
            .create_features
            .thread
            .preview
            .as_ref()
            .and_then(|preview| preview.request.feature)
            .is_some();
        if editing {
            return Ok(None);
        }
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let allowed = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::Face,
        ]);
        let candidates = self
            .part_scenes
            .part_candidates(&self.files, document, ray, &allowed)?;
        let Ok(Some(picked)) =
            candidates.resolve_hit(&allowed, &self.ambient_selection, tolerance.unwrap_or(0.0))
        else {
            return Ok(None);
        };
        let crate::part_scene::pick_candidates::PickTarget::Face(_, key) = picked.target else {
            return Ok(None);
        };
        let Some((body, info)) = self.resolve_thread_face(document, &key)? else {
            return Ok(Some(false));
        };
        self.create_features.thread.pick = Some(GuiThreadPick {
            document,
            generation,
            body,
            face: key,
            diameter_mm: info.diameter_cm * 10.0,
            full_depth_mm: info.full_depth_cm * 10.0,
            conical: info.conical,
            internal: info.internal,
        });
        Ok(Some(true))
    }

    pub(in crate::owner) fn thread_command(
        &mut self,
        action: GuiThreadAction,
    ) -> Result<bool, String> {
        match action {
            GuiThreadAction::Open | GuiThreadAction::OpenEdit(_) => {
                self.create_features.thread.active = true;
                Ok(false)
            }
            GuiThreadAction::Clear => {
                let had_state = self.create_features.thread.active
                    | self.create_features.thread.preview.take().is_some()
                    | self.create_features.thread.pick.take().is_some();
                self.create_features.thread.active = false;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiThreadAction::Preview(request) => {
                let request = *request;
                self.require_thread_generation(&request)?;
                self.create_features.thread.active = true;
                let result = self.preview_thread(&request);
                self.create_features.thread.preview = Some(ThreadPreview { request, result });
                Ok(true)
            }
            GuiThreadAction::Commit(request) => {
                let request = *request;
                self.require_thread_generation(&request)?;
                let commit = self
                    .create_features
                    .thread
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        ThreadOutcome::Ready(commit) => Some(commit),
                        ThreadOutcome::Incomplete => None,
                    })
                    .ok_or("thread: expected a current certified preview")?
                    .clone();
                self.commit_thread(request.document, commit)?;
                self.create_features.thread.preview = None;
                self.create_features.thread.pick = None;
                self.create_features.thread.active = false;
                self.model_face = None;
                self.face_appearance_picks.clear();
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_thread_generation(&self, request: &GuiThreadRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("thread: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_thread(&mut self, request: &GuiThreadRequest) -> Result<ThreadOutcome, String> {
        let Some(face) = request.face.clone() else {
            self.clear_extrusion_preview();
            return Ok(ThreadOutcome::Incomplete);
        };
        let body = match request.body.clone() {
            Some(body) => body,
            None => self
                .resolve_thread_face(request.document, &face)?
                .map(|(body, _)| body)
                .ok_or_else(|| "thread: expected a cylindrical or conical face".to_owned())?,
        };
        let inputs = seed::request_to_inputs(request, body, face);
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_thread_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                ThreadCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let edited = build_thread_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                ThreadCommit::Edit(id, Box::new(edited))
            }
        };
        match self.evaluate_thread_preview_scene(request, &commit) {
            Ok(scene) => {
                if request.preview_hidden {
                    self.clear_extrusion_preview();
                } else {
                    self.extrusion_preview = Some((
                        request.document,
                        request.generation,
                        scene,
                        super::super::owner_extrusion_preview::PreviewKind::Replacement,
                    ));
                    self.part_scenes.invalidate_presentation();
                }
                Ok(ThreadOutcome::Ready(commit))
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn evaluate_thread_preview_scene(
        &mut self,
        request: &GuiThreadRequest,
        commit: &ThreadCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            ThreadCommit::Create(prepared) => {
                let cloned = (**prepared).clone();
                let projection = session
                    .preview_projected(
                        |candidate, ids| cloned.create(candidate, ids),
                        |candidate, id| {
                            Ok(self.part_scenes.project_feature_edit(
                                &self.files,
                                request.document,
                                candidate,
                                id,
                            ))
                        },
                    )
                    .map_err(|cause| cause.to_string())?;
                projection.map_err(|cause| cause.to_string())
            }
            ThreadCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_thread(candidate, id, edited.clone()),
                        |candidate, ()| {
                            Ok(self.part_scenes.project_feature_edit(
                                &self.files,
                                request.document,
                                candidate,
                                id,
                            ))
                        },
                    )
                    .map_err(|cause| cause.to_string())?;
                projection.map_err(|cause| cause.to_string())
            }
        }
    }

    fn resolve_thread_face(
        &self,
        document: u64,
        face: &[u8],
    ) -> Result<Option<(Vec<u8>, ThreadableFaceInfo)>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        for body in evaluation.ordered_bodies() {
            let Ok(keys) = body.face_reference_keys(session.history(), guid) else {
                continue;
            };
            if keys.iter().any(|k| k.as_slice() == face) {
                let Some(info) = body.threadable_face_info(session.history(), guid, face)? else {
                    return Ok(None);
                };
                let body_key = body.body_reference_key(session.history(), guid)?.to_vec();
                return Ok(Some((body_key, info)));
            }
        }
        Ok(None)
    }

    fn commit_thread(&mut self, document: u64, commit: ThreadCommit) -> Result<(), String> {
        match commit {
            ThreadCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Thread",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            ThreadCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Thread",
                    |snapshot, _| replace_thread(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
