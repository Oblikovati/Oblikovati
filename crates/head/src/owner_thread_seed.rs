// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and automation-input conversion for Part Modify Thread (#3918).
use oblikovati_automation::ThreadFeatureInputs;
use oblikovati_gui::{GuiThreadEditSeed, GuiThreadRequest};
use oblikovati_martensite::{EvaluationFailure, NamedSolid};
use oblikovati_model::{
    feature::{
        FeatureId,
        part::{PartFeatureDefinition, PartFeatureEvaluation},
        work_geometry::DatumLength,
    },
    history::{DocumentEditHistory, DocumentSnapshot},
    identity::DocumentGuid,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(
    snapshot: &DocumentSnapshot,
    evaluation: Option<&PartFeatureEvaluation<NamedSolid, EvaluationFailure>>,
    history: &DocumentEditHistory,
    guid: DocumentGuid,
) -> BTreeMap<FeatureId, GuiThreadEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Thread(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            let face_info = evaluation.and_then(|eval| {
                eval.ordered_bodies().find_map(|body| {
                    body.threadable_face_info(history, guid, &recipe.face)
                        .ok()
                        .flatten()
                })
            });
            let (measured_depth_mm, face_conical, face_internal) = match face_info {
                Some(info) => (info.full_depth_cm * 10.0, info.conical, info.internal),
                None => {
                    let fallback = recipe
                        .depth
                        .and_then(|len| len.resolve(&snapshot.parameters).ok())
                        .unwrap_or(1.0)
                        * 10.0;
                    (fallback, false, true)
                }
            };
            let depth = if recipe.full_depth {
                format!("{measured_depth_mm:.3} mm")
            } else if let Some(len) = recipe.depth {
                length_text(snapshot, len)
            } else {
                format!("{measured_depth_mm:.3} mm")
            };
            let offset = recipe
                .offset
                .map(|len| length_text(snapshot, len))
                .unwrap_or_else(|| "0 mm".into());
            Some((
                feature.id(),
                GuiThreadEditSeed {
                    body: recipe.body,
                    face: recipe.face,
                    full_depth: recipe.full_depth,
                    measured_depth_mm,
                    depth,
                    offset,
                    display_in_model: recipe.display_in_model,
                    spec: recipe.spec,
                    face_conical,
                    face_internal,
                },
            ))
        })
        .collect()
}

pub(super) fn request_to_inputs(
    request: &GuiThreadRequest,
    body: Vec<u8>,
    face: Vec<u8>,
) -> ThreadFeatureInputs {
    ThreadFeatureInputs {
        body,
        face,
        full_depth: request.full_depth,
        depth: Some(request.depth.clone()),
        offset: Some(request.offset.clone()),
        display_in_model: request.display_in_model,
        spec: request.spec.clone(),
    }
}

fn length_text(snapshot: &DocumentSnapshot, length: DatumLength) -> String {
    if let DatumLength::Parameter(id) = length
        && let Some(param) = snapshot.parameters.get(id)
    {
        return param.name.clone();
    }
    let cm = length.resolve(&snapshot.parameters).unwrap_or(0.0);
    let mm = cm * 10.0;
    let rounded = (mm * 1000.0).round() / 1000.0;
    if rounded.fract().abs() < 1e-6 {
        format!("{} mm", rounded as i64)
    } else if (rounded * 10.0).fract().abs() < 1e-6 {
        format!("{rounded:.1} mm")
    } else if (rounded * 100.0).fract().abs() < 1e-6 {
        format!("{rounded:.2} mm")
    } else {
        format!("{rounded:.3} mm")
    }
}
