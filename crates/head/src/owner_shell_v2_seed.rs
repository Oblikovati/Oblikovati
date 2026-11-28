// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and automation-input conversion for Part Modify Shell v2 (#4106).
use oblikovati_automation::{ShellV2Inputs, ShellV2OverrideInput};
use oblikovati_gui::{GuiShellOverrideSet, GuiShellV2EditSeed, GuiShellV2Request};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition, work_geometry::DatumLength},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(
    snapshot: &DocumentSnapshot,
    generation: u64,
) -> BTreeMap<FeatureId, GuiShellV2EditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Shell(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            let thickness = length_text(snapshot, recipe.thickness);
            let overrides = recipe
                .face_thicknesses
                .iter()
                .map(|entry| GuiShellOverrideSet {
                    face: Some(entry.face.clone()),
                    thickness: length_text(snapshot, entry.thickness),
                })
                .collect();
            Some((
                feature.id(),
                GuiShellV2EditSeed {
                    generation,
                    body: recipe.body,
                    openings: recipe.openings,
                    thickness,
                    direction: recipe.direction,
                    overrides,
                    options: recipe.options,
                },
            ))
        })
        .collect()
}

pub(super) fn request_to_inputs(request: &GuiShellV2Request, body: Vec<u8>) -> ShellV2Inputs {
    let overrides = request
        .overrides
        .iter()
        .map(|(face, thickness)| ShellV2OverrideInput {
            face: face.clone(),
            thickness: thickness.clone(),
        })
        .collect();
    ShellV2Inputs {
        body,
        openings: request.openings.clone(),
        thickness: request.thickness.clone(),
        direction: request.direction,
        overrides,
        options: request.options,
    }
}

fn length_text(snapshot: &DocumentSnapshot, length: DatumLength) -> String {
    match length {
        DatumLength::Constant(cm) => format_mm(cm * 10.0),
        DatumLength::Parameter(id) => snapshot
            .parameters
            .get(id)
            .map(|p| p.name.clone())
            .unwrap_or_else(|| "1 mm".into()),
    }
}

fn format_mm(mm: f64) -> String {
    let rounded = (mm * 100.0).round() / 100.0;
    if (rounded - rounded.round()).abs() < 1e-6 {
        format!("{} mm", rounded.round() as i64)
    } else {
        format!("{rounded:.2} mm")
    }
}
