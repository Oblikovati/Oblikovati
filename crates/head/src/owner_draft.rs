// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, planar face picking, live replacement preview, and commits for Part Modify Draft (#3918).
#[path = "owner_draft_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedDraft, build_draft_definition, prepare_draft_definition, replace_draft,
};
use oblikovati_gui::{
    GuiDraftAction, GuiDraftEditSeed, GuiDraftPick, GuiDraftPreview, GuiDraftPreviewState,
    GuiDraftRequest,
};
use oblikovati_model::{
    feature::{
        FeatureId,
        draft::{DraftDefinition, DraftMode},
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum DraftCommit {
    Create(Box<PreparedDraft>),
    Edit(FeatureId, Box<DraftDefinition>),
}

enum DraftOutcome {
    Ready(DraftCommit),
    Incomplete,
}

pub(in crate::owner) struct DraftPreview {
    request: GuiDraftRequest,
    result: Result<DraftOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct DraftOwnerState {
    pub(in crate::owner) active: bool,
    pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiDraftPick>,
    pub(in crate::owner) preview: Option<DraftPreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiDraftEditSeed> {
    seed::edit_seeds(snapshot)
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
    pub(in crate::owner) fn gui_draft_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiDraftPreviewState> {
        let preview = self.create_features.draft.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiDraftPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(DraftOutcome::Ready(_)) => GuiDraftPreview::Ready,
                    Ok(DraftOutcome::Incomplete) => GuiDraftPreview::Incomplete,
                    Err(cause) => GuiDraftPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_draft_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiDraftPick> {
        self.create_features
            .draft
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn pick_draft_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.draft.active {
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
        let body = self.resolve_draft_body_for_face(document, &key)?;
        self.create_features.draft.pick_seq = self.create_features.draft.pick_seq.saturating_add(1);
        self.create_features.draft.pick = Some(GuiDraftPick {
            document,
            generation,
            sequence: self.create_features.draft.pick_seq,
            body,
            face: key,
        });
        Ok(Some(true))
    }

    pub(in crate::owner) fn draft_command(
        &mut self,
        action: GuiDraftAction,
    ) -> Result<bool, String> {
        match action {
            GuiDraftAction::Open | GuiDraftAction::OpenEdit(_) => {
                self.create_features.draft.active = true;
                Ok(false)
            }
            GuiDraftAction::Clear => {
                let had_state = self.create_features.draft.active
                    | self.create_features.draft.preview.take().is_some()
                    | self.create_features.draft.pick.take().is_some();
                self.create_features.draft.active = false;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiDraftAction::Preview(request) => {
                let request = *request;
                self.require_draft_generation(&request)?;
                self.create_features.draft.active = true;
                let result = self.preview_draft(&request);
                self.create_features.draft.preview = Some(DraftPreview { request, result });
                Ok(true)
            }
            GuiDraftAction::Commit(request) => {
                let request = *request;
                self.require_draft_generation(&request)?;
                let commit = self
                    .create_features
                    .draft
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        DraftOutcome::Ready(commit) => Some(commit),
                        DraftOutcome::Incomplete => None,
                    })
                    .ok_or("draft: expected a current certified preview")?
                    .clone();
                self.commit_draft(request.document, commit)?;
                self.create_features.draft.preview = None;
                self.create_features.draft.pick = None;
                self.create_features.draft.active = false;
                self.model_face = None;
                self.face_appearance_picks.clear();
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_draft_generation(&self, request: &GuiDraftRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("draft: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_draft(&mut self, request: &GuiDraftRequest) -> Result<DraftOutcome, String> {
        let Some(pull_direction) = request.pull_direction.clone() else {
            self.clear_extrusion_preview();
            return Ok(DraftOutcome::Incomplete);
        };
        if request.mode == DraftMode::PartingTool && request.parting_tool.is_none() {
            self.clear_extrusion_preview();
            return Ok(DraftOutcome::Incomplete);
        }
        if request.faces.is_empty() {
            self.clear_extrusion_preview();
            return Ok(DraftOutcome::Incomplete);
        }
        let body = match request.body.clone() {
            Some(body) => body,
            None => self.resolve_draft_body_for_face(request.document, &pull_direction)?,
        };
        let inputs = match seed::request_to_inputs(request, body, pull_direction) {
            Ok(inputs) => inputs,
            Err(cause) => {
                self.clear_extrusion_preview();
                return Err(cause);
            }
        };
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_draft_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                DraftCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited = build_draft_definition(&inputs).map_err(|cause| cause.to_string())?;
                DraftCommit::Edit(id, Box::new(edited))
            }
        };
        if request.preview_hidden {
            self.clear_extrusion_preview();
            return Ok(DraftOutcome::Ready(commit));
        }
        match self.evaluate_draft_preview_scene(request, &commit) {
            Ok(scene) => {
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Replacement,
                ));
                self.part_scenes.invalidate_presentation();
                Ok(DraftOutcome::Ready(commit))
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn evaluate_draft_preview_scene(
        &mut self,
        request: &GuiDraftRequest,
        commit: &DraftCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            DraftCommit::Create(prepared) => {
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
            DraftCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_draft(candidate, id, edited.clone()),
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

    fn resolve_draft_body_for_face(&self, document: u64, face: &[u8]) -> Result<Vec<u8>, String> {
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
                return Ok(body.body_reference_key(session.history(), guid)?.to_vec());
            }
        }
        Err("draft: expected a current solid face".into())
    }

    fn commit_draft(&mut self, document: u64, commit: DraftCommit) -> Result<(), String> {
        match commit {
            DraftCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Face Draft",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            DraftCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Face Draft",
                    |snapshot, _| replace_draft(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
