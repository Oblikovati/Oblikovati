// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and automation-input conversion for Part Modify Combine (#3918).
use oblikovati_automation::CombineInputs;
use oblikovati_gui::{GuiCombineEditSeed, GuiCombineRequest};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(snapshot: &DocumentSnapshot) -> BTreeMap<FeatureId, GuiCombineEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Combine(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            Some((
                feature.id(),
                GuiCombineEditSeed {
                    base_body: recipe.base_body,
                    tool_bodies: recipe.tool_bodies,
                    operation: recipe.operation,
                    keep_toolbodies: recipe.keep_toolbodies,
                },
            ))
        })
        .collect()
}

pub(super) fn request_to_inputs(request: &GuiCombineRequest, base_body: Vec<u8>) -> CombineInputs {
    CombineInputs {
        base_body,
        tool_bodies: request.tool_bodies.clone(),
        operation: request.operation,
        keep_toolbodies: request.keep_toolbodies,
    }
}

pub(super) fn decode_hex_key(hex: &str) -> Result<Vec<u8>, String> {
    if !hex.len().is_multiple_of(2) {
        return Err(format!(
            "combine body key {hex:?}: expected even hex length"
        ));
    }
    hex.as_bytes()
        .as_chunks::<2>()
        .0
        .iter()
        .map(|pair| {
            std::str::from_utf8(pair)
                .ok()
                .and_then(|text| u8::from_str_radix(text, 16).ok())
                .ok_or_else(|| format!("combine body key {hex:?}: expected hex bytes"))
        })
        .collect()
}
