// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, Existing/New face picking, highlighting, preview, and commits for Part Surface Replace Face (#3918).
use super::*;
use oblikovati_automation::{
    PreparedReplaceFace, ReplaceFaceFeatureInputs, build_replace_face_definition,
    prepare_replace_face_definition, replace_replace_face,
};
use oblikovati_gui::{
    GuiReplaceFaceAction, GuiReplaceFaceEditSeed, GuiReplaceFacePick, GuiReplaceFacePickMode,
    GuiReplaceFacePreview, GuiReplaceFacePreviewState, GuiReplaceFaceRequest,
};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition, replace_face::ReplaceFaceDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

type ResolvedFacePick = (FeatureId, Vec<u8>, bool, Vec<Vec<u8>>);

#[derive(Clone)]
enum ReplaceFaceCommit {
    Create(Box<PreparedReplaceFace>),
    Edit(FeatureId, Box<ReplaceFaceDefinition>),
}

enum ReplaceFaceOutcome {
    Ready(ReplaceFaceCommit),
    Incomplete,
}

pub(in crate::owner) struct ReplaceFacePreview {
    request: GuiReplaceFaceRequest,
    result: Result<ReplaceFaceOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct ReplaceFaceOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) pick_mode: GuiReplaceFacePickMode,
    pub(in crate::owner) automatic_face_chain: bool,
    pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiReplaceFacePick>,
    pub(in crate::owner) preview: Option<ReplaceFacePreview>,
    pub(in crate::owner) highlighted_faces: Vec<(FeatureId, Vec<u8>)>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiReplaceFaceEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::ReplaceFace(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            Some((
                feature.id(),
                GuiReplaceFaceEditSeed {
                    body: recipe.body,
                    existing_faces: recipe.existing_faces,
                    new_faces: recipe.new_faces,
                    automatic_face_chain: recipe.automatic_face_chain,
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
    pub(in crate::owner) fn gui_replace_face_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiReplaceFacePreviewState> {
        let preview = self.create_features.replace_face.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiReplaceFacePreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(ReplaceFaceOutcome::Ready(_)) => GuiReplaceFacePreview::Ready,
                    Ok(ReplaceFaceOutcome::Incomplete) => GuiReplaceFacePreview::Incomplete,
                    Err(cause) => GuiReplaceFacePreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_replace_face_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiReplaceFacePick> {
        self.create_features
            .replace_face
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn pick_replace_face_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.replace_face.active {
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
        let mut candidates =
            self.part_scenes
                .part_candidates(&self.files, document, ray, &allowed)?;
        if self.create_features.replace_face.pick_mode == GuiReplaceFacePickMode::ExistingFaces {
            let session = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?;
            let eval = self.part_scenes.current(session)?;
            let solid_features: std::collections::BTreeSet<_> = eval
                .ordered_body_outputs()
                .filter(|(_, body)| !body.is_surface())
                .map(|(feat, _)| feat)
                .collect();
            candidates.retain_targets(|target| {
                matches!(
                    target,
                    crate::part_scene::pick_candidates::PickTarget::Face(feat, _)
                        if solid_features.contains(feat)
                )
            });
        }
        let Ok(Some(picked)) =
            candidates.resolve_hit(&allowed, &self.ambient_selection, tolerance.unwrap_or(0.0))
        else {
            return Ok(None);
        };
        let crate::part_scene::pick_candidates::PickTarget::Face(feature, key) = picked.target
        else {
            return Ok(None);
        };
        let chain = self.create_features.replace_face.automatic_face_chain
            && self.create_features.replace_face.pick_mode == GuiReplaceFacePickMode::ExistingFaces;
        let (_, body, is_surface, faces) = self.resolve_replace_face_pick(document, &key, chain)?;
        let state = &mut self.create_features.replace_face;
        state.pick_seq = state.pick_seq.saturating_add(1);
        state.pick = Some(GuiReplaceFacePick {
            document,
            generation,
            sequence: state.pick_seq,
            body,
            faces: faces.clone(),
            is_surface,
        });
        for face in faces {
            if !state
                .highlighted_faces
                .iter()
                .any(|(f, k)| *f == feature && *k == face)
            {
                state.highlighted_faces.push((feature, face));
            }
        }
        self.part_scenes.invalidate_presentation();
        Ok(Some(true))
    }

    pub(in crate::owner) fn replace_face_command(
        &mut self,
        action: GuiReplaceFaceAction,
    ) -> Result<bool, String> {
        match action {
            GuiReplaceFaceAction::Open | GuiReplaceFaceAction::OpenEdit(_) => {
                self.create_features.replace_face.active = true;
                self.create_features.replace_face.highlighted_faces.clear();
                self.part_scenes.invalidate_presentation();
                Ok(false)
            }
            GuiReplaceFaceAction::SetPickMode(mode) => {
                self.create_features.replace_face.pick_mode = mode;
                Ok(false)
            }
            GuiReplaceFaceAction::SetAutomaticFaceChain(enabled) => {
                self.create_features.replace_face.automatic_face_chain = enabled;
                Ok(false)
            }
            GuiReplaceFaceAction::Clear => {
                let state = &mut self.create_features.replace_face;
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
            GuiReplaceFaceAction::Preview(request) => {
                let request = *request;
                self.require_replace_face_generation(&request)?;
                self.create_features.replace_face.active = true;
                self.create_features.replace_face.automatic_face_chain =
                    request.automatic_face_chain;
                self.sync_replace_face_highlights(&request)?;
                let result = self.preview_replace_face(&request);
                self.create_features.replace_face.preview =
                    Some(ReplaceFacePreview { request, result });
                Ok(true)
            }
            GuiReplaceFaceAction::Commit(request) => {
                let request = *request;
                self.require_replace_face_generation(&request)?;
                let commit = self
                    .create_features
                    .replace_face
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        ReplaceFaceOutcome::Ready(commit) => Some(commit),
                        ReplaceFaceOutcome::Incomplete => None,
                    })
                    .ok_or("replace face: expected a current certified preview")?
                    .clone();
                self.commit_replace_face(request.document, commit)?;
                let state = &mut self.create_features.replace_face;
                state.preview = None;
                state.pick = None;
                state.highlighted_faces.clear();
                state.active = false;
                self.model_face = None;
                self.face_appearance_picks.clear();
                self.clear_extrusion_preview();
                self.part_scenes.invalidate_presentation();
                Ok(true)
            }
        }
    }

    fn require_replace_face_generation(
        &self,
        request: &GuiReplaceFaceRequest,
    ) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("replace face: expected the current document generation".into());
        }
        Ok(())
    }

    fn sync_replace_face_highlights(
        &mut self,
        request: &GuiReplaceFaceRequest,
    ) -> Result<(), String> {
        let mut next = Vec::with_capacity(request.existing_faces.len() + request.new_faces.len());
        for face in &request.existing_faces {
            if let Ok((feature, _, _, _)) =
                self.resolve_replace_face_pick(request.document, face, false)
            {
                next.push((feature, face.clone()));
            }
        }
        for target in &request.new_faces {
            if let Ok((feature, _, _, _)) =
                self.resolve_replace_face_pick(request.document, &target.face, false)
            {
                next.push((feature, target.face.clone()));
            }
        }
        if self.create_features.replace_face.highlighted_faces != next {
            self.create_features.replace_face.highlighted_faces = next;
            self.part_scenes.invalidate_presentation();
        }
        Ok(())
    }

    fn preview_replace_face(
        &mut self,
        request: &GuiReplaceFaceRequest,
    ) -> Result<ReplaceFaceOutcome, String> {
        self.clear_extrusion_preview();
        let Some(first_existing) = request.existing_faces.first() else {
            return Ok(ReplaceFaceOutcome::Incomplete);
        };
        if request.new_faces.is_empty() {
            return Ok(ReplaceFaceOutcome::Incomplete);
        }
        let body = match request.body.clone() {
            Some(body) => body,
            None => {
                self.resolve_replace_face_pick(request.document, first_existing, false)?
                    .1
            }
        };
        let inputs = ReplaceFaceFeatureInputs {
            body,
            existing_faces: request.existing_faces.clone(),
            new_faces: request.new_faces.clone(),
            automatic_face_chain: request.automatic_face_chain,
        };
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_replace_face_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                ReplaceFaceCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited =
                    build_replace_face_definition(&inputs).map_err(|cause| cause.to_string())?;
                ReplaceFaceCommit::Edit(id, Box::new(edited))
            }
        };
        let scene = self.evaluate_replace_face_preview_scene(request.document, &commit)?;
        if !request.preview_hidden {
            self.extrusion_preview = Some((
                request.document,
                request.generation,
                scene,
                super::super::owner_extrusion_preview::PreviewKind::Replacement,
            ));
            self.part_scenes.invalidate_presentation();
        }
        Ok(ReplaceFaceOutcome::Ready(commit))
    }

    fn evaluate_replace_face_preview_scene(
        &self,
        document: u64,
        commit: &ReplaceFaceCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            ReplaceFaceCommit::Create(prepared) => {
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
                projection.map_err(|cause| cause.to_string())
            }
            ReplaceFaceCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_replace_face(candidate, id, edited.clone()),
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
                projection.map_err(|cause| cause.to_string())
            }
        }
    }

    fn resolve_replace_face_pick(
        &self,
        document: u64,
        face: &[u8],
        automatic_face_chain: bool,
    ) -> Result<ResolvedFacePick, String> {
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
            if !keys.iter().any(|k| k.as_slice() == face) {
                continue;
            }
            let body_key = body.body_reference_key(session.history(), guid)?.to_vec();
            let is_surface = body.is_surface();
            let faces = if automatic_face_chain && !is_surface {
                body.tangent_face_chain(session.history(), guid, face)?
            } else {
                vec![face.to_vec()]
            };
            return Ok((feature, body_key, is_surface, faces));
        }
        Err("replace face: expected a current body face".into())
    }

    fn commit_replace_face(
        &mut self,
        document: u64,
        commit: ReplaceFaceCommit,
    ) -> Result<(), String> {
        match commit {
            ReplaceFaceCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Replace Face",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            ReplaceFaceCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Replace Face",
                    |snapshot, _| replace_replace_face(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
