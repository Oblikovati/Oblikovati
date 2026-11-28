// SPDX-License-Identifier: GPL-2.0-only
//! Eligibility checks, preview, and transactional commit for Part Modify Mark (#3918).
#[path = "owner_mark_pick.rs"]
mod pick;

use super::*;
use oblikovati_automation::{
    MarkFeatureInputs, PreparedMark, build_mark_definition, prepare_mark_definition, replace_mark,
};
use oblikovati_gui::{
    GuiMarkAction, GuiMarkEditSeed, GuiMarkEligibility, GuiMarkFacePick, GuiMarkPreview,
    GuiMarkPreviewState, GuiMarkRequest, GuiMarkSketchPick,
};
use oblikovati_martensite::{MarkEligibility, check_mark_eligibility};
use oblikovati_model::{
    feature::{
        FeatureId,
        mark::{MarkDefinition, MarkGeometrySet, MarkMethod},
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum MarkCommit {
    Create(Box<PreparedMark>),
    Edit(FeatureId, Box<MarkDefinition>),
}

enum MarkOutcome {
    Ready(MarkCommit),
    Incomplete,
}

pub(in crate::owner) struct MarkPreview {
    pub(super) request: GuiMarkRequest,
    result: Result<MarkOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct MarkOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) face_pick: Option<GuiMarkFacePick>,
    pub(in crate::owner) sketch_pick: Option<GuiMarkSketchPick>,
    pub(in crate::owner) preview: Option<MarkPreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiMarkEditSeed> {
    pick::edit_seeds(snapshot)
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
    pub(in crate::owner) fn part_mark_eligibility(&self, document: u64) -> GuiMarkEligibility {
        let workspace = self.files.workspace();
        let Ok(session) = workspace.session(document) else {
            return GuiMarkEligibility::NoVolume;
        };
        let Ok(identity) = workspace.identity(document) else {
            return GuiMarkEligibility::NoVolume;
        };
        let Ok(evaluation) = self.part_scenes.current(session) else {
            return GuiMarkEligibility::NoVolume;
        };
        match check_mark_eligibility(
            session.history(),
            identity.internal_name,
            evaluation.ordered_bodies(),
        ) {
            MarkEligibility::Eligible => GuiMarkEligibility::Eligible,
            MarkEligibility::MultipleSolidBodies { .. } => GuiMarkEligibility::MultipleSolidBodies,
            MarkEligibility::NoVolume => GuiMarkEligibility::NoVolume,
        }
    }

    pub(in crate::owner) fn gui_mark_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiMarkPreviewState> {
        let preview = self.create_features.mark.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiMarkPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(MarkOutcome::Ready(_)) => GuiMarkPreview::Ready,
                    Ok(MarkOutcome::Incomplete) => GuiMarkPreview::Incomplete,
                    Err(cause) => GuiMarkPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_mark_face_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiMarkFacePick> {
        self.create_features
            .mark
            .face_pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn gui_mark_sketch_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiMarkSketchPick> {
        self.create_features
            .mark
            .sketch_pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn mark_command(&mut self, action: GuiMarkAction) -> Result<bool, String> {
        match action {
            GuiMarkAction::Open | GuiMarkAction::OpenEdit(_) => {
                self.create_features.mark.active = true;
                Ok(false)
            }
            GuiMarkAction::Clear => {
                let had_state = self.create_features.mark.active
                    | self.create_features.mark.preview.take().is_some()
                    | self.create_features.mark.face_pick.take().is_some()
                    | self.create_features.mark.sketch_pick.take().is_some();
                self.create_features.mark.active = false;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiMarkAction::ExpandFaceChain {
                document,
                generation,
                seed_face,
                automatic_face_chain,
            } => {
                let pick = self.resolve_mark_face_pick(
                    document,
                    generation,
                    seed_face,
                    automatic_face_chain,
                )?;
                self.create_features.mark.face_pick = Some(pick);
                Ok(true)
            }
            GuiMarkAction::Preview(request) => {
                let request = *request;
                self.require_mark_generation(&request)?;
                self.create_features.mark.active = true;
                let result = self.preview_mark(&request);
                self.create_features.mark.preview = Some(MarkPreview { request, result });
                Ok(true)
            }
            GuiMarkAction::Commit(request) => {
                let request = *request;
                self.require_mark_generation(&request)?;
                let commit = self
                    .create_features
                    .mark
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        MarkOutcome::Ready(commit) => Some(commit),
                        MarkOutcome::Incomplete => None,
                    })
                    .ok_or("mark: expected a current certified preview")?
                    .clone();
                self.commit_mark(request.document, commit)?;
                self.create_features.mark.preview = None;
                self.create_features.mark.face_pick = None;
                self.create_features.mark.sketch_pick = None;
                self.create_features.mark.active = false;
                self.model_face = None;
                self.face_appearance_picks.clear();
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_mark_generation(&self, request: &GuiMarkRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("mark: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_mark(&mut self, request: &GuiMarkRequest) -> Result<MarkOutcome, String> {
        let geometry_sets: Vec<MarkGeometrySet> = request
            .geometry_sets
            .iter()
            .filter_map(|set| {
                let sketch = set.sketch?;
                (!set.curves.is_empty()).then(|| MarkGeometrySet {
                    sketch,
                    curves: set.curves.clone(),
                    mark_type: set.mark_type,
                })
            })
            .collect();
        if geometry_sets.is_empty() {
            self.clear_extrusion_preview();
            return Ok(MarkOutcome::Incomplete);
        }
        if request.method == MarkMethod::WrapToFace && request.faces.is_empty() {
            self.clear_extrusion_preview();
            return Ok(MarkOutcome::Incomplete);
        }
        let body = match request.body.clone() {
            Some(body) => body,
            None => self.sole_mark_body_key(request.document)?,
        };
        let inputs = MarkFeatureInputs {
            body,
            geometry_sets,
            method: request.method,
            faces: request.faces.clone(),
            automatic_face_chain: request.automatic_face_chain,
            direction: request.direction,
        };
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_mark_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                MarkCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited = build_mark_definition(&inputs).map_err(|cause| cause.to_string())?;
                MarkCommit::Edit(id, Box::new(edited))
            }
        };
        match self.evaluate_mark_preview_scene(request, &commit) {
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
                Ok(MarkOutcome::Ready(commit))
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn sole_mark_body_key(&self, document: u64) -> Result<Vec<u8>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        let solids: Vec<_> = evaluation
            .ordered_bodies()
            .filter(|body| !body.is_surface())
            .collect();
        match solids.as_slice() {
            [] => Err("Mark failed. The selected component has no volume.".into()),
            [only] => Ok(only.body_reference_key(session.history(), guid)?.to_vec()),
            _ => Err("Mark is not supported when multiple solid bodies exist.".into()),
        }
    }

    fn evaluate_mark_preview_scene(
        &mut self,
        request: &GuiMarkRequest,
        commit: &MarkCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            MarkCommit::Create(prepared) => {
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
            MarkCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_mark(candidate, id, edited.clone()),
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

    fn commit_mark(&mut self, document: u64, commit: MarkCommit) -> Result<(), String> {
        match commit {
            MarkCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Mark",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            MarkCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Mark",
                    |snapshot, _| replace_mark(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}
