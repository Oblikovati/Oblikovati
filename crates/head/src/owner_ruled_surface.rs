// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, edge picking (with Automatic Edge Chain), vector picking, live preview + direction arrow, and commit for Part Surface Ruled Surface (#3918).
#[path = "owner_ruled_surface_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedRuledSurface, build_ruled_surface_definition, prepare_ruled_surface_definition,
    replace_ruled_surface,
};
use oblikovati_gui::{
    GuiRuledSurfaceAction, GuiRuledSurfaceArrow, GuiRuledSurfaceEditSeed, GuiRuledSurfacePick,
    GuiRuledSurfacePickTarget, GuiRuledSurfacePreview, GuiRuledSurfacePreviewState,
    GuiRuledSurfaceRequest,
};
use oblikovati_model::{
    feature::{
        FeatureId,
        ruled_surface::{RuledSurfaceDefinition, RuledSurfaceType, RuledSurfaceVector},
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum RuledSurfaceCommit {
    Create(Box<PreparedRuledSurface>),
    Edit(FeatureId, Box<RuledSurfaceDefinition>),
}

enum RuledSurfaceOutcome {
    Ready(RuledSurfaceCommit),
    Incomplete,
}

pub(in crate::owner) struct RuledSurfacePreview {
    request: GuiRuledSurfaceRequest,
    result: Result<RuledSurfaceOutcome, String>,
    arrow: Option<GuiRuledSurfaceArrow>,
}

struct ResolvedEdgePick {
    generation: u64,
    target_body: Vec<u8>,
    edges: Vec<Vec<u8>>,
}

#[derive(Default)]
pub(in crate::owner) struct RuledSurfaceOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) automatic_edge_chain: bool,
    pub(in crate::owner) pick_target: GuiRuledSurfacePickTarget,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiRuledSurfacePick>,
    pub(in crate::owner) preview: Option<RuledSurfacePreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiRuledSurfaceEditSeed> {
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
    pub(in crate::owner) fn gui_ruled_surface_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiRuledSurfacePreviewState> {
        let preview = self.create_features.ruled_surface.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiRuledSurfacePreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(RuledSurfaceOutcome::Ready(_)) => GuiRuledSurfacePreview::Ready,
                    Ok(RuledSurfaceOutcome::Incomplete) => GuiRuledSurfacePreview::Incomplete,
                    Err(cause) => GuiRuledSurfacePreview::Failed(cause.clone()),
                },
                arrow: preview.arrow,
            },
        )
    }

    pub(in crate::owner) fn gui_ruled_surface_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiRuledSurfacePick> {
        self.create_features
            .ruled_surface
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn pick_ruled_surface_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.ruled_surface.active {
            return Ok(None);
        }
        let edge_tol = tolerance.unwrap_or(0.12).max(0.10);
        let pick_target = self.create_features.ruled_surface.pick_target;
        let chain = self.create_features.ruled_surface.automatic_edge_chain;
        let Some(resolved) =
            self.resolve_ruled_edge_pick(document, ray, edge_tol, pick_target, chain)?
        else {
            return Ok(None);
        };
        self.create_features.ruled_surface.pick_seq = self
            .create_features
            .ruled_surface
            .pick_seq
            .saturating_add(1);
        let pick = match pick_target {
            GuiRuledSurfacePickTarget::Edges => GuiRuledSurfacePick {
                document,
                generation: resolved.generation,
                sequence: self.create_features.ruled_surface.pick_seq,
                target: GuiRuledSurfacePickTarget::Edges,
                target_body: Some(resolved.target_body),
                edges: resolved.edges,
                vector: None,
            },
            GuiRuledSurfacePickTarget::Vector => {
                let Some(edge) = resolved.edges.into_iter().next() else {
                    return Ok(None);
                };
                GuiRuledSurfacePick {
                    document,
                    generation: resolved.generation,
                    sequence: self.create_features.ruled_surface.pick_seq,
                    target: GuiRuledSurfacePickTarget::Vector,
                    target_body: None,
                    edges: Vec::new(),
                    vector: Some(RuledSurfaceVector::Edge {
                        body: resolved.target_body,
                        edge,
                    }),
                }
            }
        };
        self.create_features.ruled_surface.pick = Some(pick);
        Ok(Some(true))
    }

    fn resolve_ruled_edge_pick(
        &self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: f64,
        pick_target: GuiRuledSurfacePickTarget,
        automatic_edge_chain: bool,
    ) -> Result<Option<ResolvedEdgePick>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let generation = session.generation();
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let mut best: Option<(bool, f64, f64, FeatureId, Vec<u8>)> = None;
        for (feature, body) in evaluation.ordered_body_outputs() {
            let key = body.body_reference_key(session.history(), guid)?;
            let key_text: String = key.iter().map(|b| format!("{b:02x}")).collect();
            if !session.body_visible(&key_text) {
                continue;
            }
            let is_surface = body.is_surface();
            for hit in body.edge_hits_on_ray(session.history(), guid, ray, tolerance)? {
                if hit.distance > tolerance {
                    continue;
                }
                let better = best
                    .as_ref()
                    .is_none_or(|(prev_srf, prev_dist, prev_reach, _, _)| {
                        if pick_target == GuiRuledSurfacePickTarget::Edges
                            && is_surface != *prev_srf
                        {
                            is_surface
                        } else {
                            hit.distance < *prev_dist
                                || ((hit.distance - *prev_dist).abs() < 1.0e-6
                                    && hit.reach < *prev_reach)
                        }
                    });
                if better {
                    best = Some((is_surface, hit.distance, hit.reach, feature, hit.key));
                }
            }
        }
        let Some((_, _, _, feature, edge_key)) = best else {
            return Ok(None);
        };
        let Some(body) = evaluation.body(feature) else {
            return Ok(None);
        };
        let target_body = body.body_reference_key(session.history(), guid)?.to_vec();
        let edges = if pick_target == GuiRuledSurfacePickTarget::Edges && automatic_edge_chain {
            body.boundary_edge_chain(session.history(), guid, &edge_key)?
        } else {
            vec![edge_key]
        };
        Ok(Some(ResolvedEdgePick {
            generation,
            target_body,
            edges,
        }))
    }

    pub(in crate::owner) fn ruled_surface_command(
        &mut self,
        action: GuiRuledSurfaceAction,
    ) -> Result<bool, String> {
        match action {
            GuiRuledSurfaceAction::Open | GuiRuledSurfaceAction::OpenEdit(_) => {
                self.create_features.ruled_surface.active = true;
                self.create_features.ruled_surface.automatic_edge_chain = true;
                self.create_features.ruled_surface.pick_target = GuiRuledSurfacePickTarget::Edges;
                Ok(false)
            }
            GuiRuledSurfaceAction::SetAutomaticEdgeChain(enabled) => {
                self.create_features.ruled_surface.automatic_edge_chain = enabled;
                Ok(false)
            }
            GuiRuledSurfaceAction::SetPickTarget(target) => {
                self.create_features.ruled_surface.pick_target = target;
                Ok(false)
            }
            GuiRuledSurfaceAction::Clear => {
                let had_state = self.create_features.ruled_surface.active
                    | self.create_features.ruled_surface.preview.take().is_some()
                    | self.create_features.ruled_surface.pick.take().is_some();
                self.create_features.ruled_surface.active = false;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiRuledSurfaceAction::Preview(request) => {
                let request = *request;
                self.require_ruled_surface_generation(&request)?;
                self.create_features.ruled_surface.active = true;
                self.create_features.ruled_surface.automatic_edge_chain =
                    request.automatic_edge_chain;
                let (result, arrow) = self.preview_ruled_surface(&request);
                self.create_features.ruled_surface.preview = Some(RuledSurfacePreview {
                    request,
                    result,
                    arrow,
                });
                Ok(true)
            }
            GuiRuledSurfaceAction::Commit(request) => {
                let request = *request;
                self.require_ruled_surface_generation(&request)?;
                let commit = self
                    .create_features
                    .ruled_surface
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        RuledSurfaceOutcome::Ready(commit) => Some(commit),
                        RuledSurfaceOutcome::Incomplete => None,
                    })
                    .ok_or("ruled surface: expected a current certified preview")?
                    .clone();
                self.commit_ruled_surface(request.document, commit)?;
                self.create_features.ruled_surface.preview = None;
                self.create_features.ruled_surface.pick = None;
                self.create_features.ruled_surface.active = false;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_ruled_surface_generation(
        &self,
        request: &GuiRuledSurfaceRequest,
    ) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("ruled surface: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_ruled_surface(
        &mut self,
        request: &GuiRuledSurfaceRequest,
    ) -> (
        Result<RuledSurfaceOutcome, String>,
        Option<GuiRuledSurfaceArrow>,
    ) {
        let (Some(target_body), Some(distance), Some(angle_deg)) = (
            request.target_body.clone(),
            request.distance,
            request.angle_deg,
        ) else {
            self.clear_extrusion_preview();
            return (Ok(RuledSurfaceOutcome::Incomplete), None);
        };
        if request.edges.is_empty()
            || (request.ruled_type == RuledSurfaceType::Sweep && request.vector.is_none())
        {
            self.clear_extrusion_preview();
            return (Ok(RuledSurfaceOutcome::Incomplete), None);
        }
        let inputs = seed::request_to_inputs(request, target_body, distance, angle_deg);
        let commit = match request.feature {
            None => {
                let snapshot = match self.files.workspace().session(request.document) {
                    Ok(session) => session.snapshot(),
                    Err(cause) => return (Err(cause.to_string()), None),
                };
                match prepare_ruled_surface_definition(snapshot, &inputs) {
                    Ok(prepared) => RuledSurfaceCommit::Create(Box::new(prepared)),
                    Err(cause) => return (Err(cause.to_string()), None),
                }
            }
            Some(id) => match build_ruled_surface_definition(&inputs) {
                Ok(edited) => RuledSurfaceCommit::Edit(id, Box::new(edited)),
                Err(cause) => return (Err(cause.to_string()), None),
            },
        };
        let definition = match &commit {
            RuledSurfaceCommit::Create(prepared) => prepared.definition(),
            RuledSurfaceCommit::Edit(_, def) => def.as_ref(),
        };
        let arrow = self.compute_ruled_surface_arrow(request.document, definition);
        match self.evaluate_ruled_surface_preview_scene(request, &commit) {
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
                (Ok(RuledSurfaceOutcome::Ready(commit)), arrow)
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                (Err(cause), None)
            }
        }
    }

    fn compute_ruled_surface_arrow(
        &self,
        document: u64,
        definition: &RuledSurfaceDefinition,
    ) -> Option<GuiRuledSurfaceArrow> {
        let workspace = self.files.workspace();
        let session = workspace.session(document).ok()?;
        let guid = workspace.identity(document).ok()?.internal_name;
        let evaluation = self.part_scenes.current(session).ok()?;
        let body = evaluation.body_from_source(definition.body_source()).ok()?;
        let (vec_dir, vec_edge) = match definition.vector() {
            None => (None, None),
            Some(RuledSurfaceVector::OriginX) => (Some([1.0, 0.0, 0.0]), None),
            Some(RuledSurfaceVector::OriginY) => (Some([0.0, 1.0, 0.0]), None),
            Some(RuledSurfaceVector::OriginZ) => (Some([0.0, 0.0, 1.0]), None),
            Some(RuledSurfaceVector::WorkAxis(axis_id)) => {
                let axis = evaluation.datum_evaluation().axis(*axis_id).ok()?;
                (Some(axis.direction()), None)
            }
            Some(RuledSurfaceVector::Edge { edge, .. }) => (None, Some(edge.clone())),
        };
        let resolved = definition.resolve_with_direction(vec_dir, vec_edge);
        let arrow = body
            .ruled_surface_preview_arrow(session.history(), guid, &resolved)
            .ok()??;
        Some(GuiRuledSurfaceArrow {
            origin_world: arrow.origin,
            tip_world: arrow.tip,
            direction: arrow.direction,
            origin_pixel: None,
            tip_pixel: None,
            flipped: definition.flip(),
        })
    }

    fn evaluate_ruled_surface_preview_scene(
        &mut self,
        request: &GuiRuledSurfaceRequest,
        commit: &RuledSurfaceCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            RuledSurfaceCommit::Create(prepared) => {
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
            RuledSurfaceCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_ruled_surface(candidate, id, edited.clone()),
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

    fn commit_ruled_surface(
        &mut self,
        document: u64,
        commit: RuledSurfaceCommit,
    ) -> Result<(), String> {
        match commit {
            RuledSurfaceCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Ruled Surface",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            RuledSurfaceCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Ruled Surface",
                    |snapshot, _| replace_ruled_surface(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
