// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, multi-face picking, face highlighting, preview, and commits for Part Modify Delete Face (#3918).
use super::*;
use oblikovati_automation::{
    DeleteFaceFeatureInputs, PreparedDeleteFace, build_delete_face_definition,
    prepare_delete_face_definition, replace_delete_face,
};
use oblikovati_gui::{
    GuiDeleteFaceAction, GuiDeleteFaceEditSeed, GuiDeleteFacePick, GuiDeleteFacePreview,
    GuiDeleteFacePreviewState, GuiDeleteFaceRequest,
};
use oblikovati_model::{
    feature::{FeatureId, delete_face::DeleteFaceDefinition, part::PartFeatureDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum DeleteFaceCommit {
    Create(Box<PreparedDeleteFace>),
    Edit(FeatureId, Box<DeleteFaceDefinition>),
}

enum DeleteFaceOutcome {
    Ready(DeleteFaceCommit),
    Incomplete,
}

pub(in crate::owner) struct DeleteFacePreview {
    request: GuiDeleteFaceRequest,
    result: Result<DeleteFaceOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct DeleteFaceOwnerState {
    pub(in crate::owner) active: bool,
    pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiDeleteFacePick>,
    pub(in crate::owner) preview: Option<DeleteFacePreview>,
    pub(in crate::owner) highlighted_faces: Vec<(FeatureId, Vec<u8>)>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiDeleteFaceEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::DeleteFace(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            Some((
                feature.id(),
                GuiDeleteFaceEditSeed {
                    body: recipe.body,
                    faces: recipe.faces,
                    heal_remaining_faces: recipe.heal_remaining_faces,
                    select_lump_or_void: recipe.select_lump_or_void,
                },
            ))
        })
        .collect()
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
    pub(in crate::owner) fn gui_delete_face_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiDeleteFacePreviewState> {
        let preview = self.create_features.delete_face.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiDeleteFacePreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(DeleteFaceOutcome::Ready(_)) => GuiDeleteFacePreview::Ready,
                    Ok(DeleteFaceOutcome::Incomplete) => GuiDeleteFacePreview::Incomplete,
                    Err(cause) => GuiDeleteFacePreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_delete_face_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiDeleteFacePick> {
        self.create_features
            .delete_face
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn pick_delete_face_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.delete_face.active {
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
        let crate::part_scene::pick_candidates::PickTarget::Face(feature, key) = picked.target
        else {
            return Ok(None);
        };
        let (_, body) = self.resolve_delete_face_owner(document, &key)?;
        let state = &mut self.create_features.delete_face;
        state.pick_seq = state.pick_seq.saturating_add(1);
        state.pick = Some(GuiDeleteFacePick {
            document,
            generation,
            sequence: state.pick_seq,
            body,
            face: key.clone(),
        });
        if state
            .highlighted_faces
            .first()
            .is_some_and(|(existing, _)| *existing != feature)
        {
            state.highlighted_faces.clear();
        }
        if !state
            .highlighted_faces
            .iter()
            .any(|(f, k)| *f == feature && *k == key)
        {
            state.highlighted_faces.push((feature, key));
        }
        self.part_scenes.invalidate_presentation();
        Ok(Some(true))
    }

    pub(in crate::owner) fn delete_face_command(
        &mut self,
        action: GuiDeleteFaceAction,
    ) -> Result<bool, String> {
        match action {
            GuiDeleteFaceAction::Open | GuiDeleteFaceAction::OpenEdit(_) => {
                self.create_features.delete_face.active = true;
                self.create_features.delete_face.highlighted_faces.clear();
                self.part_scenes.invalidate_presentation();
                Ok(false)
            }
            GuiDeleteFaceAction::Clear => {
                let state = &mut self.create_features.delete_face;
                let had_state = state.active
                    | state.preview.take().is_some()
                    | state.pick.take().is_some()
                    | !state.highlighted_faces.is_empty();
                state.active = false;
                state.highlighted_faces.clear();
                if had_state {
                    self.part_scenes.invalidate_presentation();
                }
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiDeleteFaceAction::Preview(request) => {
                let request = *request;
                self.require_delete_face_generation(&request)?;
                self.create_features.delete_face.active = true;
                self.sync_delete_face_highlights(request.document, &request.faces)?;
                let result = self.preview_delete_face(&request);
                self.create_features.delete_face.preview =
                    Some(DeleteFacePreview { request, result });
                Ok(true)
            }
            GuiDeleteFaceAction::Commit(request) => {
                let request = *request;
                self.require_delete_face_generation(&request)?;
                let commit = self
                    .create_features
                    .delete_face
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        DeleteFaceOutcome::Ready(commit) => Some(commit),
                        DeleteFaceOutcome::Incomplete => None,
                    })
                    .ok_or("delete face: expected a current certified preview")?
                    .clone();
                self.commit_delete_face(request.document, commit)?;
                self.create_features.delete_face.preview = None;
                self.create_features.delete_face.pick = None;
                self.create_features.delete_face.highlighted_faces.clear();
                self.create_features.delete_face.active = false;
                self.model_face = None;
                self.face_appearance_picks.clear();
                self.clear_extrusion_preview();
                self.part_scenes.invalidate_presentation();
                Ok(true)
            }
        }
    }

    fn require_delete_face_generation(&self, request: &GuiDeleteFaceRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("delete face: expected the current document generation".into());
        }
        Ok(())
    }

    fn sync_delete_face_highlights(
        &mut self,
        document: u64,
        faces: &[Vec<u8>],
    ) -> Result<(), String> {
        let mut next = Vec::with_capacity(faces.len());
        for face in faces {
            if let Ok((feature, _)) = self.resolve_delete_face_owner(document, face) {
                next.push((feature, face.clone()));
            }
        }
        if self.create_features.delete_face.highlighted_faces != next {
            self.create_features.delete_face.highlighted_faces = next;
            self.part_scenes.invalidate_presentation();
        }
        Ok(())
    }

    fn preview_delete_face(
        &mut self,
        request: &GuiDeleteFaceRequest,
    ) -> Result<DeleteFaceOutcome, String> {
        self.clear_extrusion_preview();
        let Some(first_face) = request.faces.first() else {
            return Ok(DeleteFaceOutcome::Incomplete);
        };
        let body = match request.body.clone() {
            Some(body) => body,
            None => {
                self.resolve_delete_face_owner(request.document, first_face)?
                    .1
            }
        };
        let inputs = DeleteFaceFeatureInputs {
            body,
            faces: request.faces.clone(),
            heal_remaining_faces: request.heal_remaining_faces,
            select_lump_or_void: request.select_lump_or_void,
        };
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_delete_face_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                DeleteFaceCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited =
                    build_delete_face_definition(&inputs).map_err(|cause| cause.to_string())?;
                DeleteFaceCommit::Edit(id, Box::new(edited))
            }
        };
        // Validate the real Martensite candidate so unhealable faces fail authentically on OK,
        // while keeping the source faces visible and highlighted in the viewport until commit.
        self.validate_delete_face_candidate(request.document, &commit)?;
        Ok(DeleteFaceOutcome::Ready(commit))
    }

    fn validate_delete_face_candidate(
        &self,
        document: u64,
        commit: &DeleteFaceCommit,
    ) -> Result<(), String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            DeleteFaceCommit::Create(prepared) => {
                let cloned = (**prepared).clone();
                let projection = session
                    .preview_projected(
                        |candidate, ids| cloned.create(candidate, ids),
                        |candidate, id| {
                            Ok(self.part_scenes.project_feature_edit(
                                &self.files,
                                document,
                                candidate,
                                id,
                            ))
                        },
                    )
                    .map_err(|cause| cause.to_string())?;
                projection.map(|_| ()).map_err(|cause| cause.to_string())
            }
            DeleteFaceCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_delete_face(candidate, id, edited.clone()),
                        |candidate, ()| {
                            Ok(self.part_scenes.project_feature_edit(
                                &self.files,
                                document,
                                candidate,
                                id,
                            ))
                        },
                    )
                    .map_err(|cause| cause.to_string())?;
                projection.map(|_| ()).map_err(|cause| cause.to_string())
            }
        }
    }

    fn resolve_delete_face_owner(
        &self,
        document: u64,
        face: &[u8],
    ) -> Result<(FeatureId, Vec<u8>), String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        for (feature, body) in evaluation.ordered_body_outputs() {
            let Ok(keys) = body.face_reference_keys(session.history(), guid) else {
                continue;
            };
            if keys.iter().any(|k| k.as_slice() == face) {
                return Ok((
                    feature,
                    body.body_reference_key(session.history(), guid)?.to_vec(),
                ));
            }
        }
        Err("delete face: expected a current body face".into())
    }

    fn commit_delete_face(
        &mut self,
        document: u64,
        commit: DeleteFaceCommit,
    ) -> Result<(), String> {
        match commit {
            DeleteFaceCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Delete Face",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            DeleteFaceCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Delete Face",
                    |snapshot, _| replace_delete_face(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
