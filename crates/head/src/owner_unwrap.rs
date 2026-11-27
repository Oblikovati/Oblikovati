// SPDX-License-Identifier: GPL-2.0-only
//! Face chaining, alignment vertex discovery, preview, edit seeds, and commit for Unwrap.
use super::*;
use oblikovati_automation::{
    PreparedUnwrap, UnwrapInputs, build_unwrap_definition, prepare_unwrap_definition,
    replace_unwrap, sweep_boolean_target,
};
use oblikovati_gui::{
    GuiUnwrapAction, GuiUnwrapEditSeed, GuiUnwrapFacePick, GuiUnwrapPreview, GuiUnwrapPreviewState,
    GuiUnwrapRequest,
};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition, unwrap::UnwrapDefinition},
    history::DocumentSnapshot,
    identity::ReferenceEnvelope,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum UnwrapCommit {
    Create(Box<PreparedUnwrap>),
    Edit(FeatureId, Box<UnwrapDefinition>),
}

enum UnwrapOutcome {
    Ready(UnwrapCommit),
    Incomplete,
}

pub(in crate::owner) struct UnwrapPreview {
    request: GuiUnwrapRequest,
    result: Result<UnwrapOutcome, String>,
}

#[derive(Default)]
pub(in crate::owner) struct UnwrapOwnerState {
    pub(in crate::owner) pick: Option<GuiUnwrapFacePick>,
    pub(in crate::owner) preview: Option<UnwrapPreview>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiUnwrapEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Unwrap(definition) = feature.definition() else {
                return None;
            };
            Some((
                feature.id(),
                GuiUnwrapEditSeed {
                    target_body: Some(hex_encode(definition.target_body().0)),
                    faces: definition.faces().to_vec(),
                    alignment_vertex: definition.alignment_vertex().map(<[u8]>::to_vec),
                    auto_face_chain: definition.auto_face_chain(),
                    merge_result_body: definition.merge_result_body(),
                    alignment: definition.alignment(),
                    linear_result_edges: definition.linear_result_edges().to_vec(),
                    rigid_result_edges: definition.rigid_result_edges().to_vec(),
                    seam_edges: definition.seam_edges().to_vec(),
                },
            ))
        })
        .collect()
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
    pub(in crate::owner) fn gui_unwrap_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiUnwrapPreviewState> {
        let preview = self.create_features.unwrap.preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiUnwrapPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(UnwrapOutcome::Ready(_)) => GuiUnwrapPreview::Ready,
                    Ok(UnwrapOutcome::Incomplete) => GuiUnwrapPreview::Incomplete,
                    Err(cause) => GuiUnwrapPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_unwrap_pick(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiUnwrapFacePick> {
        self.create_features
            .unwrap
            .pick
            .as_ref()
            .filter(|pick| pick.document == document && pick.generation == generation)
            .cloned()
    }

    pub(in crate::owner) fn unwrap_command(
        &mut self,
        action: GuiUnwrapAction,
    ) -> Result<bool, String> {
        match action {
            GuiUnwrapAction::Open | GuiUnwrapAction::OpenEdit(_) => Ok(false),
            GuiUnwrapAction::Clear => {
                let had_state = self.create_features.unwrap.preview.take().is_some()
                    | self.create_features.unwrap.pick.take().is_some()
                    | self.model_face.take().is_some();
                self.face_appearance_picks.clear();
                Ok(had_state | self.clear_extrusion_preview())
            }
            GuiUnwrapAction::PickFace {
                document,
                generation,
                face,
                auto_face_chain,
            } => {
                let pick =
                    self.resolve_unwrap_face_pick(document, generation, face, auto_face_chain)?;
                self.create_features.unwrap.pick = Some(pick);
                Ok(true)
            }
            GuiUnwrapAction::Preview(request) => {
                let request = *request;
                self.require_unwrap_generation(&request)?;
                let result = self.preview_unwrap(&request);
                self.create_features.unwrap.preview = Some(UnwrapPreview { request, result });
                Ok(true)
            }
            GuiUnwrapAction::Commit(request) => {
                let request = *request;
                self.require_unwrap_generation(&request)?;
                let commit = self
                    .create_features
                    .unwrap
                    .preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        UnwrapOutcome::Ready(commit) => Some(commit),
                        UnwrapOutcome::Incomplete => None,
                    })
                    .ok_or("unwrap: expected a current certified preview")?
                    .clone();
                self.commit_unwrap(request.document, commit)?;
                self.create_features.unwrap.preview = None;
                self.create_features.unwrap.pick = None;
                self.model_face = None;
                self.face_appearance_picks.clear();
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_unwrap_generation(&self, request: &GuiUnwrapRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("unwrap: expected the current document generation".into());
        }
        Ok(())
    }

    fn resolve_unwrap_face_pick(
        &self,
        document: u64,
        generation: u64,
        face: Vec<u8>,
        auto_face_chain: bool,
    ) -> Result<GuiUnwrapFacePick, String> {
        self.require_browser_document(document)?;
        let workspace = self.files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != generation {
            return Err("unwrap pick: expected the current document generation".into());
        }
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
            if !face_keys.iter().any(|k| k.as_slice() == face.as_slice()) {
                continue;
            }
            let target_body = hex_encode(body.body_reference_key(session.history(), guid)?);
            let faces = if auto_face_chain {
                face_keys
            } else {
                vec![face.clone()]
            };
            let alignment_vertex = body
                .subentity_reference_keys(session.history(), guid)
                .ok()
                .and_then(|keys| {
                    keys.into_iter()
                        .find(|k| ReferenceEnvelope::decode(k).is_ok_and(|e| e.kind == 3))
                });
            return Ok(GuiUnwrapFacePick {
                document,
                generation,
                seed_face: face,
                auto_face_chain,
                target_body,
                faces,
                alignment_vertex,
            });
        }
        Err("unwrap pick: expected a live named solid face".into())
    }

    fn preview_unwrap(&mut self, request: &GuiUnwrapRequest) -> Result<UnwrapOutcome, String> {
        if request.faces.is_empty() {
            self.clear_extrusion_preview();
            return Ok(UnwrapOutcome::Incomplete);
        }
        let target_body = match request.target_body.as_deref() {
            Some(hex) if !hex.is_empty() => decode_hex(hex)?,
            _ => sweep_boolean_target(&self.files, &self.part_scenes, request.document, None)
                .map_err(|cause| cause.to_string())?,
        };
        let inputs = UnwrapInputs {
            target_body,
            faces: request.faces.clone(),
            alignment_vertex: request.alignment_vertex.clone(),
            auto_face_chain: request.auto_face_chain,
            merge_result_body: request.merge_result_body,
            alignment: request.alignment,
            linear_result_edges: request.linear_result_edges.clone(),
            rigid_result_edges: request.rigid_result_edges.clone(),
            seam_edges: request.seam_edges.clone(),
        };
        match request.feature {
            None => {
                let snapshot = self
                    .files
                    .workspace()
                    .session(request.document)
                    .map_err(|cause| cause.to_string())?
                    .snapshot();
                let prepared = prepare_unwrap_definition(snapshot, inputs)
                    .map_err(|cause| cause.to_string())?;
                if request.preview_hidden {
                    self.clear_extrusion_preview();
                } else {
                    let preview = prepared.clone();
                    self.preview_added_replacement(
                        request.document,
                        super::super::owner_extrusion_preview::PreviewKind::Additive,
                        |candidate, ids| preview.create(candidate, ids),
                    )?;
                }
                Ok(UnwrapOutcome::Ready(UnwrapCommit::Create(Box::new(
                    prepared,
                ))))
            }
            Some(id) => {
                let edited = build_unwrap_definition(inputs).map_err(|cause| cause.to_string())?;
                if request.preview_hidden {
                    self.clear_extrusion_preview();
                } else {
                    let session = self
                        .files
                        .workspace()
                        .session(request.document)
                        .map_err(|cause| cause.to_string())?;
                    let projection = session
                        .preview_projected(
                            |candidate, _| replace_unwrap(candidate, id, edited.clone()),
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
                    match projection {
                        Ok(scene) => {
                            self.extrusion_preview = Some((
                                request.document,
                                request.generation,
                                scene,
                                super::super::owner_extrusion_preview::PreviewKind::Replacement,
                            ));
                            self.part_scenes.invalidate_presentation();
                        }
                        Err(cause) => return Err(cause.to_string()),
                    }
                }
                Ok(UnwrapOutcome::Ready(UnwrapCommit::Edit(
                    id,
                    Box::new(edited),
                )))
            }
        }
    }

    fn commit_unwrap(&mut self, document: u64, commit: UnwrapCommit) -> Result<(), String> {
        match commit {
            UnwrapCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Unwrap",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            UnwrapCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Unwrap",
                    |snapshot, _| replace_unwrap(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}

fn hex_encode(bytes: &[u8]) -> String {
    use std::fmt::Write;
    let mut out = String::with_capacity(bytes.len() * 2);
    for byte in bytes {
        let _ = write!(&mut out, "{byte:02x}");
    }
    out
}

fn decode_hex(raw: &str) -> Result<Vec<u8>, String> {
    let trimmed = raw.trim();
    if !trimmed.len().is_multiple_of(2) {
        return Err("unwrap target body: expected even-length hex key".into());
    }
    let mut bytes = Vec::with_capacity(trimmed.len() / 2);
    for pair in trimmed.as_bytes().as_chunks::<2>().0 {
        let text = std::str::from_utf8(pair)
            .map_err(|_| "unwrap target body: expected ASCII hex".to_string())?;
        let byte = u8::from_str_radix(text, 16)
            .map_err(|_| "unwrap target body: expected hex digits".to_string())?;
        bytes.push(byte);
    }
    Ok(bytes)
}
