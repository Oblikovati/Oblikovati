// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and automation-input conversion for Part Surface Extend (#3918).
use oblikovati_automation::ExtendSurfaceInputs;
use oblikovati_gui::{GuiExtendSurfaceEditSeed, GuiExtendSurfaceRequest};
use oblikovati_model::{
    feature::{FeatureId, extend_surface::ExtendSurfaceExtent, part::PartFeatureDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiExtendSurfaceEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::ExtendSurface(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            Some((
                feature.id(),
                GuiExtendSurfaceEditSeed {
                    target_body: recipe.target_body,
                    edges: recipe.edges,
                    edge_chain: recipe.edge_chain,
                    extent: recipe.extent,
                    method: recipe.method,
                },
            ))
        })
        .collect()
}

pub(super) fn request_to_inputs(
    request: &GuiExtendSurfaceRequest,
    target_body: Vec<u8>,
    extent: ExtendSurfaceExtent,
) -> ExtendSurfaceInputs {
    ExtendSurfaceInputs {
        target_body,
        edges: request.edges.clone(),
        edge_chain: request.edge_chain,
        extent,
        method: request.method,
    }
}
