// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, live preview, and transactional commit for Part Surface Extend (#3918).
#[path = "owner_extend_surface_pick.rs"]
mod pick;
#[path = "owner_extend_surface_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedExtendSurface, build_extend_surface_definition, prepare_extend_surface_definition,
    replace_extend_surface,
};
use oblikovati_gui::{
    GuiExtendSurfaceAction, GuiExtendSurfaceEditSeed, GuiExtendSurfacePick,
    GuiExtendSurfacePreview, GuiExtendSurfacePreviewState, GuiExtendSurfaceRequest,
};
use oblikovati_model::{
    feature::{FeatureId, extend_surface::ExtendSurfaceDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum ExtendSurfaceCommit {
    Create(Box<PreparedExtendSurface>),
    Edit(FeatureId, Box<ExtendSurfaceDefinition>),
}

enum ExtendSurfaceOutcome {
    Ready {
        commit: ExtendSurfaceCommit,
        area_mm2: f64,
    },
    Incomplete,
}

pub(in crate::owner) struct ExtendSurfacePreview {
    request: GuiExtendSurfaceRequest,
    result: Result<ExtendSurfaceOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct ExtendSurfaceOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) edge_chain: bool,
    pub(in crate::owner) picking_termination: bool,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiExtendSurfacePick>,
    pub(in crate::owner) preview: Option<ExtendSurfacePreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiExtendSurfaceEditSeed> {
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
    pub(in crate::owner) fn gui_extend_surface_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiExtendSurfacePreviewState> {
        let preview = self.create_features.extend_surface.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiExtendSurfacePreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(ExtendSurfaceOutcome::Ready { area_mm2, .. }) => {
                        GuiExtendSurfacePreview::Ready {
                            area_mm2: *area_mm2,
                        }
                    }
                    Ok(ExtendSurfaceOutcome::Incomplete) => GuiExtendSurfacePreview::Incomplete,
                    Err(cause) => GuiExtendSurfacePreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_extend_surface_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiExtendSurfacePick> {
        self.create_features
            .extend_surface
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn extend_surface_command(
        &mut self,
        action: GuiExtendSurfaceAction,
    ) -> Result<bool, String> {
        match action {
            GuiExtendSurfaceAction::Open | GuiExtendSurfaceAction::OpenEdit(_) => {
                self.create_features.extend_surface.active = true;
                self.create_features.extend_surface.edge_chain = true;
                self.create_features.extend_surface.picking_termination = false;
                Ok(false)
            }
            GuiExtendSurfaceAction::SetEdgeChain(enabled) => {
                self.create_features.extend_surface.edge_chain = enabled;
                Ok(false)
            }
            GuiExtendSurfaceAction::SetPickTermination(enabled) => {
                self.create_features.extend_surface.picking_termination = enabled;
                Ok(false)
            }
            GuiExtendSurfaceAction::Clear => {
                let had_state = self.create_features.extend_surface.active
                    | self.create_features.extend_surface.preview.take().is_some()
                    | self.create_features.extend_surface.pick.take().is_some()
                    | !self.model_edges.is_empty();
                self.create_features.extend_surface.active = false;
                self.create_features.extend_surface.picking_termination = false;
                self.model_edges.clear();
                self.selected_edge_outlines = None;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiExtendSurfaceAction::Preview(request) => {
                let request = *request;
                self.require_extend_surface_generation(&request)?;
                self.create_features.extend_surface.active = true;
                self.create_features.extend_surface.edge_chain = request.edge_chain;
                let result = self.preview_extend_surface(&request);
                self.create_features.extend_surface.preview =
                    Some(ExtendSurfacePreview { request, result });
                Ok(true)
            }
            GuiExtendSurfaceAction::Commit(request) => {
                let request = *request;
                self.require_extend_surface_generation(&request)?;
                let commit = self
                    .create_features
                    .extend_surface
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        ExtendSurfaceOutcome::Ready { commit, .. } => Some(commit),
                        ExtendSurfaceOutcome::Incomplete => None,
                    })
                    .ok_or("extend surface: expected a current certified preview")?
                    .clone();
                self.commit_extend_surface(request.document, commit)?;
                self.create_features.extend_surface.preview = None;
                self.create_features.extend_surface.pick = None;
                self.create_features.extend_surface.active = false;
                self.create_features.extend_surface.picking_termination = false;
                self.model_edges.clear();
                self.selected_edge_outlines = None;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_extend_surface_generation(
        &self,
        request: &GuiExtendSurfaceRequest,
    ) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("extend surface: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_extend_surface(
        &mut self,
        request: &GuiExtendSurfaceRequest,
    ) -> Result<ExtendSurfaceOutcome, String> {
        let (Some(target_body), Some(extent)) =
            (request.target_body.clone(), request.resolved_extent())
        else {
            self.clear_extrusion_preview();
            return Ok(ExtendSurfaceOutcome::Incomplete);
        };
        if request.edges.is_empty() {
            self.model_edges.clear();
            self.selected_edge_outlines = None;
            self.clear_extrusion_preview();
            return Ok(ExtendSurfaceOutcome::Incomplete);
        }
        let inputs = seed::request_to_inputs(request, target_body, extent);
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_extend_surface_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                ExtendSurfaceCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited =
                    build_extend_surface_definition(&inputs).map_err(|cause| cause.to_string())?;
                ExtendSurfaceCommit::Edit(id, Box::new(edited))
            }
        };
        match self.evaluate_extend_surface_preview(request, &commit) {
            Ok((scene, area_mm2)) => {
                if request.preview_hidden {
                    self.clear_extrusion_preview();
                } else {
                    self.extrusion_preview = Some((
                        request.document,
                        request.generation,
                        scene,
                        super::super::owner_extrusion_preview::PreviewKind::BlendInput,
                    ));
                    self.part_scenes.invalidate_presentation();
                }
                Ok(ExtendSurfaceOutcome::Ready { commit, area_mm2 })
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn evaluate_extend_surface_preview(
        &mut self,
        request: &GuiExtendSurfaceRequest,
        commit: &ExtendSurfaceCommit,
    ) -> Result<(oblikovati_viewport::ViewportScene, f64), String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(request.document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let policy = self.extrusion_access.as_ref().map(|access| access.0);
        match commit {
            ExtendSurfaceCommit::Create(prepared) => {
                let cloned = (**prepared).clone();
                session
                    .preview_projected(
                        |candidate, ids| cloned.create(candidate, ids),
                        |candidate, id| {
                            self.project_extend_surface_preview(
                                request.document,
                                candidate,
                                id,
                                session.history(),
                                guid,
                                policy,
                            )
                        },
                    )
                    .map_err(|cause| cause.to_string())
            }
            ExtendSurfaceCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                session
                    .preview_projected(
                        |candidate, _| replace_extend_surface(candidate, id, edited.clone()),
                        |candidate, ()| {
                            self.project_extend_surface_preview(
                                request.document,
                                candidate,
                                id,
                                session.history(),
                                guid,
                                policy,
                            )
                        },
                    )
                    .map_err(|cause| cause.to_string())
            }
        }
    }

    fn project_extend_surface_preview(
        &self,
        document: u64,
        candidate: &DocumentSnapshot,
        id: FeatureId,
        history: &oblikovati_model::history::DocumentEditHistory,
        guid: oblikovati_model::identity::DocumentGuid,
        policy: Option<oblikovati_martensite::PrismResolutionPolicy>,
    ) -> Result<(oblikovati_viewport::ViewportScene, f64), oblikovati_api::Error> {
        let scene = self
            .part_scenes
            .project_feature_edit(&self.files, document, candidate, id)
            .map_err(|cause| oblikovati_api::Error(cause.to_string()))?;
        let preview = self
            .files
            .workspace()
            .preview_owned_part(document, policy, candidate)
            .map_err(|cause| oblikovati_api::Error(cause.to_string()))?;
        let area = preview
            .evaluation
            .body(id)
            .ok_or_else(|| {
                oblikovati_api::Error("extend surface preview: expected extended body".into())
            })?
            .surface_area_mm2(history, guid)
            .map_err(oblikovati_api::Error)?;
        Ok((scene, area))
    }

    fn commit_extend_surface(
        &mut self,
        document: u64,
        commit: ExtendSurfaceCommit,
    ) -> Result<(), String> {
        match commit {
            ExtendSurfaceCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Extend Surface",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            ExtendSurfaceCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Extend Surface",
                    |snapshot, _| replace_extend_surface(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
