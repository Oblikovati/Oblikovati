// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and request conversion for Part Surface Ruled Surface (#3918).
use oblikovati_automation::RuledSurfaceInputs;
use oblikovati_gui::{GuiRuledSurfaceEditSeed, GuiRuledSurfaceRequest};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiRuledSurfaceEditSeed> {
    let mut seeds = BTreeMap::new();
    for feature in snapshot.work_geometry.features() {
        let PartFeatureDefinition::RuledSurface(def) = feature.definition() else {
            continue;
        };
        seeds.insert(
            feature.id(),
            GuiRuledSurfaceEditSeed {
                target_body: def.target_body().to_vec(),
                edges: def.edges().to_vec(),
                ruled_type: def.ruled_type(),
                distance: def.distance(),
                angle_deg: def.angle_deg(),
                flip: def.flip(),
                automatic_edge_chain: def.automatic_edge_chain(),
                vector: def.vector().cloned(),
            },
        );
    }
    seeds
}

pub(super) fn request_to_inputs(
    request: &GuiRuledSurfaceRequest,
    target_body: Vec<u8>,
    distance: f64,
    angle_deg: f64,
) -> RuledSurfaceInputs {
    RuledSurfaceInputs {
        target_body,
        edges: request.edges.clone(),
        ruled_type: request.ruled_type,
        distance,
        angle_deg,
        flip: request.flip,
        automatic_edge_chain: request.automatic_edge_chain,
        vector: request.vector.clone(),
    }
}
