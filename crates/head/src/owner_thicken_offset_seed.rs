// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and automation-input conversion for Part Modify Thicken/Offset (#3918).
use oblikovati_automation::ThickenOffsetInputs;
use oblikovati_gui::{GuiThickenOffsetEditSeed, GuiThickenOffsetRequest};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiThickenOffsetEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::ThickenOffset(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            Some((
                feature.id(),
                GuiThickenOffsetEditSeed {
                    target_body: recipe.target_body,
                    faces: recipe.faces,
                    output: recipe.output,
                    distance: recipe.distance,
                    direction: recipe.direction,
                    boolean: recipe.boolean,
                    automatic_face_chain: recipe.automatic_face_chain,
                    automatic_blending: recipe.automatic_blending,
                    allow_approximation: recipe.allow_approximation,
                    approximation_type: recipe.approximation_type,
                    approximation_tolerance: recipe.approximation_tolerance,
                    approximation_percent: recipe.approximation_percent,
                },
            ))
        })
        .collect()
}

pub(super) fn request_to_inputs(
    request: &GuiThickenOffsetRequest,
    target_body: Vec<u8>,
    distance: f64,
) -> ThickenOffsetInputs {
    ThickenOffsetInputs {
        target_body,
        faces: request.faces.clone(),
        output: request.output,
        distance,
        direction: request.direction,
        boolean: request.boolean,
        automatic_face_chain: request.automatic_face_chain,
        automatic_blending: request.automatic_blending,
        allow_approximation: request.allow_approximation,
        approximation_type: request.approximation_type,
        approximation_tolerance: request.approximation_tolerance,
        approximation_percent: request.approximation_percent,
    }
}
