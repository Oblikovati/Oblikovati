// SPDX-License-Identifier: GPL-2.0-only
//! Selection-mode viewport picking, partial-blend probing, live preview, and commit for Fillet v2.
#[path = "owner_fillet_v2_pick.rs"]
mod pick;
#[path = "owner_fillet_v2_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedFillet, build_fillet_v2_definition, prepare_fillet_v2, replace_fillet,
};
use oblikovati_gui::{
    GuiFilletEdgeSet, GuiFilletSelectionMode, GuiFilletSharedVertex, GuiFilletV2Action,
    GuiFilletV2EditSeed, GuiFilletV2Pick, GuiFilletV2Preview, GuiFilletV2PreviewState,
    GuiFilletV2Request,
};
use oblikovati_model::{
    feature::{FeatureId, fillet::FilletDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum FilletV2Commit {
    Create(Box<PreparedFillet>),
    Edit(FeatureId, Box<FilletDefinition>),
}

enum FilletV2Outcome {
    Ready(FilletV2Commit),
    PartialFailure {
        failed_count: usize,
        total_count: usize,
        kept_sets: Vec<GuiFilletEdgeSet>,
    },
    Incomplete,
}

pub(in crate::owner) struct FilletV2Preview {
    request: GuiFilletV2Request,
    result: Result<FilletV2Outcome, String>,
    available_vertices: Vec<GuiFilletSharedVertex>,
}

#[derive(Default)]
pub(in crate::owner) struct FilletV2OwnerState {
    pub(in crate::owner) active_mode: Option<GuiFilletSelectionMode>,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiFilletV2Pick>,
    pub(in crate::owner) preview: Option<FilletV2Preview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
    generation: u64,
) -> BTreeMap<FeatureId, GuiFilletV2EditSeed> {
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
    #[cfg(test)]
    pub(crate) fn part_scenes_current(
        &self,
        document: u64,
    ) -> Result<
        &oblikovati_model::feature::part::PartFeatureEvaluation<
            oblikovati_martensite::NamedSolid,
            oblikovati_martensite::EvaluationFailure,
        >,
        String,
    > {
        let session = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?;
        self.part_scenes.current(session)
    }

    pub(in crate::owner) fn gui_fillet_v2_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiFilletV2PreviewState> {
        let preview = self.create_features.fillet_v2.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiFilletV2PreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(FilletV2Outcome::Ready(_)) => GuiFilletV2Preview::Ready,
                    Ok(FilletV2Outcome::Incomplete) => GuiFilletV2Preview::Incomplete,
                    Ok(FilletV2Outcome::PartialFailure {
                        failed_count,
                        total_count,
                        kept_sets,
                    }) => GuiFilletV2Preview::PartialFailure {
                        failed_count: *failed_count,
                        total_count: *total_count,
                        kept_sets: kept_sets.clone(),
                    },
                    Err(cause) => GuiFilletV2Preview::Failed(cause.clone()),
                },
                available_vertices: preview.available_vertices.clone(),
            },
        )
    }

    pub(in crate::owner) fn gui_fillet_v2_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiFilletV2Pick> {
        self.create_features
            .fillet_v2
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn fillet_v2_command(
        &mut self,
        action: GuiFilletV2Action,
    ) -> Result<bool, String> {
        match action {
            GuiFilletV2Action::SetSelectionMode(mode) => {
                self.create_features.fillet_v2.active_mode = Some(mode);
                Ok(true)
            }
            GuiFilletV2Action::Clear => {
                let had = self.create_features.fillet_v2.active_mode.take().is_some()
                    | self.create_features.fillet_v2.preview.take().is_some()
                    | self.create_features.fillet_v2.pick.take().is_some();
                Ok(had | self.clear_extrusion_preview())
            }
            GuiFilletV2Action::Preview(request) => {
                let request = *request;
                self.require_fillet_v2_generation(&request)?;
                let available_vertices = self
                    .shared_vertices_for_request(&request)
                    .unwrap_or_default();
                let result = self.preview_fillet_v2(&request);
                self.create_features.fillet_v2.preview = Some(FilletV2Preview {
                    request,
                    result,
                    available_vertices,
                });
                Ok(true)
            }
            GuiFilletV2Action::Commit(request) => {
                let request = *request;
                self.require_fillet_v2_generation(&request)?;
                let commit = self
                    .create_features
                    .fillet_v2
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        FilletV2Outcome::Ready(commit) => Some(commit),
                        _ => None,
                    })
                    .ok_or("fillet: expected a current certified preview")?
                    .clone();
                self.commit_fillet_v2(request.document, commit)?;
                self.create_features.fillet_v2.preview = None;
                self.create_features.fillet_v2.pick = None;
                self.create_features.fillet_v2.active_mode = None;
                self.model_edges.clear();
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_fillet_v2_generation(&self, request: &GuiFilletV2Request) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("fillet: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_fillet_v2(
        &mut self,
        request: &GuiFilletV2Request,
    ) -> Result<FilletV2Outcome, String> {
        let total_edges: usize = request.edge_sets.iter().map(|s| s.edges.len()).sum();
        if total_edges == 0 {
            self.clear_extrusion_preview();
            return Ok(FilletV2Outcome::Incomplete);
        }
        let body = match request.body.clone() {
            Some(body) if !body.is_empty() => body,
            _ => self.resolve_body_for_edge(request.document, &request.edge_sets[0].edges[0])?,
        };
        let inputs = seed::request_to_inputs(request, body.clone())?;
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared =
                    prepare_fillet_v2(snapshot, &inputs).map_err(|cause| cause.to_string())?;
                FilletV2Commit::Create(Box::new(prepared))
            }
            Some(id) => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let edited = build_fillet_v2_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                FilletV2Commit::Edit(id, Box::new(edited))
            }
        };
        if request.solid_selected
            && let Some(partial) = self.check_partial_solid_blend(request, &body)?
        {
            return Ok(partial);
        }
        match self.evaluate_fillet_v2_preview_scene(request, &commit) {
            Ok(scene) => {
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Replacement,
                ));
                self.part_scenes.invalidate_presentation();
                Ok(FilletV2Outcome::Ready(commit))
            }
            Err(cause) => {
                if request.solid_selected
                    && let Some(partial) = self.check_partial_solid_blend(request, &body)?
                {
                    return Ok(partial);
                }
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn check_partial_solid_blend(
        &mut self,
        request: &GuiFilletV2Request,
        body_key: &[u8],
    ) -> Result<Option<FilletV2Outcome>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(request.document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let snapshot = session.snapshot();
        let evaluation = self.part_scenes.current(session)?;
        let Some(body) = evaluation.ordered_bodies().into_iter().find(|b| {
            b.body_reference_key(session.history(), guid)
                .is_ok_and(|k| k == body_key)
        }) else {
            return Ok(None);
        };
        let mut raw_sets = Vec::new();
        let mut radius_strs = Vec::new();
        for set in &request.edge_sets {
            let len =
                oblikovati_automation::evaluate_positive_cm(snapshot, &set.radius).unwrap_or(0.2);
            raw_sets.push((set.edges.clone(), len));
            radius_strs.push(set.radius.clone());
        }
        let (kept, failed_count) =
            body.partition_blendable_sets(session.history(), guid, &raw_sets);
        if failed_count == 0 || kept.is_empty() {
            return Ok(None);
        }
        let total_count: usize = request.edge_sets.iter().map(|s| s.edges.len()).sum();
        let kept_sets: Vec<GuiFilletEdgeSet> = kept
            .into_iter()
            .enumerate()
            .map(|(idx, (edges, _))| GuiFilletEdgeSet {
                edges,
                radius: radius_strs
                    .get(idx)
                    .cloned()
                    .unwrap_or_else(|| "2 mm".into()),
                is_loop: false,
            })
            .collect();
        // Highlight preview with the kept blendable edges so the body remains visibly highlighted.
        let mut kept_req = request.clone();
        kept_req.edge_sets = kept_sets.clone();
        kept_req.solid_selected = false;
        if let Ok(inputs) = seed::request_to_inputs(&kept_req, body_key.to_vec())
            && let Ok(prepared) = prepare_fillet_v2(snapshot, &inputs)
            && let Ok(scene) = self.evaluate_fillet_v2_preview_scene(
                &kept_req,
                &FilletV2Commit::Create(Box::new(prepared)),
            )
        {
            self.extrusion_preview = Some((
                request.document,
                request.generation,
                scene,
                super::super::owner_extrusion_preview::PreviewKind::Replacement,
            ));
            self.part_scenes.invalidate_presentation();
        }
        Ok(Some(FilletV2Outcome::PartialFailure {
            failed_count,
            total_count,
            kept_sets,
        }))
    }

    fn evaluate_fillet_v2_preview_scene(
        &mut self,
        request: &GuiFilletV2Request,
        commit: &FilletV2Commit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            FilletV2Commit::Create(prepared) => {
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
            FilletV2Commit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_fillet(candidate, id, edited.clone()),
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

    fn commit_fillet_v2(&mut self, document: u64, commit: FilletV2Commit) -> Result<(), String> {
        match commit {
            FilletV2Commit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Fillet",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            FilletV2Commit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Fillet",
                    |snapshot, _| replace_fillet(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
