// SPDX-License-Identifier: GPL-2.0-only
//! Face picking, tangent expansion, live preview, and commit for Shell v2 (#4106).
#[path = "owner_shell_v2_seed.rs"]
mod seed;

use super::*;
use oblikovati_automation::{
    PreparedShell, build_shell_v2_definition, prepare_shell_v2, replace_shell,
};
use oblikovati_gui::{
    GuiShellPickMode, GuiShellV2Action, GuiShellV2EditSeed, GuiShellV2Pick, GuiShellV2Preview,
    GuiShellV2PreviewState, GuiShellV2Request,
};
use oblikovati_model::{
    feature::{FeatureId, shell::ShellDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum ShellV2Commit {
    Create(Box<PreparedShell>),
    Edit(FeatureId, Box<ShellDefinition>),
}

enum ShellV2Outcome {
    Ready(ShellV2Commit),
    Incomplete,
}

pub(in crate::owner) struct ShellV2Preview {
    request: GuiShellV2Request,
    result: Result<ShellV2Outcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct ShellV2OwnerState {
    pub(in crate::owner) pick_mode: GuiShellPickMode,
    pub(in crate::owner) pick_seq: u64,
    pub(in crate::owner) pick: Option<GuiShellV2Pick>,
    pub(in crate::owner) preview: Option<ShellV2Preview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
    generation: u64,
) -> BTreeMap<FeatureId, GuiShellV2EditSeed> {
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
    pub(in crate::owner) fn gui_shell_v2_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiShellV2PreviewState> {
        let preview = self.create_features.shell_v2.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiShellV2PreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(ShellV2Outcome::Ready(_)) => GuiShellV2Preview::Ready,
                    Ok(ShellV2Outcome::Incomplete) => GuiShellV2Preview::Incomplete,
                    Err(cause) => GuiShellV2Preview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_shell_v2_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiShellV2Pick> {
        self.create_features
            .shell_v2
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn shell_v2_command(
        &mut self,
        action: GuiShellV2Action,
    ) -> Result<bool, String> {
        match action {
            GuiShellV2Action::SetPickMode(mode) => {
                self.create_features.shell_v2.pick_mode = mode;
                Ok(true)
            }
            GuiShellV2Action::ClearPreview => {
                let had = self.create_features.shell_v2.preview.take().is_some();
                Ok(had | self.clear_extrusion_preview())
            }
            GuiShellV2Action::Clear => {
                let had = self.create_features.shell_v2.pick_mode != GuiShellPickMode::None
                    || self.create_features.shell_v2.preview.take().is_some()
                    || self.create_features.shell_v2.pick.take().is_some();
                self.create_features.shell_v2.pick_mode = GuiShellPickMode::None;
                Ok(had | self.clear_extrusion_preview())
            }
            GuiShellV2Action::Preview(request) => {
                let request = *request;
                self.require_shell_v2_generation(&request)?;
                let result = self.preview_shell_v2(&request);
                self.create_features.shell_v2.preview = Some(ShellV2Preview { request, result });
                Ok(true)
            }
            GuiShellV2Action::Commit(request) => {
                let request = *request;
                self.finish_shell_v2(request)
            }
        }
    }

    fn finish_shell_v2(&mut self, request: GuiShellV2Request) -> Result<bool, String> {
        self.require_shell_v2_generation(&request)?;
        let commit = self
            .create_features
            .shell_v2
            .preview
            .as_ref()
            .filter(|preview| preview.request == request)
            .and_then(|preview| preview.result.as_ref().ok())
            .and_then(|outcome| match outcome {
                ShellV2Outcome::Ready(commit) => Some(commit),
                ShellV2Outcome::Incomplete => None,
            })
            .ok_or("shell: expected a current certified preview")?
            .clone();
        self.commit_shell_v2(request.document, commit)?;
        self.create_features.shell_v2.preview = None;
        self.create_features.shell_v2.pick = None;
        self.create_features.shell_v2.pick_mode = GuiShellPickMode::None;
        self.model_face = None;
        self.face_appearance_picks.clear();
        self.clear_extrusion_preview();
        Ok(true)
    }

    pub(in crate::owner) fn pick_shell_v2_geometry(
        &mut self,
        document: u64,
        ray: [[f64; 3]; 2],
        tolerance: Option<f64>,
    ) -> Result<Option<bool>, String> {
        let mode = self.create_features.shell_v2.pick_mode;
        if mode == GuiShellPickMode::None {
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
        let crate::part_scene::pick_candidates::PickTarget::Face(feature, face_key) = picked.target
        else {
            return Ok(None);
        };
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let generation = session.generation();
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let body = self
            .part_scenes
            .current(session)?
            .body(feature)
            .ok_or("shell face pick: expected a current body")?;
        let body_key = body.body_reference_key(session.history(), guid)?.to_vec();
        let editing_feature = self
            .create_features
            .shell_v2
            .preview
            .as_ref()
            .and_then(|p| p.request.feature);
        let resolved_face = unwrap_edited_shell_face(&face_key, editing_feature);
        let faces = match mode {
            GuiShellPickMode::Remove => body
                .shell_tangent_face_chain(session.history(), guid, &resolved_face)
                .unwrap_or_else(|_| vec![resolved_face]),
            GuiShellPickMode::Override(_) => vec![resolved_face],
            GuiShellPickMode::None => Vec::new(),
        };
        self.create_features.shell_v2.pick_seq += 1;
        self.create_features.shell_v2.pick = Some(GuiShellV2Pick {
            document,
            generation,
            sequence: self.create_features.shell_v2.pick_seq,
            mode,
            body: body_key,
            faces,
        });
        if matches!(mode, GuiShellPickMode::Override(_)) {
            self.create_features.shell_v2.pick_mode = GuiShellPickMode::None;
        }
        Ok(Some(true))
    }

    fn require_shell_v2_generation(&self, request: &GuiShellV2Request) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("shell: expected the current document generation".into());
        }
        Ok(())
    }

    fn resolve_body_for_shell_face(
        &self,
        document: u64,
        face_key: &[u8],
    ) -> Result<Vec<u8>, String> {
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.part_scenes.current(session)?;
        for body in evaluation.ordered_bodies().filter(|b| !b.is_surface()) {
            let faces = body.face_reference_keys(session.history(), guid)?;
            if faces
                .iter()
                .any(|candidate| candidate.as_slice() == face_key)
            {
                return Ok(body.body_reference_key(session.history(), guid)?.to_vec());
            }
        }
        Err("shell face: expected a face on a current solid body".into())
    }

    fn preview_shell_v2(&mut self, request: &GuiShellV2Request) -> Result<ShellV2Outcome, String> {
        if request.openings.is_empty() {
            self.clear_extrusion_preview();
            return Ok(ShellV2Outcome::Incomplete);
        }
        let body = match request.body.clone() {
            Some(body) if !body.is_empty() => body,
            _ => self.resolve_body_for_shell_face(request.document, &request.openings[0])?,
        };
        let inputs = seed::request_to_inputs(request, body);
        let commit = match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared =
                    prepare_shell_v2(snapshot, &inputs).map_err(|cause| cause.to_string())?;
                ShellV2Commit::Create(Box::new(prepared))
            }
            Some(id) => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let edited = build_shell_v2_definition(snapshot, &inputs)
                    .map_err(|cause| cause.to_string())?;
                ShellV2Commit::Edit(id, Box::new(edited))
            }
        };
        match self.evaluate_shell_v2_preview_scene(request, &commit) {
            Ok(scene) => {
                self.extrusion_preview = Some((
                    request.document,
                    request.generation,
                    scene,
                    super::super::owner_extrusion_preview::PreviewKind::Replacement,
                ));
                self.part_scenes.invalidate_presentation();
                Ok(ShellV2Outcome::Ready(commit))
            }
            Err(cause) => {
                self.clear_extrusion_preview();
                Err(cause)
            }
        }
    }

    fn evaluate_shell_v2_preview_scene(
        &mut self,
        request: &GuiShellV2Request,
        commit: &ShellV2Commit,
    ) -> Result<oblikovati_viewport::ViewportScene, String> {
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        match commit {
            ShellV2Commit::Create(prepared) => {
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
            ShellV2Commit::Edit(id, definition) => {
                let id = *id;
                let edited = (**definition).clone();
                let projection = session
                    .preview_projected(
                        |candidate, _| replace_shell(candidate, id, edited.clone()),
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

    fn commit_shell_v2(&mut self, document: u64, commit: ShellV2Commit) -> Result<(), String> {
        match commit {
            ShellV2Commit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Shell",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            ShellV2Commit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Shell",
                    |snapshot, _| replace_shell(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}

fn unwrap_edited_shell_face(face_key: &[u8], editing_feature: Option<FeatureId>) -> Vec<u8> {
    let Some(feature) = editing_feature else {
        return face_key.to_vec();
    };
    let Ok(derived) = oblikovati_model::identity::DerivedReferenceName::decode(face_key) else {
        return face_key.to_vec();
    };
    if derived.feature() != feature.0 {
        return face_key.to_vec();
    }
    for parent in derived.parents() {
        if let Ok(env) = oblikovati_model::identity::ReferenceEnvelope::decode(parent)
            && env.kind == 1
        {
            return parent.clone();
        }
    }
    face_key.to_vec()
}
