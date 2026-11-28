// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and automation-input conversion for Part Surface Stitch (#3918).
use oblikovati_automation::StitchInputs;
use oblikovati_gui::{GuiStitchEditSeed, GuiStitchFreeEdge, GuiStitchRequest};
use oblikovati_martensite::StitchAnalysisReport;
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(snapshot: &DocumentSnapshot) -> BTreeMap<FeatureId, GuiStitchEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Stitch(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            Some((
                feature.id(),
                GuiStitchEditSeed {
                    surfaces: recipe.surfaces,
                    maintain_as_surface: recipe.maintain_as_surface,
                    maximum_tolerance_mm: recipe.maximum_tolerance_mm,
                    show_edge_conditions: recipe.show_edge_conditions,
                    show_near_tangent: recipe.show_near_tangent,
                },
            ))
        })
        .collect()
}

pub(super) fn request_to_inputs(
    request: &GuiStitchRequest,
    maximum_tolerance_mm: f64,
) -> StitchInputs {
    StitchInputs {
        surfaces: request.surfaces.clone(),
        maintain_as_surface: request.maintain_as_surface,
        maximum_tolerance_mm,
        show_edge_conditions: request.show_edge_conditions,
        show_near_tangent: request.show_near_tangent,
    }
}

pub(super) fn free_edges_to_gui(report: &StitchAnalysisReport) -> Vec<GuiStitchFreeEdge> {
    report
        .free_edges
        .iter()
        .map(|edge| GuiStitchFreeEdge {
            index: edge.index,
            label: format!("Edge {}", edge.index),
            length_mm: edge.length_mm,
            max_gap_mm: edge.max_gap_mm,
        })
        .collect()
}

pub(super) fn decode_hex_key(hex: &str) -> Option<Vec<u8>> {
    if !hex.len().is_multiple_of(2) {
        return None;
    }
    let mut out = Vec::with_capacity(hex.len() / 2);
    for pair in hex.as_bytes().as_chunks::<2>().0 {
        let text = std::str::from_utf8(pair).ok()?;
        let byte = u8::from_str_radix(text, 16).ok()?;
        out.push(byte);
    }
    Some(out)
}
