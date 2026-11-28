// SPDX-License-Identifier: GPL-2.0-only
//! Preview, edit seeds, and checkpointed commit for the desktop Coil Properties panel.
use super::*;
use oblikovati_api::Error;
use oblikovati_automation::{
    CoilInputs, PartUpdateAccess, PreparedCoil, automatic_revolve_profile, build_coil_definition,
    prepare_coil_definition, sweep_boolean_target,
};
use oblikovati_gui::{
    FeatureOutputMode, GuiCoilAction, GuiCoilEditSeed, GuiCoilPreview, GuiCoilPreviewState,
    GuiCoilRequest,
};
use oblikovati_model::{
    feature::{
        FeatureId,
        part::PartFeatureDefinition,
        sketch_coil::SketchCoilDefinition,
        sketch_prism::PrismBoolean,
        work_geometry::{DatumAngle, DatumLength},
    },
    history::DocumentSnapshot,
    quantity::{Dimension, Quantity},
};
use std::collections::BTreeMap;

#[derive(Clone)]
enum CoilCommit {
    Create(Box<PreparedCoil>),
    Edit(FeatureId, Box<SketchCoilDefinition>),
}

enum CoilOutcome {
    Ready(CoilCommit),
    UnchangedCut(String),
}

pub(in crate::owner) struct CoilPreview {
    request: GuiCoilRequest,
    result: Result<Option<CoilOutcome>, String>,
}

pub(in crate::owner) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiCoilEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::SketchCoil(definition) = feature.definition() else {
                return None;
            };
            let pitch = definition
                .pitch()
                .and_then(|v| length_field(snapshot, v))
                .unwrap_or_else(|| "15 mm".into());
            let revolutions = definition
                .revolutions()
                .map(format_number)
                .unwrap_or_else(|| "2.5".into());
            let height = definition
                .height()
                .and_then(|v| length_field(snapshot, v))
                .unwrap_or_else(|| "37.5 mm".into());
            let taper = angle_field(snapshot, definition.taper()).unwrap_or_else(|| "0 deg".into());
            let (close_start, start_flat, start_transition) =
                end_fields(snapshot, definition.start_end());
            let (close_end, end_flat, end_transition) = end_fields(snapshot, definition.end_end());
            Some((
                feature.id(),
                GuiCoilEditSeed {
                    axis: Some(definition.axis()),
                    axis_reversed: definition.axis_reversed(),
                    method: definition.method(),
                    pitch,
                    revolutions,
                    height,
                    taper,
                    handedness: definition.handedness(),
                    close_start,
                    start_flat,
                    start_transition,
                    close_end,
                    end_flat,
                    end_transition,
                    output: saved_output(definition),
                },
            ))
        })
        .collect()
}

fn saved_output(definition: &SketchCoilDefinition) -> FeatureOutputMode {
    match definition.boolean_target().map(|(op, _, _)| op) {
        Some(PrismBoolean::Join) => FeatureOutputMode::Join,
        Some(PrismBoolean::Cut) => FeatureOutputMode::Cut,
        Some(PrismBoolean::Intersect) => FeatureOutputMode::Intersect,
        None => FeatureOutputMode::NewSolid,
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

fn angle_field(snapshot: &DocumentSnapshot, angle: DatumAngle) -> Option<String> {
    match angle.parameter_id() {
        Some(id) => Some(snapshot.parameters.get(id)?.name.clone()),
        None => {
            let degrees = angle.resolve(&snapshot.parameters).ok()?.to_degrees();
            Some(format!("{} deg", format_number(degrees)))
        }
    }
}

fn end_fields(
    snapshot: &DocumentSnapshot,
    end: Option<oblikovati_model::feature::sketch_coil::CoilEndTreatment>,
) -> (bool, String, String) {
    match end {
        Some(treatment) => (
            true,
            angle_field(snapshot, treatment.flat_angle).unwrap_or_else(|| "90 deg".into()),
            angle_field(snapshot, treatment.transition_angle).unwrap_or_else(|| "90 deg".into()),
        ),
        None => (false, "90 deg".into(), "90 deg".into()),
    }
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

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(in crate::owner) fn gui_coil_preview(
        &self,
        document: u64,
        generation: u64,
    ) -> Option<GuiCoilPreviewState> {
        let preview = self.coil_preview.as_ref()?;
        (preview.request.document == document && preview.request.generation == generation).then(
            || GuiCoilPreviewState {
                request: preview.request.clone(),
                result: match &preview.result {
                    Ok(Some(CoilOutcome::Ready(_))) => GuiCoilPreview::Ready,
                    Ok(Some(CoilOutcome::UnchangedCut(cause))) => {
                        GuiCoilPreview::UnchangedCut(cause.clone())
                    }
                    Ok(None) if preview.request.profile_cleared => GuiCoilPreview::MissingProfile,
                    Ok(None) => GuiCoilPreview::MissingAxis,
                    Err(cause) => GuiCoilPreview::Failed(cause.clone()),
                },
            },
        )
    }

    pub(in crate::owner) fn coil_command(&mut self, action: GuiCoilAction) -> Result<bool, String> {
        match action {
            GuiCoilAction::Open | GuiCoilAction::OpenEdit(_) => Ok(false),
            GuiCoilAction::Clear => {
                Ok(self.coil_preview.take().is_some() | self.clear_extrusion_preview())
            }
            GuiCoilAction::Preview(request) => {
                let request = *request;
                self.require_coil_generation(&request)?;
                let result = self.preview_coil(&request);
                self.coil_preview = Some(CoilPreview { request, result });
                Ok(true)
            }
            GuiCoilAction::Commit(request) => {
                let request = *request;
                self.require_coil_generation(&request)?;
                let commit = self
                    .coil_preview
                    .as_ref()
                    .filter(|preview| preview.request == request)
                    .and_then(|preview| preview.result.as_ref().ok())
                    .and_then(Option::as_ref)
                    .and_then(|outcome| match outcome {
                        CoilOutcome::Ready(commit) => Some(commit),
                        CoilOutcome::UnchangedCut(_) => None,
                    })
                    .ok_or("coil: expected a current certified preview")?
                    .clone();
                self.commit_coil(request.document, commit)?;
                self.coil_preview = None;
                self.clear_extrusion_preview();
                Ok(true)
            }
        }
    }

    fn require_coil_generation(&self, request: &GuiCoilRequest) -> Result<(), String> {
        self.require_browser_document(request.document)?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        if session.generation() != request.generation {
            return Err("coil: expected the current document generation".into());
        }
        Ok(())
    }

    fn hide_edited_or_clear(&mut self, request: &GuiCoilRequest) {
        if let Some(feature) = request.feature
            && let Ok(scene) = self.part_scenes.part_scene_without_feature_body(
                &self.files,
                request.document,
                feature,
            )
        {
            self.extrusion_preview = Some((
                request.document,
                request.generation,
                scene,
                super::super::owner_extrusion_preview::PreviewKind::Replacement,
            ));
            self.part_scenes.invalidate_presentation();
        } else if self.extrusion_preview.take().is_some() {
            self.part_scenes.invalidate_presentation();
        }
    }

    fn preview_coil(&mut self, request: &GuiCoilRequest) -> Result<Option<CoilOutcome>, String> {
        if request.profile_cleared || request.axis.is_none() {
            self.hide_edited_or_clear(request);
            return Ok(None);
        }
        let outcome = match request.feature {
            Some(feature) => self.preview_coil_edit(request, feature).map(Some),
            None => self.preview_coil_create(request),
        };
        if outcome.is_err() {
            self.hide_edited_or_clear(request);
        }
        outcome
    }

    fn preview_coil_create(
        &mut self,
        request: &GuiCoilRequest,
    ) -> Result<Option<CoilOutcome>, String> {
        let document = request.document;
        let axis = request.axis.ok_or("coil: expected an axis")?;
        let policy =
            <crate::part_scene::DesktopPartScenes as PartUpdateAccess<O, W, F>>::resolution_policy(
                &self.part_scenes,
            )
            .ok_or("coil: expected a certified resolution policy")?;
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, document)?;
        let snapshot = self
            .files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let Some((sketch, selection)) = automatic_revolve_profile(snapshot, &datums, policy) else {
            self.hide_edited_or_clear(request);
            return Ok(None);
        };
        let boolean_target = request
            .output
            .boolean()
            .map(|op| {
                sweep_boolean_target(&self.files, &self.part_scenes, document, None)
                    .map(|key| (op, key))
                    .map_err(|cause| cause.to_string())
            })
            .transpose()?;
        let prepared = prepare_coil_definition(
            snapshot,
            &datums,
            coil_inputs(request, sketch, selection, axis, boolean_target),
        )
        .map_err(|cause| cause.to_string())?;
        if request.preview_hidden {
            self.hide_edited_or_clear(request);
        } else {
            let preview = prepared.clone();
            self.preview_selected_blend(document, |candidate, ids| preview.create(candidate, ids))?;
        }
        Ok(Some(CoilOutcome::Ready(CoilCommit::Create(Box::new(
            prepared,
        )))))
    }

    fn preview_coil_edit(
        &mut self,
        request: &GuiCoilRequest,
        id: FeatureId,
    ) -> Result<CoilOutcome, String> {
        use crate::part_scene::FeatureEditPreviewError;
        let axis = request.axis.ok_or("coil: expected an axis")?;
        let datums = self
            .part_scenes
            .datum_evaluation(&mut self.files, request.document)?;
        let snapshot = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let existing = coil_definition(snapshot, id)?.clone();
        let boolean_target = request
            .output
            .boolean()
            .map(|op| match existing.boolean_target() {
                Some((_, key, _)) => Ok((op, key.to_vec())),
                None => {
                    sweep_boolean_target(&self.files, &self.part_scenes, request.document, Some(id))
                        .map(|key| (op, key))
                        .map_err(|cause| cause.to_string())
                }
            })
            .transpose()?;
        let edited = build_coil_definition(
            snapshot,
            &datums,
            coil_inputs(
                request,
                existing.sketch(),
                existing.selection().clone(),
                axis,
                boolean_target,
            ),
        )
        .map_err(|cause| cause.to_string())?;
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let projection = session
            .preview_projected(
                |candidate, _| replace_coil(candidate, id, edited.clone()),
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
                    self.hide_edited_or_clear(request);
                } else {
                    self.extrusion_preview = Some((
                        request.document,
                        request.generation,
                        scene,
                        super::super::owner_extrusion_preview::PreviewKind::Replacement,
                    ));
                    self.part_scenes.invalidate_presentation();
                }
                Ok(CoilOutcome::Ready(CoilCommit::Edit(id, Box::new(edited))))
            }
            Err(FeatureEditPreviewError::UnchangedCut(cause)) => {
                Ok(CoilOutcome::UnchangedCut(cause))
            }
            Err(cause) => Err(cause.to_string()),
        }
    }

    fn commit_coil(&mut self, document: u64, commit: CoilCommit) -> Result<(), String> {
        match commit {
            CoilCommit::Create(prepared) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Coil",
                    |snapshot, ids| prepared.create(snapshot, ids),
                )
                .map_err(|cause| cause.to_string())?;
            }
            CoilCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Coil",
                    |snapshot, _| replace_coil(snapshot, id, *definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}

fn coil_inputs<'a>(
    request: &'a GuiCoilRequest,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
    selection: oblikovati_model::feature::sketch_prism::SketchPrismSelection,
    axis: oblikovati_model::feature::sketch_coil::CoilAxisSource,
    boolean_target: Option<(PrismBoolean, Vec<u8>)>,
) -> CoilInputs<'a> {
    CoilInputs {
        sketch,
        selection,
        axis,
        axis_reversed: request.axis_reversed,
        method: request.method,
        pitch: &request.pitch,
        revolutions: &request.revolutions,
        height: &request.height,
        taper: &request.taper,
        handedness: request.handedness,
        start_end: request.close_start.then_some((
            request.start_flat.as_str(),
            request.start_transition.as_str(),
        )),
        end_end: request
            .close_end
            .then_some((request.end_flat.as_str(), request.end_transition.as_str())),
        boolean_target,
    }
}

fn coil_definition(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
) -> Result<&SketchCoilDefinition, String> {
    let authored = snapshot
        .work_geometry
        .get(id)
        .map_err(|cause| cause.to_string())?;
    let PartFeatureDefinition::SketchCoil(definition) = authored.definition() else {
        return Err(format!("feature {id:?}: expected a Coil"));
    };
    Ok(definition)
}

fn replace_coil(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: SketchCoilDefinition,
) -> Result<(), Error> {
    let reads = definition.reads(snapshot).map_err(Error)?;
    snapshot
        .work_geometry
        .replace(
            id,
            PartFeatureDefinition::SketchCoil(Box::new(definition)),
            reads,
        )
        .map_err(|cause| Error(cause.to_string()))
        .map(|_| ())
}
