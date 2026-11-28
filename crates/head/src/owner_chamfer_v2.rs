// SPDX-License-Identifier: GPL-2.0-only
//! Face/partial-edge picking, live preview, Apply continuation, and commit for Chamfer v2 (#4046).
#[path = "owner_chamfer_v2_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedChamfer, build_chamfer_v2_definition, prepare_chamfer_v2, replace_chamfer,
};
use oblikovati_gui::{
    GuiChamferPickMode, GuiChamferV2Action, GuiChamferV2EditSeed, GuiChamferV2Pick,
    GuiChamferV2Preview, GuiChamferV2PreviewState, GuiChamferV2Request,
};
use oblikovati_model::{
    feature::{FeatureId, chamfer::ChamferDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum ChamferV2Commit {
    Create(Box<PreparedChamfer>),
    Edit(FeatureId, Box<ChamferDefinition>),
}

enum ChamferV2Outcome {
    Ready(ChamferV2Commit),
    Incomplete,
}

pub(in crate::owner) struct ChamferV2Preview {
    request: GuiChamferV2Request,
    result: Result<ChamferV2Outcome, String>,
    edge_length_cm: Option<f64>,
    edge_endpoints: Option<[[f64; 3]; 2]>,
}

#[derive(Default)]
pub(in crate::owner) struct ChamferV2OwnerState {
    pub(in crate::owner) pick_mode: GuiChamferPickMode,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiChamferV2Pick>,
    pub(in crate::owner) preview: Option<ChamferV2Preview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
    generation: u64,
) -> BTreeMap<FeatureId, GuiChamferV2EditSeed> {
    seed::edit_seeds(snapshot, generation)
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
    pub(in crate::owner) fn gui_chamfer_v2_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiChamferV2PreviewState> {
        let preview = self.create_features.chamfer_v2.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiChamferV2PreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(ChamferV2Outcome::Ready(_)) => GuiChamferV2Preview::Ready,
                    Ok(ChamferV2Outcome::Incomplete) => GuiChamferV2Preview::Incomplete,
                    Err(cause) => GuiChamferV2Preview::Failed(cause.clone()),
                },
                edge_length_cm: preview.edge_length_cm,
                edge_endpoints: preview.edge_endpoints,
            },
        )
    }

    pub(in crate::owner) fn gui_chamfer_v2_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiChamferV2Pick> {
        self.create_features
            .chamfer_v2
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn chamfer_v2_command(
        &mut self,
        action: GuiChamferV2Action,
    ) -> Result<bool, String> {
        match action {
            GuiChamferV2Action::SetPickMode(mode) => {
                self.create_features.chamfer_v2.pick_mode = mode;
                Ok(true)
            }
            GuiChamferV2Action::Clear => {
                let had = self.create_features.chamfer_v2.pick_mode != GuiChamferPickMode::None
                    || self.create_features.chamfer_v2.preview.take().is_some()
                    || self.create_features.chamfer_v2.pick.take().is_some();
                self.create_features.chamfer_v2.pick_mode = GuiChamferPickMode::None;
                Ok(had | self.clear_extrusion_preview())
            }
            GuiChamferV2Action::Preview(request) => {
                let request = *request;
                self.require_chamfer_v2_generation(&request)?;
                let (edge_length_cm, edge_endpoints) = self.measure_chamfer_edge(&request);
                let result = self.preview_chamfer_v2(&request);
                self.create_features.chamfer_v2.preview = Some(ChamferV2Preview {
                    request,
                    result,
                    edge_length_cm,
                    edge_endpoints,
                });
                Ok(true)
            }
            GuiChamferV2Action::Commit(request) => {
                let request = *request;
                self.finish_chamfer_v2(request, true)
            }
            GuiChamferV2Action::Apply(request) => {
                let request = *request;
                self.finish_chamfer_v2(request, false)
            }
        }
    }

    fn finish_chamfer_v2(
        &mut self,
        request: GuiChamferV2Request,
        clear_pick_mode: bool,
    ) -> Result<bool, String> {
        self.require_chamfer_v2_generation(&request)?;
        let commit = self
            .create_features
            .chamfer_v2
            .preview
            .as_ref()
            .filter(|preview| preview.request == request)
            .and_then(|preview| preview.result.as_ref().ok())
            .and_then(|outcome| match outcome {
                ChamferV2Outcome::Ready(commit) => Some(commit),
                ChamferV2Outcome::Incomplete => None,
            })
            .ok_or("chamfer: expected a current certified preview")?
            .clone();
        self.commit_chamfer_v2(request.document, commit)?;
        self.create_features.chamfer_v2.preview = None;
        self.create_features.chamfer_v2.pick = None;
        if clear_pick_mode {
            self.create_features.chamfer_v2.pick_mode = GuiChamferPickMode::None;
        }
        self.model_edges.clear();
        self.clear_extrusion_preview();
        Ok(true)
    }

    pub(in crate::owner) fn pick_chamfer_v2_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let mode = self.create_features.chamfer_v2.pick_mode;
        if mode == GuiChamferPickMode::None {
            return Ok(None);
        }
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        match mode {
            GuiChamferPickMode::Face => {
                let allowed = oblikovati_session::selection::SelectionFilter::only([
                    oblikovati_session::selection::SelectionKind::Face,
                ]);
                let candidates =
                    self.part_scenes
                        .part_candidates(&self.files, document, ray, &allowed)?;
                let Ok(Some(picked)) = candidates.resolve_hit(
                    &allowed,
                    &self.ambient_selection,
                    tolerance.unwrap_or(0.0),
                ) else {
                    return Ok(None);
                };
                let crate::part_scene::pick_candidates::PickTarget::Face(feature, face_key) =
                    picked.target
                else {
                    return Ok(None);
                };
                let workspace = self.files.workspace();
                let session = workspace
                    .session(document)
                    .map_err(|cause| cause.to_string())?;
                let guid = workspace
                    .identity(document)
                    .map_err(|cause| cause.to_string())?
                    .internal_name;
                let body = self
                    .part_scenes
                    .current(session)?
                    .body(feature)
                    .ok_or("chamfer face pick: expected a current body")?;
                let body_key = body.body_reference_key(session.history(), guid)?.to_vec();
                self.create_features.chamfer_v2.pick_seq += 1;
                self.create_features.chamfer_v2.pick = Some(GuiChamferV2Pick {
                    document,
                    generation,
                    sequence: self.create_features.chamfer_v2.pick_seq,
                    mode,
                    body: body_key,
                    face: Some(face_key),
                });
                self.create_features.chamfer_v2.pick_mode = GuiChamferPickMode::None;
                Ok(Some(true))
            }
            GuiChamferPickMode::PartialEdge => {
                let allowed = oblikovati_session::selection::SelectionFilter::only([
                    oblikovati_session::selection::SelectionKind::Edge,
                ]);
                let mut candidates =
                    self.part_scenes
                        .part_candidates(&self.files, document, ray, &allowed)?;
                if let Some(tol) = tolerance {
                    let _ = self.part_scenes.append_edge_candidates(
                        &self.files,
                        document,
                        ray,
                        tol,
                        &mut candidates,
                    );
                    candidates.include_edges();
                }
                let Ok(Some(picked)) = candidates.resolve_hit(
                    &allowed,
                    &self.ambient_selection,
                    tolerance.unwrap_or(0.0),
                ) else {
                    return Ok(None);
                };
                if let crate::part_scene::pick_candidates::PickTarget::Edge(_, _) = picked.target {
                    self.create_features.chamfer_v2.pick_seq += 1;
                    self.create_features.chamfer_v2.pick = Some(GuiChamferV2Pick {
                        document,
                        generation,
                        sequence: self.create_features.chamfer_v2.pick_seq,
                        mode,
                        body: Vec::new(),
                        face: None,
                    });
                    return Ok(Some(true));
                }
                Ok(None)
            }
            GuiChamferPickMode::None => Ok(None),
        }
    }

    fn require_chamfer_v2_generation(&self, request: &GuiChamferV2Request) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("chamfer: expected the current document generation".into());
        }
        Ok(())
    }

    fn measure_chamfer_edge(
        &self,
        request: &GuiChamferV2Request,
    ) -> (Option<f64>, Option<[[f64; 3]; 2]>) {
        let Some(edge) = request.edges.first() else {
            return (None, None);
        };
        let workspace = self.files.workspace();
        let Ok(session) = workspace.session(request.document) else {
            return (None, None);
        };
        let Ok(guid) = workspace
            .identity(request.document)
            .map(|id| id.internal_name)
        else {
            return (None, None);
        };
        let Ok(evaluation) = self.part_scenes.current(session) else {
            return (None, None);
        };
        for body in evaluation.ordered_bodies() {
            if let Ok((p0, p1, len)) = body.edge_span(session.history(), guid, edge) {
                return (Some(len), Some([p0, p1]));
            }
        }
        (None, None)
    }

    fn preview_chamfer_v2(
        &mut self,
        request: &GuiChamferV2Request,
    ) -> Result<ChamferV2Outcome, String> {
        if request.edges.is_empty() {
            self.clear_extrusion_preview();
            return Ok(ChamferV2Outcome::Incomplete);
        }
        let body = match request.body.clone() {
            Some(body) if !body.is_empty() => body,
            _ => self.resolve_body_for_edge(request.document, &request.edges[0])?,
        };
        let inputs = seed::request_to_inputs(request, body)?;
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared =
                    prepare_chamfer_v2(snapshot, &inputs).map_err(|cause| cause.to_string())?;
                ChamferV2Commit::Create(Box::new(prepared))
            }
            Some(id) => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let edited = build_chamfer_v2_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                ChamferV2Commit::Edit(id, Box::new(edited))
            }
        };
        match self.evaluate_chamfer_v2_preview_scene(request, &commit) {
            Ok(scene) => {
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Replacement,
                ));
                self.part_scenes.invalidate_presentation();
                Ok(ChamferV2Outcome::Ready(commit))
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn evaluate_chamfer_v2_preview_scene(
        &mut self,
        request: &GuiChamferV2Request,
        commit: &ChamferV2Commit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            ChamferV2Commit::Create(prepared) => {
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
            ChamferV2Commit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_chamfer(candidate, id, edited.clone()),
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

    fn commit_chamfer_v2(&mut self, document: u64, commit: ChamferV2Commit) -> Result<(), String> {
        match commit {
            ChamferV2Commit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Chamfer",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            ChamferV2Commit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Chamfer",
                    |snapshot, _| replace_chamfer(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
