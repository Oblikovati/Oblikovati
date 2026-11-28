// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, planar face picking, live red cut-volume preview, and commit for Part Modify Slot.
#[path = "owner_slot_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedSlot, build_slot_definition, prepare_slot_definition, replace_slot,
};
use oblikovati_gui::{
    GuiHoleTermination, GuiSlotAction, GuiSlotEditSeed, GuiSlotPick, GuiSlotPreview,
    GuiSlotPreviewState, GuiSlotRequest,
};
use oblikovati_model::{
    feature::{FeatureId, slot::SlotDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

const CUT_PREVIEW_RED: [f32; 4] = [0.88, 0.20, 0.18, 1.0];

#[derive(Clone)]
enum SlotCommit {
    Create(Box<PreparedSlot>),
    Edit(FeatureId, Box<SlotDefinition>),
}

enum SlotOutcome {
    Ready(SlotCommit),
    Incomplete,
    UnchangedCut(String),
}

pub(in crate::owner) struct SlotPreview {
    request: GuiSlotRequest,
    result: Result<SlotOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct SlotOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) pick: Option<GuiSlotPick>,
    pub(in crate::owner) preview: Option<SlotPreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiSlotEditSeed> {
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
    pub(in crate::owner) fn gui_slot_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiSlotPreviewState> {
        let preview = self.create_features.slot.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiSlotPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(SlotOutcome::Ready(_)) => GuiSlotPreview::Ready,
                    Ok(SlotOutcome::Incomplete) => GuiSlotPreview::Incomplete,
                    Ok(SlotOutcome::UnchangedCut(cause)) => {
                        GuiSlotPreview::UnchangedCut(cause.clone())
                    }
                    Err(cause) => GuiSlotPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_slot_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiSlotPick> {
        self.create_features
            .slot
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn pick_slot_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.slot.active {
            return Ok(None);
        }
        let editing = self
            .create_features
            .slot
            .preview
            .as_ref()
            .and_then(|preview| preview.request.feature)
            .is_some();
        if editing {
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
        let body = self.resolve_slot_body_for_face(document, &key)?;
        self.create_features.slot.pick = Some(GuiSlotPick {
            document,
            generation,
            body,
            face: key,
            position: None,
        });
        Ok(Some(true))
    }

    pub(in crate::owner) fn slot_command(&mut self, action: GuiSlotAction) -> Result<bool, String> {
        match action {
            GuiSlotAction::Open | GuiSlotAction::OpenEdit(_) => {
                self.create_features.slot.active = true;
                Ok(false)
            }
            GuiSlotAction::Clear => {
                let had_state = self.create_features.slot.active
                    | self.create_features.slot.preview.take().is_some()
                    | self.create_features.slot.pick.take().is_some();
                self.create_features.slot.active = false;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiSlotAction::Preview(request) => {
                let request = *request;
                self.require_slot_generation(&request)?;
                self.create_features.slot.active = true;
                let result = self.preview_slot(&request);
                self.create_features.slot.preview = Some(SlotPreview { request, result });
                Ok(true)
            }
            GuiSlotAction::Commit(request) => {
                let request = *request;
                self.require_slot_generation(&request)?;
                let commit = self
                    .create_features
                    .slot
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        SlotOutcome::Ready(commit) => Some(commit),
                        SlotOutcome::Incomplete | SlotOutcome::UnchangedCut(_) => None,
                    })
                    .ok_or("slot: expected a current certified preview")?
                    .clone();
                self.commit_slot(request.document, commit)?;
                self.create_features.slot.preview = None;
                self.create_features.slot.pick = None;
                self.create_features.slot.active = false;
                self.model_face = None;
                self.face_appearance_picks.clear();
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_slot_generation(&self, request: &GuiSlotRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("slot: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_slot(&mut self, request: &GuiSlotRequest) -> Result<SlotOutcome, String> {
        let Some(face) = request.face.clone() else {
            self.clear_extrusion_preview();
            return Ok(SlotOutcome::Incomplete);
        };
        if request.termination == GuiHoleTermination::ToSurface
            && request.termination_surface.is_none()
        {
            self.clear_extrusion_preview();
            return Ok(SlotOutcome::Incomplete);
        }
        let body = match request.body.clone() {
            Some(body) => body,
            None => self.resolve_slot_body_for_face(request.document, &face)?,
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
                let prepared = prepare_slot_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                let def = prepared.definition().clone();
                (def, SlotCommit::Create(Box::new(prepared)))
            }
            Some(id) => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let edited =
                    build_slot_definition(snapshot, &inputs).map_err(|cause| cause.to_string())?;
                (edited.clone(), SlotCommit::Edit(id, Box::new(edited)))
            }
        };
        let red_triangles = self
            .build_red_slot_tool_triangles(request.document, &body, &face, &definition)
            .unwrap_or_default();
        if request.preview_hidden {
            self.clear_extrusion_preview();
            return Ok(SlotOutcome::Ready(commit));
        }
        match self.evaluate_slot_preview_scene(request, &commit) {
            Ok(mut scene) => {
                scene.triangles.extend(red_triangles);
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Replacement,
                ));
                self.part_scenes.invalidate_presentation();
                Ok(SlotOutcome::Ready(commit))
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
                        return Ok(SlotOutcome::UnchangedCut(cause));
                    }
                } else {
                    self.clear_extrusion_preview();
                }
                Err(cause)
            }
        }
    }

    fn evaluate_slot_preview_scene(
        &mut self,
        request: &GuiSlotRequest,
        commit: &SlotCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            SlotCommit::Create(prepared) => {
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
            SlotCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_slot(candidate, id, edited.clone()),
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

    fn build_red_slot_tool_triangles(
        &self,
        document: u64,
        body_key: &[u8],
        face_key: &[u8],
        definition: &SlotDefinition,
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
            let tris = body.slot_preview_triangles(
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
        Err("slot preview: source body not found".into())
    }

    fn resolve_slot_body_for_face(&self, document: u64, face: &[u8]) -> Result<Vec<u8>, String> {
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
        Err("slot: expected a current solid face".into())
    }

    fn commit_slot(&mut self, document: u64, commit: SlotCommit) -> Result<(), String> {
        match commit {
            SlotCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Slot",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            SlotCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Slot",
                    |snapshot, _| replace_slot(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
