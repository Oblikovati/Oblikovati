// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and request conversion for Surface Boundary Patch (#3918).
use oblikovati_automation::BoundaryPatchInputs;
use oblikovati_gui::{GuiBoundaryPatchEditSeed, GuiBoundaryPatchRequest};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(
    snapshot: &DocumentSnapshot,
) -> BTreeMap<FeatureId, GuiBoundaryPatchEditSeed> {
    let mut seeds = BTreeMap::new();
    for feature in snapshot.work_geometry.features() {
        let PartFeatureDefinition::BoundaryPatch(definition) = feature.definition() else {
            continue;
        };
        seeds.insert(
            feature.id(),
            GuiBoundaryPatchEditSeed {
                automatic_edge_chain: definition.automatic_edge_chain(),
                edge_sets: definition.edge_sets().to_vec(),
                guide_rails: definition.guide_rails().to_vec(),
            },
        );
    }
    seeds
}

pub(super) fn request_to_inputs(request: &GuiBoundaryPatchRequest) -> BoundaryPatchInputs {
    BoundaryPatchInputs {
        automatic_edge_chain: request.automatic_edge_chain,
        edge_sets: request.edge_sets.clone(),
        guide_rails: request.guide_rails.clone(),
    }
}
