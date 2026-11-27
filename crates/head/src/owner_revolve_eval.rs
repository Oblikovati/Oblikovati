// SPDX-License-Identifier: GPL-2.0-only
//! Private Revolve preview evaluation and checkpointed commit helpers.
use super::*;
use oblikovati_api::Error;
use oblikovati_automation::{
    PartUpdateAccess, RevolveInputs, automatic_revolve_profile, parse_positive_turn_angle,
    prepare_revolve_definition, sweep_boolean_target,
};
use oblikovati_gui::{GuiRevolveExtent, GuiRevolveRequest};
use oblikovati_model::{
    feature::{
        FeatureId,
        part::PartFeatureDefinition,
        sketch_revolve::{SketchRevolveDefinition, SketchRevolveDirection},
        work_geometry::DatumAngle,
    },
    history::DocumentSnapshot,
};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn preview_revolve_create(
        &mut self,
        request: &GuiRevolveRequest,
    ) -> Result<(usize, Option<RevolveOutcome>), String> {
        let document = request.document;
        let policy =
            <crate::part_scene::DesktopPartScenes as PartUpdateAccess<O, W, F>>::resolution_policy(
                &self.part_scenes,
            )
            .ok_or("revolve: expected a certified resolution policy")?;
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
            self.clear_preview_mesh();
            return Ok((0, None));
        };
        let curves = selection.outer.len();
        let boolean_target = request
            .output
            .boolean()
            .map(|op| {
                sweep_boolean_target(&self.files, &self.part_scenes, document, None)
                    .map(|key| (op, key))
                    .map_err(|cause| cause.to_string())
            })
            .transpose()?;
        let second = (request.direction == SketchRevolveDirection::Asymmetric)
            .then_some(request.second_angle.as_str());
        let prepared = prepare_revolve_definition(
            snapshot,
            &datums,
            RevolveInputs {
                sketch,
                selection,
                axis: request.axis,
                full: request.extent == GuiRevolveExtent::Full,
                direction: request.direction,
                angle: &request.angle,
                second_angle: second,
                surface: request.output.is_surface(),
                boolean_target,
            },
        )
        .map_err(|cause| cause.to_string())?;
        if request.preview_hidden {
            self.clear_preview_mesh();
        } else {
            let preview = prepared.clone();
            self.preview_selected_blend(document, |candidate, ids| preview.create(candidate, ids))?;
        }
        Ok((
            curves,
            Some(RevolveOutcome::Ready(RevolveCommit::Create(Box::new(
                prepared,
            )))),
        ))
    }

    pub(super) fn preview_revolve_edit(
        &mut self,
        request: &GuiRevolveRequest,
        id: FeatureId,
    ) -> Result<RevolveOutcome, String> {
        use crate::part_scene::FeatureEditPreviewError;
        let snapshot = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?
            .snapshot();
        let existing = revolve_definition(snapshot, id)?.clone();
        let full = request.extent == GuiRevolveExtent::Full;
        let primary = if full {
            DatumAngle::constant(std::f64::consts::TAU).map_err(|cause| cause.to_string())?
        } else {
            parse_positive_turn_angle(snapshot, &request.angle, "revolve angle")
                .map_err(|cause| cause.to_string())?
        };
        let mut edited =
            SketchRevolveDefinition::new(existing.sketch(), existing.selection().clone(), primary)
                .map_err(|cause| cause.to_string())?;
        if let Some(axis) = request.axis.or_else(|| existing.axis()) {
            edited = edited.with_axis(axis).map_err(|cause| cause.to_string())?;
        }
        if full {
            edited = edited.with_full_turn();
        } else if request.direction == SketchRevolveDirection::Asymmetric {
            let second =
                parse_positive_turn_angle(snapshot, &request.second_angle, "revolve angle B")
                    .map_err(|cause| cause.to_string())?;
            edited = edited
                .with_asymmetric_angle(second)
                .map_err(|cause| cause.to_string())?;
        } else {
            edited = edited.with_direction(request.direction);
        }
        if request.output.is_surface() {
            edited = edited.with_surface_output(true);
        } else if let Some(operation) = request.output.boolean() {
            let key = match existing.boolean_target() {
                Some((_, key, _)) => key.to_vec(),
                None => {
                    sweep_boolean_target(&self.files, &self.part_scenes, request.document, Some(id))
                        .map_err(|cause| cause.to_string())?
                }
            };
            edited = edited
                .with_boolean_target(operation, key)
                .map_err(|cause| cause.to_string())?;
        } else {
            edited = edited.without_boolean_target();
        }
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        let projection = session
            .preview_projected(
                |candidate, _| replace_revolve(candidate, id, edited.clone()),
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
                    self.clear_preview_mesh();
                } else {
                    self.extrusion_preview = Some((
                        request.document,
                        request.generation,
                        scene,
                        super::super::super::owner_extrusion_preview::PreviewKind::Replacement,
                    ));
                    self.part_scenes.invalidate_presentation();
                }
                Ok(RevolveOutcome::Ready(RevolveCommit::Edit(id, edited)))
            }
            Err(FeatureEditPreviewError::UnchangedCut(cause)) => {
                Ok(RevolveOutcome::UnchangedCut(cause))
            }
            Err(cause) => Err(cause.to_string()),
        }
    }

    pub(super) fn commit_revolve(
        &mut self,
        document: u64,
        commit: RevolveCommit,
    ) -> Result<(), String> {
        match commit {
            RevolveCommit::Create(prepared) => {
                let label = if prepared.definition().is_surface() {
                    "Surface Revolution"
                } else {
                    "Revolution"
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
            RevolveCommit::Edit(id, definition) => {
                oblikovati_automation::edit_part_with_checkpoints(
                    &mut self.files,
                    &mut self.part_scenes,
                    document,
                    "Edit Revolution",
                    |snapshot, _| replace_revolve(snapshot, id, definition),
                )
                .map_err(|cause| cause.to_string())?;
            }
        }
        Ok(())
    }
}

pub(super) fn revolve_definition(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
) -> Result<&SketchRevolveDefinition, String> {
    let authored = snapshot
        .work_geometry
        .get(id)
        .map_err(|cause| cause.to_string())?;
    let PartFeatureDefinition::SketchRevolve(definition) = authored.definition() else {
        return Err(format!("feature {id:?}: expected a Revolution"));
    };
    Ok(definition)
}

fn replace_revolve(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: SketchRevolveDefinition,
) -> Result<(), Error> {
    let reads = definition
        .reads(snapshot)
        .map_err(|cause| Error(cause.to_string()))?;
    snapshot
        .work_geometry
        .replace(
            id,
            PartFeatureDefinition::SketchRevolve(Box::new(definition)),
            reads,
        )
        .map_err(|cause| Error(cause.to_string()))
        .map(|_| ())
}
