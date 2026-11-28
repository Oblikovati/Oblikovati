// SPDX-License-Identifier: GPL-2.0-only
//! Edit seeds, preview, and commit for Part Modify Split (#3918).
#[path = "owner_split_pick.rs"]
mod pick;
#[path = "owner_split_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedSplit, build_split_definition, prepare_split_definition, replace_split,
};
use oblikovati_gui::{
    GuiSplitAction, GuiSplitEditSeed, GuiSplitPick, GuiSplitPickMode, GuiSplitPreview,
    GuiSplitPreviewState, GuiSplitRequest,
};
use oblikovati_martensite::{EvaluationFailure, NamedSolid};
use oblikovati_model::{
    feature::{
        FeatureId,
        part::PartFeatureEvaluation,
        split::{SplitDefinition, SplitKeepSide, SplitMode},
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::{DerivedReferenceName, DocumentGuid},
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum SplitCommit {
    Create(Box<PreparedSplit>),
    Edit(FeatureId, Box<SplitDefinition>),
}

enum SplitOutcome {
    Ready(SplitCommit),
    Incomplete,
}

pub(in crate::owner) struct SplitPreview {
    request: GuiSplitRequest,
    result: Result<SplitOutcome, String>,
}

struct SplitBodyNaming {
    guid: DocumentGuid,
    active_parent: Vec<u8>,
    preserved: Vec<(String, String)>,
    next_num: usize,
}

#[derive(Default)]
pub(in crate::owner) struct SplitOwnerState {
    pub(in crate::owner) active: bool,
    pub(in crate::owner) editing: Option<FeatureId>,
    pub(in crate::owner) pick_mode: GuiSplitPickMode,
    pub(in crate::owner) split_mode: SplitMode,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiSplitPick>,
    pub(in crate::owner) preview: Option<SplitPreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
    evaluation: Option<&PartFeatureEvaluation<NamedSolid, EvaluationFailure>>,
    history: &DocumentEditHistory,
    guid: DocumentGuid,
) -> BTreeMap<FeatureId, GuiSplitEditSeed> {
    seed::edit_seeds(snapshot, evaluation, history, guid)
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
    pub(in crate::owner) fn gui_split_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiSplitPreviewState> {
        let preview = self.create_features.split.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiSplitPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(SplitOutcome::Ready(_)) => GuiSplitPreview::Ready,
                    Ok(SplitOutcome::Incomplete) => GuiSplitPreview::Incomplete,
                    Err(cause) => GuiSplitPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_split_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiSplitPick> {
        self.create_features
            .split
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn split_command(
        &mut self,
        action: GuiSplitAction,
    ) -> Result<bool, String> {
        match action {
            GuiSplitAction::Open => {
                self.create_features.split.active = true;
                self.create_features.split.editing = None;
                self.create_features.split.pick_mode = GuiSplitPickMode::Tool;
                self.create_features.split.split_mode = SplitMode::Faces;
                Ok(false)
            }
            GuiSplitAction::OpenEdit(id) => {
                self.create_features.split.active = true;
                self.create_features.split.editing = Some(id);
                self.create_features.split.pick_mode = GuiSplitPickMode::Target;
                Ok(false)
            }
            GuiSplitAction::SetPickMode(pick_mode, split_mode) => {
                self.create_features.split.pick_mode = pick_mode;
                self.create_features.split.split_mode = split_mode;
                Ok(false)
            }
            GuiSplitAction::Clear => {
                let had_state = self.create_features.split.active
                    | self.create_features.split.preview.take().is_some()
                    | self.create_features.split.pick.take().is_some();
                self.create_features.split.active = false;
                self.create_features.split.editing = None;
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiSplitAction::Preview(request) => {
                let request = *request;
                self.require_split_generation(&request)?;
                self.create_features.split.active = true;
                self.create_features.split.editing = request.feature;
                self.create_features.split.split_mode = request.mode;
                let result = self.preview_split(&request);
                self.create_features.split.preview = Some(SplitPreview { request, result });
                Ok(true)
            }
            GuiSplitAction::Commit(request) => {
                let request = *request;
                self.require_split_generation(&request)?;
                let commit = self
                    .create_features
                    .split
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        SplitOutcome::Ready(commit) => Some(commit),
                        SplitOutcome::Incomplete => None,
                    })
                    .ok_or("split: expected a current certified preview")?
                    .clone();
                self.commit_split(request.document, commit)?;
                self.create_features.split.preview = None;
                self.create_features.split.pick = None;
                self.create_features.split.active = false;
                self.create_features.split.editing = None;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_split_generation(&self, request: &GuiSplitRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("split: expected the current document generation".into());
        }
        Ok(())
    }

    fn preview_split(&mut self, request: &GuiSplitRequest) -> Result<SplitOutcome, String> {
        let Some(inputs) = seed::request_to_inputs(request) else {
            self.clear_extrusion_preview();
            return Ok(SplitOutcome::Incomplete);
        };
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_split_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                SplitCommit::Create(Box::new(prepared))
            }
            Some(id) => {
                let edited = build_split_definition(&inputs).map_err(|cause| cause.to_string())?;
                SplitCommit::Edit(id, Box::new(edited))
            }
        };
        match self.evaluate_split_preview_scene(request, &commit) {
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
                Ok(SplitOutcome::Ready(commit))
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn evaluate_split_preview_scene(
        &mut self,
        request: &GuiSplitRequest,
        commit: &SplitCommit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            SplitCommit::Create(prepared) => {
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
            SplitCommit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_split(candidate, id, edited.clone()),
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

    fn commit_split(&mut self, document: u64, commit: SplitCommit) -> Result<(), String> {
        match commit {
            SplitCommit::Create(prepared) => {
                let multi_body = prepared.definition().mode() == SplitMode::Solid
                    && prepared.definition().keep_side() == SplitKeepSide::Both;
                let naming = if multi_body {
                    Some(self.prepare_split_body_naming(document, prepared.definition().body())?)
                } else {
                    None
                };
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Split",
                    |snapshot, ids| {
                        let id = prepared.create(snapshot, ids)?;
                        if let Some(SplitBodyNaming {
                            guid,
                            active_parent,
                            preserved,
                            next_num,
                        }) = naming
                        {
                            for (key, name) in preserved {
                                snapshot.body_names.entry(key).or_insert(name);
                            }
                            let key_a = DerivedReferenceName::new(
                                guid,
                                id.0,
                                4,
                                "split-a",
                                std::slice::from_ref(&active_parent),
                            )
                            .and_then(|name| name.encode())
                            .map_err(oblikovati_api::Error)?;
                            let key_b = DerivedReferenceName::new(
                                guid,
                                id.0,
                                4,
                                "split-b",
                                std::slice::from_ref(&active_parent),
                            )
                            .and_then(|name| name.encode())
                            .map_err(oblikovati_api::Error)?;
                            snapshot
                                .body_names
                                .insert(seed::encode_hex_key(&key_a), format!("Solid{next_num}"));
                            snapshot.body_names.insert(
                                seed::encode_hex_key(&key_b),
                                format!("Solid{}", next_num + 1),
                            );
                        }
                        Ok(id)
                    },
                )
                .map_err(|cause| cause.to_string())?;
            }
            SplitCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Split",
                    |snapshot, _| replace_split(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }

    fn prepare_split_body_naming(
        &self,
        document: u64,
        target_body: &[u8],
    ) -> Result<SplitBodyNaming, String> {
        let rows = self.part_scenes.body_rows(&self.files, document)?;
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let max_num = seed::max_solid_number(&rows, &session.snapshot().body_names);
        let target_hex = seed::encode_hex_key(target_body);
        let preserved: Vec<(String, String)> = rows
            .iter()
            .filter(|row| !row.surface && row.key != target_hex)
            .map(|row| (row.key.clone(), row.name.clone()))
            .collect();
        let evaluation = self.part_scenes.current(session)?;
        let active_parent = evaluation
            .ordered_bodies()
            .find(|body| {
                body.body_reference_key(session.history(), guid)
                    .is_ok_and(|key| key == target_body)
            })
            .and_then(|body| body.active_body_reference_key(session.history(), guid).ok())
            .map(|key| key.to_vec())
            .unwrap_or_else(|| target_body.to_vec());
        Ok(SplitBodyNaming {
            guid,
            active_parent,
            preserved,
            next_num: max_num + 1,
        })
    }
}
