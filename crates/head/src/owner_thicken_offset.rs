// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, face picking (with tangent face chaining), live preview, and commit for Part Modify Thicken/Offset (#3918).
#[path = "owner_thicken_offset_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedThickenOffset, build_thicken_offset_definition, prepare_thicken_offset_definition,
    replace_thicken_offset,
};
use oblikovati_gui::{
    GuiThickenOffsetAction, GuiThickenOffsetEditSeed, GuiThickenOffsetPick,
    GuiThickenOffsetPreview, GuiThickenOffsetPreviewState, GuiThickenOffsetRequest,
};
use oblikovati_model::{
    feature::{
        FeatureId,
        thicken_offset::{ThickenOffsetDefinition, ThickenOutputMode},
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum ThickenOffsetCommit {
    Create(Box<PreparedThickenOffset>),
    Edit(FeatureId, Box<ThickenOffsetDefinition>),
}

enum ThickenOffsetOutcome {
    Ready(ThickenOffsetCommit),
    Incomplete,
}

pub(in crate::owner) struct ThickenOffsetPreview {
    request: GuiThickenOffsetRequest,
    result: Result<ThickenOffsetOutcome, String>,
}

struct ResolvedFacePick {
    generation: u64,
    target_body: Vec<u8>,
    faces: Vec<Vec<u8>>,
}

#[derive(Default)]
pub(in crate::owner) struct ThickenOffsetOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) automatic_face_chain: bool,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiThickenOffsetPick>,
    pub(in crate::owner) preview: Option<ThickenOffsetPreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiThickenOffsetEditSeed> {
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
    pub(in crate::owner) fn gui_thicken_offset_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiThickenOffsetPreviewState> {
        let preview = self.create_features.thicken_offset.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiThickenOffsetPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(ThickenOffsetOutcome::Ready(_)) => GuiThickenOffsetPreview::Ready,
                    Ok(ThickenOffsetOutcome::Incomplete) => GuiThickenOffsetPreview::Incomplete,
                    Err(cause) => GuiThickenOffsetPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_thicken_offset_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiThickenOffsetPick> {
        self.create_features
            .thicken_offset
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn pick_thicken_offset_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        if !self.create_features.thicken_offset.active {
            return Ok(None);
        }
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
        let crate::part_scene::pick_candidates::PickTarget::Face(_, face_key) = picked.target
        else {
            return Ok(None);
        };
        let chain = self.create_features.thicken_offset.automatic_face_chain;
        let resolved = self.resolve_thicken_face_pick(document, &face_key, chain)?;
        self.create_features.thicken_offset.pick_seq = self
            .create_features
            .thicken_offset
            .pick_seq
            .saturating_add(1);
        self.create_features.thicken_offset.pick = Some(GuiThickenOffsetPick {
            document,
            generation: resolved.generation,
            sequence: self.create_features.thicken_offset.pick_seq,
            target_body: resolved.target_body,
            faces: resolved.faces,
        });
        Ok(Some(true))
    }

    fn resolve_thicken_face_pick(
        &self,
        document: u64,
        face_key: &[u8],
        automatic_face_chain: bool,
    ) -> Result<ResolvedFacePick, String> {
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
        for feature in session.snapshot().work_geometry.features() {
            let Some(body) = evaluation.body(feature.id()) else {
                continue;
            };
            let Ok(face_keys) = body.face_reference_keys(session.history(), guid) else {
                continue;
            };
            if !face_keys.iter().any(|k| k.as_slice() == face_key) {
                continue;
            }
            let target_body = body.body_reference_key(session.history(), guid)?.to_vec();
            let faces = if automatic_face_chain {
                body.tangent_face_chain(session.history(), guid, face_key)?
            } else {
                vec![face_key.to_vec()]
            };
            return Ok(ResolvedFacePick {
                generation,
                target_body,
                faces,
            });
        }
        Err("thicken/offset pick: expected a live named face".into())
    }

    pub(in crate::owner) fn thicken_offset_command(
        &mut self,
        action: GuiThickenOffsetAction,
    ) -> Result<bool, String> {
        match action {
            GuiThickenOffsetAction::Open | GuiThickenOffsetAction::OpenEdit(_) => {
                self.create_features.thicken_offset.active = true;
                Ok(false)
            }
            GuiThickenOffsetAction::SetAutomaticFaceChain(enabled) => {
                self.create_features.thicken_offset.automatic_face_chain = enabled;
                Ok(false)
            }
            GuiThickenOffsetAction::Clear => {
                let had_state = self.create_features.thicken_offset.active
                    | self.create_features.thicken_offset.preview.take().is_some()
                    | self.create_features.thicken_offset.pick.take().is_some();
                self.create_features.thicken_offset.active = false;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiThickenOffsetAction::Preview(request) => {
                let request = *request;
                self.require_thicken_offset_generation(&request)?;
                self.create_features.thicken_offset.active = true;
                self.create_features.thicken_offset.automatic_face_chain =
                    request.automatic_face_chain;
                let result = self.preview_thicken_offset(&request);
                self.create_features.thicken_offset.preview =
                    Some(ThickenOffsetPreview { request, result });
                Ok(true)
            }
            GuiThickenOffsetAction::Commit(request) => {
                let request = *request;
                self.require_thicken_offset_generation(&request)?;
                let commit = self
                    .create_features
                    .thicken_offset
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        ThickenOffsetOutcome::Ready(commit) => Some(commit),
                        ThickenOffsetOutcome::Incomplete => None,
                    })
                    .ok_or("thicken/offset: expected a current certified preview")?
                    .clone();
                self.commit_thicken_offset(request.document, commit)?;
                self.create_features.thicken_offset.preview = None;
                self.create_features.thicken_offset.pick = None;
                self.create_features.thicken_offset.active = false;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_thicken_offset_generation(
        &self,
        request: &GuiThickenOffsetRequest,
    ) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("thicken/offset: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_thicken_offset(
        &mut self,
        request: &GuiThickenOffsetRequest,
    ) -> Result<ThickenOffsetOutcome, String> {
        let (Some(target_body), Some(distance)) = (request.target_body.clone(), request.distance)
        else {
            self.clear_extrusion_preview();
            return Ok(ThickenOffsetOutcome::Incomplete);
        };
        if request.faces.is_empty() || !request.approximation_valid {
            self.clear_extrusion_preview();
            return Ok(ThickenOffsetOutcome::Incomplete);
        }
        let inputs = seed::request_to_inputs(request, target_body, distance);
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_thicken_offset_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                ThickenOffsetCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited =
                    build_thicken_offset_definition(&inputs).map_err(|cause| cause.to_string())?;
                ThickenOffsetCommit::Edit(id, Box::new(edited))
            }
        };
        match self.evaluate_thicken_offset_preview_scene(request, &commit) {
            Ok(scene) => {
                if request.preview_hidden {
                    self.clear_extrusion_preview();
                } else {
                    let kind = if request.output == ThickenOutputMode::Surface {
                        super::super::owner_extrusion_preview::PreviewKind::Additive
                    } else {
                        super::super::owner_extrusion_preview::PreviewKind::Replacement
                    };
                    self.extrusion_preview =
                        Some((request.document, request.generation, scene, kind));
                    self.part_scenes.invalidate_presentation();
                }
                Ok(ThickenOffsetOutcome::Ready(commit))
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn evaluate_thicken_offset_preview_scene(
        &mut self,
        request: &GuiThickenOffsetRequest,
        commit: &ThickenOffsetCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            ThickenOffsetCommit::Create(prepared) => {
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
            ThickenOffsetCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_thicken_offset(candidate, id, edited.clone()),
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

    fn commit_thicken_offset(
        &mut self,
        document: u64,
        commit: ThickenOffsetCommit,
    ) -> Result<(), String> {
        match commit {
            ThickenOffsetCommit::Create(prepared) => {
                let label = if prepared.definition().output().is_surface() {
                    "Offset Surface"
                } else {
                    "Thicken"
                };
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    label,
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            ThickenOffsetCommit::Edit(id, definition) => {
                let label = if definition.output().is_surface() {
                    "Edit Offset Surface"
                } else {
                    "Edit Thicken"
                };
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    label,
                    |snapshot, _| replace_thicken_offset(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
