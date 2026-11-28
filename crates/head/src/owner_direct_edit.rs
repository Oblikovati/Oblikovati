// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, face/solid picking, live scale preview, and commits for Direct Edit (#3918).
#[path = "owner_direct_edit_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedDirectEdit, build_direct_edit_definition, prepare_direct_edit_definition,
    replace_direct_edit,
};
use oblikovati_gui::{
    GuiDirectEditAction, GuiDirectEditEditSeed, GuiDirectEditPick, GuiDirectEditPreview,
    GuiDirectEditPreviewState, GuiDirectEditRequest,
};
use oblikovati_model::{
    feature::{
        FeatureId,
        direct_edit::{DirectEditDefinition, DirectEditMode, DirectEditSelectionFilter},
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum DirectEditCommit {
    Create(Box<PreparedDirectEdit>),
    Edit(FeatureId, Box<DirectEditDefinition>),
}

enum DirectEditOutcome {
    Ready(DirectEditCommit),
    Incomplete,
}

pub(in crate::owner) struct DirectEditPreview {
    request: GuiDirectEditRequest,
    result: Result<DirectEditOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct DirectEditOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) filter: DirectEditSelectionFilter,
    pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiDirectEditPick>,
    pub(in crate::owner) preview: Option<DirectEditPreview>,
    pub(in crate::owner) last_committed_feature: Option<FeatureId>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiDirectEditEditSeed> {
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
    pub(in crate::owner) fn gui_direct_edit_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiDirectEditPreviewState> {
        let state = &self.create_features.direct_edit;
        if let Some(preview) = state.preview.as_ref()
            && preview.request.document == document
            && preview.request.generation == generation
        {
            return Some(GuiDirectEditPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(DirectEditOutcome::Ready(_)) => GuiDirectEditPreview::Ready,
                    Ok(DirectEditOutcome::Incomplete) => GuiDirectEditPreview::Incomplete,
                    Err(cause) => GuiDirectEditPreview::Failed(cause.clone()),
                },
                anchor_pixels: None,
                last_committed_feature: state.last_committed_feature,
            });
        }
        None
    }

    pub(in crate::owner) fn gui_direct_edit_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiDirectEditPick> {
        self.create_features
            .direct_edit
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn pick_direct_edit_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.direct_edit.active {
            return Ok(None);
        }
        let filter = self.create_features.direct_edit.filter;
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let kind = match filter {
            DirectEditSelectionFilter::Faces => oblikovati_session::selection::SelectionKind::Face,
            DirectEditSelectionFilter::Solids => oblikovati_session::selection::SelectionKind::Body,
        };
        let allowed = oblikovati_session::selection::SelectionFilter::only([kind]);
        let candidates = self
            .part_scenes
            .part_candidates(&self.files, document, ray, &allowed)?;
        let Ok(Some(picked)) =
            candidates.resolve_hit(&allowed, &self.ambient_selection, tolerance.unwrap_or(0.0))
        else {
            return Ok(None);
        };
        match (filter, picked.target) {
            (
                DirectEditSelectionFilter::Faces,
                crate::part_scene::pick_candidates::PickTarget::Face(feature, key),
            ) => {
                let body = self.resolve_direct_edit_body_for_face(document, &key)?;
                self.highlight_direct_edit_face(document, feature, key.clone())?;
                self.record_direct_edit_pick(document, generation, filter, body, Some(key));
                Ok(Some(true))
            }
            (
                DirectEditSelectionFilter::Solids,
                crate::part_scene::pick_candidates::PickTarget::Body(feature, hex_key),
            ) => {
                let body = seed::decode_hex_key(&hex_key)?;
                self.highlight_direct_edit_body(document, feature, hex_key)?;
                self.record_direct_edit_pick(document, generation, filter, body, None);
                Ok(Some(true))
            }
            _ => Ok(None),
        }
    }

    fn record_direct_edit_pick(
        &mut self,
        document: u64,
        generation: u64,
        filter: DirectEditSelectionFilter,
        body: Vec<u8>,
        face: Option<Vec<u8>>,
    ) {
        self.create_features.direct_edit.pick_seq =
            self.create_features.direct_edit.pick_seq.saturating_add(1);
        self.create_features.direct_edit.pick = Some(GuiDirectEditPick {
            document,
            generation,
            sequence: self.create_features.direct_edit.pick_seq,
            filter,
            body,
            face,
            anchor_pixels: None,
        });
    }

    fn highlight_direct_edit_face(
        &mut self,
        document: u64,
        feature: FeatureId,
        key: Vec<u8>,
    ) -> Result<(), String> {
        let _ = self.browser_choice(document, &oblikovati_gui::GuiBrowserChoice::Nothing);
        self.face_appearance_picks.clear();
        self.model_face = Some(super::super::browser_selection::ChosenModelFace {
            document,
            feature,
            key,
        });
        self.browser_outline = None;
        Ok(())
    }

    fn highlight_direct_edit_body(
        &mut self,
        document: u64,
        feature: FeatureId,
        hex_key: String,
    ) -> Result<(), String> {
        self.model_face = None;
        self.face_appearance_picks.clear();
        let _ = self.browser_choice(
            document,
            &oblikovati_gui::GuiBrowserChoice::Body {
                feature,
                key: hex_key,
                toggle: false,
            },
        );
        self.browser_outline = None;
        Ok(())
    }

    fn clear_direct_edit_highlight(&mut self) {
        if let Some(doc) = self.files.workspace().active_document() {
            let _ = self.browser_choice(doc, &oblikovati_gui::GuiBrowserChoice::Nothing);
        }
        self.model_face = None;
        self.face_appearance_picks.clear();
        self.browser_outline = None;
    }

    pub(in crate::owner) fn direct_edit_command(
        &mut self,
        action: GuiDirectEditAction,
    ) -> Result<bool, String> {
        match action {
            GuiDirectEditAction::Open => {
                self.create_features.direct_edit.active = true;
                self.create_features.direct_edit.filter = DirectEditSelectionFilter::Faces;
                self.create_features.direct_edit.last_committed_feature = None;
                Ok(false)
            }
            GuiDirectEditAction::OpenEdit { feature, .. } => {
                self.create_features.direct_edit.active = true;
                self.create_features.direct_edit.last_committed_feature = Some(feature);
                Ok(false)
            }
            GuiDirectEditAction::SetFilter(filter) => {
                self.create_features.direct_edit.filter = filter;
                Ok(false)
            }
            GuiDirectEditAction::ClearSelection => {
                self.create_features.direct_edit.preview = None;
                self.create_features.direct_edit.pick = None;
                self.clear_direct_edit_highlight();
                Ok(self.clear_extrusion_preview() || true)
            }
            GuiDirectEditAction::Clear => {
                let had_state = self.create_features.direct_edit.active
                    | self.create_features.direct_edit.preview.take().is_some()
                    | self.create_features.direct_edit.pick.take().is_some();
                self.create_features.direct_edit.active = false;
                self.create_features.direct_edit.last_committed_feature = None;
                self.clear_direct_edit_highlight();
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiDirectEditAction::Preview(request) => {
                let request = *request;
                self.require_direct_edit_generation(&request)?;
                self.create_features.direct_edit.active = true;
                self.create_features.direct_edit.filter = request.filter;
                let result = self.preview_direct_edit(&request);
                self.create_features.direct_edit.preview =
                    Some(DirectEditPreview { request, result });
                Ok(true)
            }
            GuiDirectEditAction::Commit(request) => {
                let request = *request;
                self.require_direct_edit_generation(&request)?;
                let commit = self
                    .create_features
                    .direct_edit
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        DirectEditOutcome::Ready(commit) => Some(commit),
                        DirectEditOutcome::Incomplete => None,
                    })
                    .ok_or("direct edit: expected a current certified preview")?
                    .clone();
                let committed = self.commit_direct_edit(request.document, commit)?;
                self.create_features.direct_edit.last_committed_feature = Some(committed);
                self.create_features.direct_edit.preview = None;
                self.create_features.direct_edit.pick = None;
                self.clear_direct_edit_highlight();
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_direct_edit_generation(&self, request: &GuiDirectEditRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("direct edit: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_direct_edit(
        &mut self,
        request: &GuiDirectEditRequest,
    ) -> Result<DirectEditOutcome, String> {
        if !seed::request_has_selection(request) {
            self.clear_extrusion_preview();
            return Ok(DirectEditOutcome::Incomplete);
        }
        let Some(body) = request.body.clone() else {
            self.clear_extrusion_preview();
            return Ok(DirectEditOutcome::Incomplete);
        };
        let snapshot = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let inputs = match seed::build_feature_inputs(snapshot, request, body) {
            Ok(inputs) => inputs,
            Err(cause) => {
                self.clear_extrusion_preview();
                return Err(cause);
            }
        };
        let commit = match request.feature {
            None => {
                let prepared = prepare_direct_edit_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                DirectEditCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited =
                    build_direct_edit_definition(&inputs).map_err(|cause| cause.to_string())?;
                DirectEditCommit::Edit(id, Box::new(edited))
            }
        };
        if request.mode != DirectEditMode::Scale {
            self.clear_extrusion_preview();
            return Ok(DirectEditOutcome::Ready(commit));
        }
        match self.evaluate_direct_edit_preview_scene(request, &commit) {
            Ok(scene) => {
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Replacement,
                ));
                self.part_scenes.invalidate_presentation();
                Ok(DirectEditOutcome::Ready(commit))
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn evaluate_direct_edit_preview_scene(
        &mut self,
        request: &GuiDirectEditRequest,
        commit: &DirectEditCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            DirectEditCommit::Create(prepared) => {
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
            DirectEditCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_direct_edit(candidate, id, edited.clone()),
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

    fn resolve_direct_edit_body_for_face(
        &self,
        document: u64,
        face: &[u8],
    ) -> Result<Vec<u8>, String> {
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
        Err("direct edit: expected a current solid face".into())
    }

    fn commit_direct_edit(
        &mut self,
        document: u64,
        commit: DirectEditCommit,
    ) -> Result<FeatureId, String> {
        match commit {
            DirectEditCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Direct Edit",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map(|(id, _)| id)
                .map_err(|cause| cause.to_string())
            }
            DirectEditCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Direct Edit",
                    |snapshot, _| replace_direct_edit(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
                Ok(id)
            }
        }
    }
}
