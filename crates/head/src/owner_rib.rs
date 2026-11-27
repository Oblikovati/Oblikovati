// SPDX-License-Identifier: GPL-2.0-only
//! Preview, edit seeds, open-sketch discovery, and commit for the desktop Rib Properties panel.
use super::*;
use oblikovati_automation::{
    PartUpdateAccess, PreparedRib, RibInputs, build_rib_definition, open_profile_lines,
    prepare_rib_definition, replace_rib, sweep_boolean_target,
};
use oblikovati_gui::{
    GuiRibAction, GuiRibEditSeed, GuiRibPreview, GuiRibPreviewState, GuiRibRequest,
    GuiRibSketchCandidate,
};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, SketchPlacement},
    feature::{
        FeatureId,
        part::PartFeatureDefinition,
        sketch_rib::SketchRibDefinition,
        work_geometry::{DatumAngle, DatumLength, WorkGeometryEvaluation},
    },
    history::DocumentSnapshot,
    quantity::{Dimension, Quantity},
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum RibCommit {
    Create(Box<PreparedRib>),
    Edit(FeatureId, Box<SketchRibDefinition>),
}

enum RibOutcome {
    Ready(RibCommit),
    MissingProfile,
    MissingSolid,
    RangeError(String),
}

pub(in crate::owner) struct RibPreview {
    request: GuiRibRequest,
    result: Result<RibOutcome, String>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiRibEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::SketchRib(definition) = feature.definition() else {
                return None;
            };
            let thickness =
                length_field(snapshot, definition.thickness()).unwrap_or_else(|| "1 mm".into());
            let depth = definition
                .depth()
                .and_then(|d| length_field(snapshot, d))
                .unwrap_or_else(|| "5 mm".into());
            let draft = angle_field(snapshot, definition.draft());
            Some((
                feature.id(),
                GuiRibEditSeed {
                    sketch: Some(definition.sketch()),
                    profile_curves: definition.profile_curves().to_vec(),
                    thickness,
                    depth,
                    to_next: definition.to_next(),
                    direction: definition.direction(),
                    flip_direction: definition.flip_direction(),
                    thicken_side: definition.thicken_side(),
                    extend_profile: definition.extend_profile(),
                    draft,
                    thickness_plane: definition.thickness_plane(),
                    target_body: Some(hex_encode(definition.target_body().0)),
                },
            ))
        })
        .collect()
}

pub(in crate::owner) fn visible_open_sketches(
    snapshot: &DocumentSnapshot,
) -> Vec<GuiRibSketchCandidate> {
    snapshot
        .sketches
        .sketches()
        .iter()
        .filter(|sketch| {
            sketch.visible()
                && snapshot
                    .work_geometry
                    .sketch_consumers(sketch.id())
                    .next()
                    .is_none()
        })
        .filter_map(|sketch| {
            let curves = open_profile_lines(snapshot, sketch.id());
            (!curves.is_empty()).then(|| GuiRibSketchCandidate {
                sketch: sketch.id(),
                name: sketch.name().into(),
                curves,
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
    pub(in crate::owner) fn gui_rib_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiRibPreviewState> {
        let preview = self.rib_preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiRibPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(RibOutcome::Ready(_)) => GuiRibPreview::Ready,
                    Ok(RibOutcome::MissingProfile) => GuiRibPreview::MissingProfile,
                    Ok(RibOutcome::MissingSolid) => GuiRibPreview::MissingSolid,
                    Ok(RibOutcome::RangeError(msg)) => GuiRibPreview::RangeError(msg.clone()),
                    Err(cause) => GuiRibPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn gui_rib_sketches(&self, document: u64) -> Vec<GuiRibSketchCandidate> {
        self.files
            .workspace()
            .session(document)
            .map(|session| visible_open_sketches(session.snapshot()))
            .unwrap_or_default()
    }

    pub(in crate::owner) fn rib_command(&mut self, action: GuiRibAction) -> Result<bool, String> {
        match action {
            GuiRibAction::Open | GuiRibAction::OpenEdit(_) => Ok(false),
            GuiRibAction::Clear => {
                Ok(self.rib_preview.take().is_some() | self.clear_extrusion_preview())
            }
            GuiRibAction::Preview(request) => {
                let request = *request;
                self.require_rib_generation(&request)?;
                let result = self.preview_rib(&request);
                self.rib_preview = Some(RibPreview { request, result });
                Ok(true)
            }
            GuiRibAction::Commit(request) => {
                let request = *request;
                self.require_rib_generation(&request)?;
                let commit = self
                    .rib_preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(|outcome| match outcome {
                        RibOutcome::Ready(commit) => Some(commit),
                        _ => None,
                    })
                    .ok_or("rib: expected a current certified preview")?
                    .clone();
                self.commit_rib(request.document, commit)?;
                self.rib_preview = None;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_rib_generation(&self, request: &GuiRibRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("rib: expected the current document generation".into());
        }
        Ok(())
    }

    fn clear_rib_scene(&mut self) {
        if self.extrusion_preview.take().is_some() {
            self.part_scenes.invalidate_presentation();
        }
    }

    fn preview_rib(&mut self, request: &GuiRibRequest) -> Result<RibOutcome, String> {
        let outcome = match request.feature {
            Some(feature) => self.preview_rib_edit(request, feature),
            None => self.preview_rib_create(request),
        };
        if !matches!(outcome, Ok(RibOutcome::Ready(_))) {
            self.clear_rib_scene();
        }
        outcome
    }

    fn rib_datums_for_sketch(
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

    fn preview_rib_create(&mut self, request: &GuiRibRequest) -> Result<RibOutcome, String> {
        let document = request.document;
        let Some(sketch) = request.sketch else {
            return Ok(RibOutcome::MissingProfile);
        };
        if request.profile_curves.is_empty() {
            return Ok(RibOutcome::MissingProfile);
        }
        let target_body = match request.target_body.as_deref() {
            Some(hex) if !hex.is_empty() => decode_hex(hex)?,
            _ => match sweep_boolean_target(&self.files, &self.part_scenes, document, None) {
                Ok(key) => key,
                Err(_) => return Ok(RibOutcome::MissingSolid),
            },
        };
        let datums = self.rib_datums_for_sketch(document, sketch)?;
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let prepared = match prepare_rib_definition(
            snapshot,
            &datums,
            rib_inputs(request, sketch, target_body),
        ) {
            Ok(prepared) => prepared,
            Err(cause) => {
                let message = cause.to_string();
                if message.contains("out of the expected range") {
                    return Ok(RibOutcome::RangeError(message));
                }
                return Err(message);
            }
        };
        if request.preview_hidden {
            self.clear_rib_scene();
        } else {
            let preview = prepared.clone();
            self.preview_selected_blend(document, |candidate, ids| preview.create(candidate, ids))?;
        }
        Ok(RibOutcome::Ready(RibCommit::Create(Box::new(prepared))))
    }

    fn preview_rib_edit(
        &mut self,
        request: &GuiRibRequest,
        id: FeatureId,
    ) -> Result<RibOutcome, String> {
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
            let PartFeatureDefinition::SketchRib(definition) = authored.definition() else {
                return Err(format!("feature {id:?}: expected a Rib"));
            };
            definition.clone()
        };
        let sketch = request.sketch.unwrap_or_else(|| existing.sketch());
        let datums = self.rib_datums_for_sketch(request.document, sketch)?;
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
        let edited =
            match build_rib_definition(snapshot, &datums, rib_inputs(request, sketch, target_body))
            {
                Ok(edited) => edited,
                Err(cause) => {
                    let message = cause.to_string();
                    if message.contains("out of the expected range") {
                        return Ok(RibOutcome::RangeError(message));
                    }
                    return Err(message);
                }
            };
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let projection = session
            .preview_projected(
                |candidate, _| replace_rib(candidate, id, edited.clone()),
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
                if request.preview_hidden {
                    self.clear_rib_scene();
                } else {
                    self.extrusion_preview = Some((
                        request.document,
                        request.generation,
                        scene,
                        super::super::owner_extrusion_preview::PreviewKind::Replacement,
                    ));
                    self.part_scenes.invalidate_presentation();
                }
                Ok(RibOutcome::Ready(RibCommit::Edit(id, Box::new(edited))))
            }
            Err(cause) => Err(cause.to_string()),
        }
    }

    fn commit_rib(&mut self, document: u64, commit: RibCommit) -> Result<(), String> {
        match commit {
            RibCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Rib",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            RibCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Rib",
                    |snapshot, _| replace_rib(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}

fn rib_inputs<'a>(
    request: &'a GuiRibRequest,
    sketch: DocumentSketchId,
    target_body: Vec<u8>,
) -> RibInputs<'a> {
    RibInputs {
        sketch,
        profile_curves: request.profile_curves.clone(),
        thickness: &request.thickness,
        depth: (!request.to_next && !request.depth.trim().is_empty())
            .then_some(request.depth.as_str()),
        to_next: request.to_next,
        direction: request.direction,
        flip_direction: request.flip_direction,
        thicken_side: request.thicken_side,
        extend_profile: request.extend_profile,
        draft: &request.draft,
        thickness_plane: request.thickness_plane,
        target_body,
    }
}

fn length_field(snapshot: &DocumentSnapshot, length: DatumLength) -> Option<String> {
    match length.parameter() {
        Some(id) => Some(snapshot.parameters.get(id)?.name.clone()),
        None => {
            let value = length.resolve(&snapshot.parameters).ok()?;
            let units = snapshot.parameters.units();
            let preferred = units.to_preferred(Quantity::new(value, Dimension::Length));
            let unit = units.preferred_name(Dimension::Length);
            Some(format!("{} {unit}", format_number(preferred)))
        }
    }
}

fn angle_field(snapshot: &DocumentSnapshot, angle: DatumAngle) -> String {
    if let Some(id) = angle.parameter_id()
        && let Some(param) = snapshot.parameters.get(id)
    {
        return param.name.clone();
    }
    let degrees = angle
        .resolve(&snapshot.parameters)
        .map(f64::to_degrees)
        .unwrap_or(0.0);
    format!("{} deg", format_number(degrees))
}

fn format_number(value: f64) -> String {
    let text = format!("{value:.4}");
    let trimmed = text.trim_end_matches('0').trim_end_matches('.');
    if trimmed == "-0" || trimmed.is_empty() {
        "0".into()
    } else {
        trimmed.into()
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
        return Err("rib target body: expected even-length hex key".into());
    }
    let mut bytes = Vec::with_capacity(trimmed.len() / 2);
    for pair in trimmed.as_bytes().as_chunks::<2>().0 {
        let text = std::str::from_utf8(pair)
            .map_err(|_| "rib target body: expected ASCII hex".to_string())?;
        let byte = u8::from_str_radix(text, 16)
            .map_err(|_| "rib target body: expected hex digits".to_string())?;
        bytes.push(byte);
    }
    Ok(bytes)
}
