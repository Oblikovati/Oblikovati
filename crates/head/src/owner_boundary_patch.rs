// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, live Surface Warm-Amber preview, and commit for Surface Boundary Patch (#3918).
#[path = "owner_boundary_patch_pick.rs"]
mod pick;
#[path = "owner_boundary_patch_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedBoundaryPatch, build_boundary_patch_definition, prepare_boundary_patch_definition,
    replace_boundary_patch,
};
use oblikovati_gui::{
    GuiBoundaryPatchAction, GuiBoundaryPatchEditSeed, GuiBoundaryPatchPick,
    GuiBoundaryPatchPreview, GuiBoundaryPatchPreviewState, GuiBoundaryPatchRequest,
    GuiBoundaryPatchSelectionMode,
};
use oblikovati_model::{
    feature::{FeatureId, boundary_patch::BoundaryPatchDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum BoundaryPatchCommit {
    Create(Box<PreparedBoundaryPatch>),
    Edit(FeatureId, Box<BoundaryPatchDefinition>),
}

enum BoundaryPatchOutcome {
    Ready(BoundaryPatchCommit),
    Incomplete,
}

pub(in crate::owner) struct BoundaryPatchPreview {
    request: GuiBoundaryPatchRequest,
    result: Result<BoundaryPatchOutcome, String>,
}

pub(in crate::owner) struct BoundaryPatchOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) automatic_edge_chain: bool,
    pub(in crate::owner) selection_mode: GuiBoundaryPatchSelectionMode,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiBoundaryPatchPick>,
    pub(in crate::owner) preview: Option<BoundaryPatchPreview>,
}

impl Default for BoundaryPatchOwnerState {
    fn default() -> Self {
        Self {
            active: false,
            automatic_edge_chain: true,
            selection_mode: GuiBoundaryPatchSelectionMode::Boundary,
            pick_seq: 0,
            pick: None,
            preview: None,
        }
    }
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiBoundaryPatchEditSeed> {
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
    pub(in crate::owner) fn gui_boundary_patch_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiBoundaryPatchPreviewState> {
        let preview = self.create_features.boundary_patch.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiBoundaryPatchPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(BoundaryPatchOutcome::Ready(_)) => GuiBoundaryPatchPreview::Ready,
                    Ok(BoundaryPatchOutcome::Incomplete) => GuiBoundaryPatchPreview::Incomplete,
                    Err(cause) => GuiBoundaryPatchPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_boundary_patch_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiBoundaryPatchPick> {
        self.create_features
            .boundary_patch
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn boundary_patch_command(
        &mut self,
        action: GuiBoundaryPatchAction,
    ) -> Result<bool, String> {
        match action {
            GuiBoundaryPatchAction::Open | GuiBoundaryPatchAction::OpenEdit(_) => {
                self.create_features.boundary_patch.active = true;
                Ok(false)
            }
            GuiBoundaryPatchAction::SetAutomaticEdgeChain(enabled) => {
                self.create_features.boundary_patch.automatic_edge_chain = enabled;
                Ok(false)
            }
            GuiBoundaryPatchAction::SetSelectionMode(mode) => {
                self.create_features.boundary_patch.selection_mode = mode;
                Ok(false)
            }
            GuiBoundaryPatchAction::Clear => {
                let had_state = self.create_features.boundary_patch.active
                    | self.create_features.boundary_patch.preview.take().is_some()
                    | self.create_features.boundary_patch.pick.take().is_some();
                self.create_features.boundary_patch.active = false;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiBoundaryPatchAction::Preview(request) => {
                let request = *request;
                self.require_boundary_patch_generation(&request)?;
                self.create_features.boundary_patch.active = true;
                self.create_features.boundary_patch.automatic_edge_chain =
                    request.automatic_edge_chain;
                let result = self.preview_boundary_patch(&request);
                self.create_features.boundary_patch.preview =
                    Some(BoundaryPatchPreview { request, result });
                Ok(true)
            }
            GuiBoundaryPatchAction::Commit(request) => {
                let request = *request;
                self.require_boundary_patch_generation(&request)?;
                let commit = self
                    .create_features
                    .boundary_patch
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        BoundaryPatchOutcome::Ready(commit) => Some(commit),
                        BoundaryPatchOutcome::Incomplete => None,
                    })
                    .ok_or("boundary patch: expected a current certified preview")?
                    .clone();
                self.commit_boundary_patch(request.document, commit)?;
                self.create_features.boundary_patch.preview = None;
                self.create_features.boundary_patch.pick = None;
                self.create_features.boundary_patch.active = false;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_boundary_patch_generation(
        &self,
        request: &GuiBoundaryPatchRequest,
    ) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("boundary patch: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_boundary_patch(
        &mut self,
        request: &GuiBoundaryPatchRequest,
    ) -> Result<BoundaryPatchOutcome, String> {
        let total_edges: usize = request.edge_sets.iter().map(|s| s.edges.len()).sum();
        if total_edges == 0 {
            self.clear_extrusion_preview();
            return Ok(BoundaryPatchOutcome::Incomplete);
        }
        let inputs = seed::request_to_inputs(request);
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_boundary_patch_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                BoundaryPatchCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited =
                    build_boundary_patch_definition(&inputs).map_err(|cause| cause.to_string())?;
                BoundaryPatchCommit::Edit(id, Box::new(edited))
            }
        };
        match self.evaluate_boundary_patch_preview_scene(request, &commit) {
            Ok(scene) => {
                if request.preview_hidden {
                    self.clear_extrusion_preview();
                } else {
                    self.extrusion_preview = Some((
                        request.document,
                        request.generation,
                        scene,
                        super::super::owner_extrusion_preview::PreviewKind::Additive,
                    ));
                    self.part_scenes.invalidate_presentation();
                }
                Ok(BoundaryPatchOutcome::Ready(commit))
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                if total_edges == 1 && cause.contains("at least two boundary edges") {
                    return Ok(BoundaryPatchOutcome::Incomplete);
                }
                Err(cause)
            }
        }
    }

    fn evaluate_boundary_patch_preview_scene(
        &mut self,
        request: &GuiBoundaryPatchRequest,
        commit: &BoundaryPatchCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            BoundaryPatchCommit::Create(prepared) => {
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
            BoundaryPatchCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_boundary_patch(candidate, id, edited.clone()),
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

    fn commit_boundary_patch(
        &mut self,
        document: u64,
        commit: BoundaryPatchCommit,
    ) -> Result<(), String> {
        match commit {
            BoundaryPatchCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Boundary Patch",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            BoundaryPatchCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Boundary Patch",
                    |snapshot, _| replace_boundary_patch(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
