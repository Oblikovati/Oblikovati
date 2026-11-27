// SPDX-License-Identifier: GPL-2.0-only
//! Preview, edit seeds, image-sketch discovery, and commit for the desktop Decal Properties panel.
use super::*;
use oblikovati_automation::{
    DecalInputs, PartUpdateAccess, PreparedDecal, build_decal_definition, prepare_decal_definition,
    replace_decal, sweep_boolean_target,
};
use oblikovati_gui::{
    GuiDecalAction, GuiDecalEditSeed, GuiDecalImageCandidate, GuiDecalPreview,
    GuiDecalPreviewState, GuiDecalRequest,
};
use oblikovati_martensite::CertifiedNamedSolidEvaluator;
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchPlacement},
    feature::{
        FeatureId,
        part::{PartFeatureDefinition, PartFeatureEvaluator},
        sketch_decal::SketchDecalDefinition,
        work_geometry::WorkGeometryEvaluation,
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum DecalCommit {
    Create(Box<PreparedDecal>),
    Edit(FeatureId, Box<SketchDecalDefinition>),
}

enum DecalOutcome {
    Ready(DecalCommit),
    Incomplete,
}

pub(in crate::owner) struct DecalPreview {
    request: GuiDecalRequest,
    wrap_to_face_available: bool,
    result: Result<DecalOutcome, String>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiDecalEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::SketchDecal(definition) = feature.definition() else {
                return None;
            };
            Some((
                feature.id(),
                GuiDecalEditSeed {
                    sketch: Some(definition.sketch()),
                    image: Some(definition.image()),
                    target_face: Some(definition.target_face().to_vec()),
                    target_body: Some(hex_encode(definition.target_body().0)),
                    chain_faces: definition.chain_faces(),
                    wrap_to_face: definition.wrap_to_face(),
                    wrap_to_face_available: definition.wrap_to_face(),
                },
            ))
        })
        .collect()
}

pub(in crate::owner) fn visible_image_candidates(
    snapshot: &DocumentSnapshot,
) -> Vec<GuiDecalImageCandidate> {
    snapshot
        .sketches
        .sketches()
        .iter()
        .filter(|sketch| sketch.visible())
        .flat_map(|sketch| {
            sketch
                .images()
                .iter()
                .map(move |img| GuiDecalImageCandidate {
                    sketch: sketch.id(),
                    sketch_name: sketch.name().into(),
                    image: img.id,
                    reference: img.reference.clone(),
                })
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
    pub(in crate::owner) fn gui_decal_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiDecalPreviewState> {
        let preview = self.create_features.decal.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiDecalPreviewState {
                request: preview.request.clone(),
                wrap_to_face_available: preview.wrap_to_face_available,
                result: match &preview.result {
                    Ok(DecalOutcome::Ready(_)) => GuiDecalPreview::Ready,
                    Ok(DecalOutcome::Incomplete) => GuiDecalPreview::Incomplete,
                    Err(cause) => GuiDecalPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_decal_images(&self, document: u64) -> Vec<GuiDecalImageCandidate> {
        self.files
            .workspace()
            .session(document)
            .map(|session| visible_image_candidates(session.snapshot()))
            .unwrap_or_default()
    }

    pub(in crate::owner) fn decal_command(
        &mut self,
        action: GuiDecalAction,
    ) -> Result<bool, String> {
        match action {
            GuiDecalAction::Open | GuiDecalAction::OpenEdit(_) => Ok(false),
            GuiDecalAction::Clear => {
                Ok(self.create_features.decal.take().is_some() | self.clear_extrusion_preview())
            }
            GuiDecalAction::Preview(request) => {
                let request = *request;
                self.require_decal_generation(&request)?;
                let (wrap_to_face_available, result) = self.preview_decal(&request);
                self.create_features.decal = Some(DecalPreview {
                    request,
                    wrap_to_face_available,
                    result,
                });
                Ok(true)
            }
            GuiDecalAction::Commit(request) => {
                let request = *request;
                self.require_decal_generation(&request)?;
                let commit = self
                    .create_features
                    .decal
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        DecalOutcome::Ready(commit) => Some(commit),
                        DecalOutcome::Incomplete => None,
                    })
                    .ok_or("decal: expected a current certified preview")?
                    .clone();
                self.commit_decal(request.document, commit)?;
                self.create_features.decal = None;
                self.model_face = None;
                self.face_appearance_picks.clear();
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_decal_generation(&self, request: &GuiDecalRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("decal: expected the current document generation".into());
        }
        Ok(())
    }

    fn face_supports_wrap(&self, document: u64, face_key: &[u8]) -> bool {
        let workspace = self.files.workspace();
        let Ok(session) = workspace.session(document) else {
            return false;
        };
        let Ok(identity) = workspace.identity(document) else {
            return false;
        };
        let Ok(evaluation) = self.part_scenes.current(session) else {
            return false;
        };
        let evaluator = CertifiedNamedSolidEvaluator::with_optional_policy(
            session.history(),
            identity.internal_name,
            None,
        );
        for feature in session.snapshot().work_geometry.features() {
            let Some(body) = evaluation.body(feature.id()) else {
                continue;
            };
            if evaluator.resolve_planar_face(body, face_key).is_ok() {
                return false;
            }
            if body
                .face_reference_keys(session.history(), identity.internal_name)
                .is_ok_and(|keys| keys.iter().any(|k| k.as_slice() == face_key))
            {
                return true;
            }
        }
        false
    }

    fn preview_decal(&mut self, request: &GuiDecalRequest) -> (bool, Result<DecalOutcome, String>) {
        let wrap_available = request
            .target_face
            .as_deref()
            .is_some_and(|face| self.face_supports_wrap(request.document, face));
        let outcome = match request.feature {
            Some(feature) => self.preview_decal_edit(request, feature, wrap_available),
            None => self.preview_decal_create(request, wrap_available),
        };
        (wrap_available, outcome)
    }

    fn decal_datums_for_sketch(
        &mut self,
        document: u64,
        sketch: DocumentSketchId,
    ) -> Result<WorkGeometryEvaluation, String> {
        let placement = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .sketches
            .get(sketch)
            .map_err(|cause| cause.to_string())?
            .placement();
        if let SketchPlacement::Face(face) = placement {
            let current =
                self.part_scenes
                    .current_part_datums_for_face(&mut self.files, document, &face)?;
            let session = self
                .files
                .workspace()
                .session(document)
                .map_err(|cause| cause.to_string())?;
            return current
                .current(session)
                .cloned()
                .map_err(|cause| cause.to_string());
        }
        self.part_scenes.datum_evaluation(&mut self.files, document)
    }

    fn preview_decal_create(
        &mut self,
        request: &GuiDecalRequest,
        wrap_available: bool,
    ) -> Result<DecalOutcome, String> {
        let document = request.document;
        let (Some(sketch), Some(image), Some(target_face)) =
            (request.sketch, request.image, request.target_face.clone())
        else {
            return Ok(DecalOutcome::Incomplete);
        };
        let target_body = match request.target_body.as_deref() {
            Some(hex) if !hex.is_empty() => decode_hex(hex)?,
            _ => sweep_boolean_target(&self.files, &self.part_scenes, document, None)
                .map_err(|cause| cause.to_string())?,
        };
        let datums = self.decal_datums_for_sketch(document, sketch)?;
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let prepared = prepare_decal_definition(
            snapshot,
            &datums,
            DecalInputs {
                sketch,
                image,
                target_face,
                target_body,
                chain_faces: request.chain_faces,
                wrap_to_face: request.wrap_to_face && wrap_available,
            },
        )
        .map_err(|cause| cause.to_string())?;
        if !request.preview_hidden {
            let preview = prepared.clone();
            self.preview_selected_blend(document, |candidate, ids| preview.create(candidate, ids))?;
        }
        Ok(DecalOutcome::Ready(DecalCommit::Create(Box::new(prepared))))
    }

    fn preview_decal_edit(
        &mut self,
        request: &GuiDecalRequest,
        id: FeatureId,
        wrap_available: bool,
    ) -> Result<DecalOutcome, String> {
        let existing = {
            let snapshot = self
                .files
                .workspace()
                .session(request.document)
                .map_err(|cause| cause.to_string())?
                .snapshot();
            let authored = snapshot
                .work_geometry
                .get(id)
                .map_err(|cause| cause.to_string())?;
            let PartFeatureDefinition::SketchDecal(definition) = authored.definition() else {
                return Err(format!("feature {id:?}: expected a Decal"));
            };
            definition.clone()
        };
        let (Some(sketch), Some(image), Some(target_face)) =
            (request.sketch, request.image, request.target_face.clone())
        else {
            return Ok(DecalOutcome::Incomplete);
        };
        let datums = self.decal_datums_for_sketch(request.document, sketch)?;
        let target_body = match request.target_body.as_deref() {
            Some(hex) if !hex.is_empty() => decode_hex(hex)?,
            _ => existing.target_body().0.to_vec(),
        };
        let snapshot = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let edited = build_decal_definition(
            snapshot,
            &datums,
            DecalInputs {
                sketch,
                image,
                target_face,
                target_body,
                chain_faces: request.chain_faces,
                wrap_to_face: request.wrap_to_face && wrap_available,
            },
        )
        .map_err(|cause| cause.to_string())?;
        Ok(DecalOutcome::Ready(DecalCommit::Edit(id, Box::new(edited))))
    }

    fn commit_decal(&mut self, document: u64, commit: DecalCommit) -> Result<(), String> {
        match commit {
            DecalCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Decal",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            DecalCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Decal",
                    |snapshot, _| replace_decal(snapshot, id, *definition),
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
        return Err("decal target body: expected even-length hex key".into());
    }
    let mut bytes = Vec::with_capacity(trimmed.len() / 2);
    for pair in trimmed.as_bytes().as_chunks::<2>().0 {
        let text = std::str::from_utf8(pair)
            .map_err(|_| "decal target body: expected ASCII hex".to_string())?;
        let byte = u8::from_str_radix(text, 16)
            .map_err(|_| "decal target body: expected hex digits".to_string())?;
        bytes.push(byte);
    }
    Ok(bytes)
}
