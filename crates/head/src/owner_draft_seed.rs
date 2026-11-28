// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and automation-input conversion for Part Modify Draft (#3918).
use oblikovati_automation::DraftFeatureInputs;
use oblikovati_gui::{GuiDraftEditSeed, GuiDraftRequest};
use oblikovati_model::{
    feature::{FeatureId, draft::validate_angle_deg, part::PartFeatureDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(snapshot: &DocumentSnapshot) -> BTreeMap<FeatureId, GuiDraftEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Draft(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            Some((
                feature.id(),
                GuiDraftEditSeed {
                    body: recipe.body,
                    mode: recipe.mode,
                    pull_direction: recipe.pull_direction,
                    parting_tool: recipe.parting_tool,
                    faces: recipe.faces,
                    angle: format_angle_deg(recipe.angle_deg),
                    angle_mode: recipe.angle_mode,
                    direction_flipped: recipe.direction_flipped,
                    automatic_blending: recipe.automatic_blending,
                },
            ))
        })
        .collect()
}

pub(super) fn request_to_inputs(
    request: &GuiDraftRequest,
    body: Vec<u8>,
    pull_direction: Vec<u8>,
) -> Result<DraftFeatureInputs, String> {
    let angle_deg = parse_draft_angle(&request.angle)?;
    validate_angle_deg(angle_deg)?;
    Ok(DraftFeatureInputs {
        body,
        mode: request.mode,
        pull_direction,
        parting_tool: request.parting_tool.clone(),
        faces: request.faces.clone(),
        angle_deg,
        angle_mode: request.angle_mode,
        direction_flipped: request.direction_flipped,
        automatic_blending: request.automatic_blending,
    })
}

fn format_angle_deg(deg: f64) -> String {
    let rounded = (deg * 100.0).round() / 100.0;
    if rounded.fract().abs() < 1e-6 {
        format!("{} deg", rounded as i64)
    } else if (rounded * 10.0).fract().abs() < 1e-6 {
        format!("{rounded:.1} deg")
    } else {
        format!("{rounded:.2} deg")
    }
}

fn parse_draft_angle(raw: &str) -> Result<f64, String> {
    let cleaned = raw
        .trim()
        .replace(',', ".")
        .replace('\u{00b0}', "")
        .replace("deg", "");
    let value: f64 = cleaned.trim().parse().map_err(|_| {
        format!("draft angle {raw:?}: expected degrees; errors during the part update")
    })?;
    if !value.is_finite() {
        return Err(format!(
            "draft angle {value}: expected finite degrees; errors during the part update"
        ));
    }
    Ok(value)
}
