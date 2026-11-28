// SPDX-License-Identifier: GPL-2.0-only
//! Surface picking, B-rep sewing preview, edge-condition analysis, and Apply/Done for Stitch (#3918).
#[path = "owner_stitch_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedStitch, build_stitch_definition, prepare_stitch_definition, replace_stitch,
};
use oblikovati_gui::{
    GuiStitchAction, GuiStitchEditSeed, GuiStitchFreeEdge, GuiStitchPick, GuiStitchPreview,
    GuiStitchPreviewState, GuiStitchRequest,
};
use oblikovati_martensite::analyze_stitch_surfaces;
use oblikovati_model::{
    feature::{FeatureId, stitch::StitchDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum StitchCommit {
    Create(Box<PreparedStitch>),
    Edit(FeatureId, Box<StitchDefinition>),
}

enum StitchOutcome {
    Ready {
        commit: StitchCommit,
        stitched_edge_count: usize,
        free_edges: Vec<GuiStitchFreeEdge>,
        near_tangent_count: usize,
    },
    Incomplete,
}

pub(in crate::owner) struct StitchPreview {
    request: GuiStitchRequest,
    result: Result<StitchOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct StitchOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiStitchPick>,
    pub(in crate::owner) preview: Option<StitchPreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiStitchEditSeed> {
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
    pub(in crate::owner) fn gui_stitch_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiStitchPreviewState> {
        let preview = self.create_features.stitch.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiStitchPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(StitchOutcome::Ready {
                        stitched_edge_count,
                        free_edges,
                        near_tangent_count,
                        ..
                    }) => GuiStitchPreview::Ready {
                        stitched_edge_count: *stitched_edge_count,
                        free_edges: free_edges.clone(),
                        near_tangent_count: *near_tangent_count,
                    },
                    Ok(StitchOutcome::Incomplete) => GuiStitchPreview::Incomplete,
                    Err(cause) => GuiStitchPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_stitch_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiStitchPick> {
        self.create_features
            .stitch
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn pick_stitch_browser_body(
        &mut self,
        document: u64,
        hex_key: &str,
    ) -> Result<bool, String> {
        if !self.create_features.stitch.active {
            return Ok(false);
        }
        let rows = self.part_scenes.body_rows(&self.files, document)?;
        let Some(row) = rows.iter().find(|r| r.surface && r.key == hex_key) else {
            return Ok(false);
        };
        let Some(surface_key) = seed::decode_hex_key(hex_key) else {
            return Ok(false);
        };
        let generation = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .generation();
        self.record_stitch_pick(document, generation, surface_key, row.name.clone());
        Ok(true)
    }

    pub(in crate::owner) fn pick_stitch_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.stitch.active {
            return Ok(None);
        }
        let allowed = oblikovati_session::selection::SelectionFilter::only([
            oblikovati_session::selection::SelectionKind::Face,
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
        let feature_id = match picked.target {
            crate::part_scene::pick_candidates::PickTarget::Face(feature, _)
            | crate::part_scene::pick_candidates::PickTarget::Body(feature, _) => feature,
            _ => return Ok(None),
        };
        let Some((generation, surface_key, surface_name)) =
            self.resolve_surface_body_for_feature(document, feature_id)?
        else {
            return Ok(Some(false));
        };
        self.record_stitch_pick(document, generation, surface_key, surface_name);
        Ok(Some(true))
    }

    fn record_stitch_pick(
        &mut self,
        document: u64,
        generation: u64,
        surface_key: Vec<u8>,
        surface_name: String,
    ) {
        self.create_features.stitch.pick_seq =
            self.create_features.stitch.pick_seq.saturating_add(1);
        self.create_features.stitch.pick = Some(GuiStitchPick {
            document,
            generation,
            sequence: self.create_features.stitch.pick_seq,
            surface_key,
            surface_name,
        });
    }

    fn resolve_surface_body_for_feature(
        &self,
        document: u64,
        feature: FeatureId,
    ) -> Result<Option<(u64, Vec<u8>, String)>, String> {
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
        let Some(body) = evaluation.body(feature) else {
            return Ok(None);
        };
        if !body.is_surface() {
            return Ok(None);
        }
        let key = body.body_reference_key(session.history(), guid)?.to_vec();
        let name = self
            .part_scenes
            .body_rows(&self.files, document)?
            .into_iter()
            .find(|r| r.surface && r.feature == feature)
            .map_or_else(|| "Surface".to_string(), |r| r.name);
        Ok(Some((generation, key, name)))
    }

    pub(in crate::owner) fn stitch_command(
        &mut self,
        action: GuiStitchAction,
    ) -> Result<bool, String> {
        match action {
            GuiStitchAction::Open | GuiStitchAction::OpenEdit(_) => {
                self.create_features.stitch.active = true;
                Ok(false)
            }
            GuiStitchAction::Clear => {
                let had = self.create_features.stitch.preview.take().is_some();
                Ok(had | self.clear_extrusion_preview())
            }
            GuiStitchAction::Done => {
                let had = self.create_features.stitch.active
                    | self.create_features.stitch.preview.take().is_some()
                    | self.create_features.stitch.pick.take().is_some();
                self.create_features.stitch.active = false;
                Ok(had | self.clear_extrusion_preview())
            }
            GuiStitchAction::Preview(request) => {
                let request = *request;
                self.require_stitch_generation(&request)?;
                self.create_features.stitch.active = true;
                let result = self.preview_stitch(&request);
                self.create_features.stitch.preview = Some(StitchPreview { request, result });
                Ok(true)
            }
            GuiStitchAction::Apply(request) => {
                let request = *request;
                self.require_stitch_generation(&request)?;
                let commit = self
                    .create_features
                    .stitch
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        StitchOutcome::Ready { commit, .. } => Some(commit),
                        StitchOutcome::Incomplete => None,
                    })
                    .ok_or("stitch: expected a current certified preview")?
                    .clone();
                self.commit_stitch(request.document, commit)?;
                // Apply keeps the Stitch dialog open and active until Done is clicked.
                self.create_features.stitch.preview = None;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_stitch_generation(&self, request: &GuiStitchRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("stitch: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_stitch(&mut self, request: &GuiStitchRequest) -> Result<StitchOutcome, String> {
        let Some(tolerance_mm) = request.maximum_tolerance_mm else {
            self.clear_extrusion_preview();
            return Ok(StitchOutcome::Incomplete);
        };
        if request.surfaces.len() < 2 {
            self.clear_extrusion_preview();
            return Ok(StitchOutcome::Incomplete);
        }
        let inputs = seed::request_to_inputs(request, tolerance_mm);
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_stitch_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                StitchCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited = build_stitch_definition(&inputs).map_err(|cause| cause.to_string())?;
                StitchCommit::Edit(id, Box::new(edited))
            }
        };
        let report = self.analyze_selected_surfaces(request, tolerance_mm * 0.1)?;
        match self.evaluate_stitch_preview_scene(request, &commit) {
            Ok(scene) => {
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Additive,
                ));
                self.part_scenes.invalidate_presentation();
                Ok(StitchOutcome::Ready {
                    commit,
                    stitched_edge_count: report.stitched_edge_count,
                    free_edges: seed::free_edges_to_gui(&report),
                    near_tangent_count: report.near_tangent_edges.len(),
                })
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn analyze_selected_surfaces(
        &self,
        request: &GuiStitchRequest,
        tolerance_cm: f64,
    ) -> Result<oblikovati_martensite::StitchAnalysisReport, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(request.document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let mut selected = Vec::with_capacity(request.surfaces.len());
        for key in &request.surfaces {
            let mut matched = None;
            for body in evaluation.ordered_bodies() {
                if let Ok(body_key) = body.body_reference_key(session.history(), guid)
                    && body_key == key.as_slice()
                {
                    matched = Some(body);
                    break;
                }
            }
            let body = matched.ok_or_else(|| {
                format!("stitch surface key {key:?}: expected a live surface body")
            })?;
            selected.push(body);
        }
        analyze_stitch_surfaces(&selected, session.history(), guid, tolerance_cm)
    }

    fn evaluate_stitch_preview_scene(
        &mut self,
        request: &GuiStitchRequest,
        commit: &StitchCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            StitchCommit::Create(prepared) => {
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
            StitchCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_stitch(candidate, id, edited.clone()),
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

    fn commit_stitch(&mut self, document: u64, commit: StitchCommit) -> Result<(), String> {
        match commit {
            StitchCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Stitch Surface",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            StitchCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Stitch Surface",
                    |snapshot, _| replace_stitch(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
