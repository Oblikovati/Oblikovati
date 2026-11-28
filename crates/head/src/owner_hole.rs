// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, live red cut-volume preview, and commit for Part Modify Hole (#4093).
#[path = "owner_hole_pick.rs"]
mod pick;
#[path = "owner_hole_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedHole, build_hole_definition, prepare_hole_definition, replace_hole,
};
use oblikovati_gui::{
    GuiHoleAction, GuiHoleEditSeed, GuiHolePick, GuiHolePreview, GuiHolePreviewState,
    GuiHoleRequest, GuiHoleTermination,
};
use oblikovati_model::{
    feature::{FeatureId, hole::HoleDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

const CUT_PREVIEW_RED: [f32; 4] = [0.88, 0.20, 0.18, 1.0];

#[derive(Clone)]
enum HoleCommit {
    Create(Box<PreparedHole>),
    Edit(FeatureId, Box<HoleDefinition>),
}

enum HoleOutcome {
    Ready(HoleCommit),
    Incomplete,
    UnchangedCut(String),
}

pub(in crate::owner) struct HolePreview {
    request: GuiHoleRequest,
    result: Result<HoleOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct HoleOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) pick: Option<GuiHolePick>,
    pub(in crate::owner) preview: Option<HolePreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiHoleEditSeed> {
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
    pub(in crate::owner) fn gui_hole_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiHolePreviewState> {
        let preview = self.create_features.hole.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiHolePreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(HoleOutcome::Ready(_)) => GuiHolePreview::Ready,
                    Ok(HoleOutcome::Incomplete) => GuiHolePreview::Incomplete,
                    Ok(HoleOutcome::UnchangedCut(cause)) => {
                        GuiHolePreview::UnchangedCut(cause.clone())
                    }
                    Err(cause) => GuiHolePreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_hole_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiHolePick> {
        self.create_features
            .hole
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn hole_command(&mut self, action: GuiHoleAction) -> Result<bool, String> {
        match action {
            GuiHoleAction::Open | GuiHoleAction::OpenEdit(_) => {
                self.create_features.hole.active = true;
                Ok(false)
            }
            GuiHoleAction::Clear => {
                let had_state = self.create_features.hole.active
                    | self.create_features.hole.preview.take().is_some()
                    | self.create_features.hole.pick.take().is_some();
                self.create_features.hole.active = false;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiHoleAction::Preview(request) => {
                let request = *request;
                self.require_hole_generation(&request)?;
                self.create_features.hole.active = true;
                let result = self.preview_hole(&request);
                self.create_features.hole.preview = Some(HolePreview { request, result });
                Ok(true)
            }
            GuiHoleAction::Commit(request) => {
                let request = *request;
                self.require_hole_generation(&request)?;
                let commit = self
                    .create_features
                    .hole
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        HoleOutcome::Ready(commit) => Some(commit),
                        HoleOutcome::Incomplete | HoleOutcome::UnchangedCut(_) => None,
                    })
                    .ok_or("hole: expected a current certified preview")?
                    .clone();
                self.commit_hole(request.document, commit)?;
                self.create_features.hole.preview = None;
                self.create_features.hole.pick = None;
                self.create_features.hole.active = false;
                self.model_face = None;
                self.face_appearance_picks.clear();
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_hole_generation(&self, request: &GuiHoleRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("hole: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_hole(&mut self, request: &GuiHoleRequest) -> Result<HoleOutcome, String> {
        let Some(face) = request.face.clone() else {
            self.clear_extrusion_preview();
            return Ok(HoleOutcome::Incomplete);
        };
        if request.termination == GuiHoleTermination::ToSurface
            && request.termination_surface.is_none()
        {
            self.clear_extrusion_preview();
            return Ok(HoleOutcome::Incomplete);
        }
        let body = match request.body.clone() {
            Some(body) => body,
            None => self.resolve_hole_body_for_face(request.document, &face)?,
        };
        let inputs = seed::request_to_inputs(request, body.clone(), face.clone())?;
        let (definition, commit) = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_hole_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                let def = prepared.definition().clone();
                (def, HoleCommit::Create(Box::new(prepared)))
            }
            Some(id) => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let edited =
                    build_hole_definition(snapshot, &inputs).map_err(|cause| cause.to_string())?;
                (edited.clone(), HoleCommit::Edit(id, Box::new(edited)))
            }
        };
        let red_triangles = self
            .build_red_hole_tool_triangles(request.document, &body, &face, &definition)
            .unwrap_or_default();
        if request.preview_hidden {
            self.clear_extrusion_preview();
            return Ok(HoleOutcome::Ready(commit));
        }
        let eval_result = self.evaluate_hole_preview_scene(request, &commit);
        match eval_result {
            Ok(mut scene) => {
                scene.triangles.extend(red_triangles);
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Replacement,
                ));
                self.part_scenes.invalidate_presentation();
                Ok(HoleOutcome::Ready(commit))
            }
            Err(cause) => {
                if !red_triangles.is_empty() {
                    let scene = oblikovati_viewport::ViewportScene {
                        background: [0.0, 0.0, 0.0],
                        triangles: red_triangles,
                    };
                    self.extrusion_preview = Some((
                        request.document,
                        request.generation,
                        scene,
                        super::super::owner_extrusion_preview::PreviewKind::Additive,
                    ));
                    self.part_scenes.invalidate_presentation();
                    if request.direction_flipped {
                        return Ok(HoleOutcome::UnchangedCut(cause));
                    }
                } else {
                    self.clear_extrusion_preview();
                }
                Err(cause)
            }
        }
    }

    fn evaluate_hole_preview_scene(
        &mut self,
        request: &GuiHoleRequest,
        commit: &HoleCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            HoleCommit::Create(prepared) => {
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
            HoleCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_hole(candidate, id, edited.clone()),
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

    fn build_red_hole_tool_triangles(
        &self,
        document: u64,
        body_key: &[u8],
        face_key: &[u8],
        definition: &HoleDefinition,
    ) -> Result<Vec<oblikovati_viewport::ViewportTriangle>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let resolved = definition
            .resolve(&session.snapshot().parameters)
            .map_err(|cause| cause.to_string())?;
        let evaluation = self.part_scenes.current(session)?;
        for body in evaluation.ordered_bodies() {
            let Ok(key) = body.body_reference_key(session.history(), guid) else {
                continue;
            };
            if key != body_key {
                continue;
            }
            let tris = body.hole_preview_triangles(
                session.history(),
                guid,
                face_key,
                &resolved,
                crate::part_scene::DISPLAY_MESH,
            )?;
            return Ok(tris
                .into_iter()
                .map(|tri| oblikovati_viewport::ViewportTriangle {
                    positions: tri.positions.map(|p| p.map(|c| c as f32)),
                    normals: tri.normals.map(|ns| ns.map(|n| n.map(|c| c as f32))),
                    color: CUT_PREVIEW_RED,
                    face_index: None,
                })
                .collect());
        }
        Err("hole preview: source body not found".into())
    }

    fn resolve_hole_body_for_face(&self, document: u64, face: &[u8]) -> Result<Vec<u8>, String> {
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
        Err("hole: expected a current solid face".into())
    }

    fn commit_hole(&mut self, document: u64, commit: HoleCommit) -> Result<(), String> {
        match commit {
            HoleCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Hole",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            HoleCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Hole",
                    |snapshot, _| replace_hole(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
