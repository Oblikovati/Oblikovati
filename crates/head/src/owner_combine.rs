// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, solid body picking, live Boolean preview, and commit for Part Modify Combine (#3918).
#[path = "owner_combine_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedCombine, build_combine_definition, prepare_combine_definition, replace_combine,
};
use oblikovati_gui::{
    GuiCombineAction, GuiCombineEditSeed, GuiCombinePick, GuiCombinePickMode, GuiCombinePreview,
    GuiCombinePreviewState, GuiCombineRequest,
};
use oblikovati_model::{
    feature::{FeatureId, combine::CombineDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum CombineCommit {
    Create(Box<PreparedCombine>),
    Edit(FeatureId, Box<CombineDefinition>),
}

enum CombineOutcome {
    Ready(CombineCommit),
    Incomplete,
}

pub(in crate::owner) struct CombinePreview {
    request: GuiCombineRequest,
    result: Result<CombineOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct CombineOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) pick_mode: GuiCombinePickMode,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiCombinePick>,
    pub(in crate::owner) preview: Option<CombinePreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiCombineEditSeed> {
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
    pub(in crate::owner) fn gui_combine_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiCombinePreviewState> {
        let preview = self.create_features.combine.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiCombinePreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(CombineOutcome::Ready(_)) => GuiCombinePreview::Ready,
                    Ok(CombineOutcome::Incomplete) => GuiCombinePreview::Incomplete,
                    Err(cause) => GuiCombinePreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_combine_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiCombinePick> {
        self.create_features
            .combine
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn pick_combine_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.combine.active {
            return Ok(None);
        }
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        let allowed = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::Body,
        ]);
        let candidates = self
            .part_scenes
            .part_candidates(&self.files, document, ray, &allowed)?;
        let Ok(Some(picked)) =
            candidates.resolve_hit(&allowed, &self.ambient_selection, tolerance.unwrap_or(0.0))
        else {
            return Ok(None);
        };
        let crate::part_scene::pick_candidates::PickTarget::Body(_, hex_key) = picked.target else {
            return Ok(None);
        };
        let body = seed::decode_hex_key(&hex_key)?;
        self.create_features.combine.pick_seq =
            self.create_features.combine.pick_seq.saturating_add(1);
        self.create_features.combine.pick = Some(GuiCombinePick {
            document,
            generation,
            sequence: self.create_features.combine.pick_seq,
            body,
        });
        Ok(Some(true))
    }

    pub(in crate::owner) fn combine_command(
        &mut self,
        action: GuiCombineAction,
    ) -> Result<bool, String> {
        match action {
            GuiCombineAction::Open => {
                self.create_features.combine.active = true;
                self.create_features.combine.pick_mode = GuiCombinePickMode::Base;
                Ok(false)
            }
            GuiCombineAction::OpenEdit(_) => {
                self.create_features.combine.active = true;
                self.create_features.combine.pick_mode = GuiCombinePickMode::Toolbodies;
                Ok(false)
            }
            GuiCombineAction::SetPickMode(mode) => {
                self.create_features.combine.pick_mode = mode;
                Ok(false)
            }
            GuiCombineAction::Clear => {
                let had_state = self.create_features.combine.active
                    | self.create_features.combine.preview.take().is_some()
                    | self.create_features.combine.pick.take().is_some();
                self.create_features.combine.active = false;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiCombineAction::Preview(request) => {
                let request = *request;
                self.require_combine_generation(&request)?;
                self.create_features.combine.active = true;
                let result = self.preview_combine(&request);
                self.create_features.combine.preview = Some(CombinePreview { request, result });
                Ok(true)
            }
            GuiCombineAction::Commit(request) => {
                let request = *request;
                self.require_combine_generation(&request)?;
                let commit = self
                    .create_features
                    .combine
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        CombineOutcome::Ready(commit) => Some(commit),
                        CombineOutcome::Incomplete => None,
                    })
                    .ok_or("combine: expected a current certified preview")?
                    .clone();
                self.commit_combine(request.document, commit)?;
                self.create_features.combine.preview = None;
                self.create_features.combine.pick = None;
                self.create_features.combine.active = false;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_combine_generation(&self, request: &GuiCombineRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("combine: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_combine(&mut self, request: &GuiCombineRequest) -> Result<CombineOutcome, String> {
        let Some(base_body) = request.base_body.clone() else {
            self.clear_extrusion_preview();
            return Ok(CombineOutcome::Incomplete);
        };
        if request.tool_bodies.is_empty() {
            self.clear_extrusion_preview();
            return Ok(CombineOutcome::Incomplete);
        }
        let inputs = seed::request_to_inputs(request, base_body);
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_combine_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                CombineCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited =
                    build_combine_definition(&inputs).map_err(|cause| cause.to_string())?;
                CombineCommit::Edit(id, Box::new(edited))
            }
        };
        match self.evaluate_combine_preview_scene(request, &commit) {
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
                Ok(CombineOutcome::Ready(commit))
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn evaluate_combine_preview_scene(
        &mut self,
        request: &GuiCombineRequest,
        commit: &CombineCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            CombineCommit::Create(prepared) => {
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
            CombineCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_combine(candidate, id, edited.clone()),
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

    fn commit_combine(&mut self, document: u64, commit: CombineCommit) -> Result<(), String> {
        match commit {
            CombineCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Combine",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            CombineCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Combine",
                    |snapshot, _| replace_combine(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
