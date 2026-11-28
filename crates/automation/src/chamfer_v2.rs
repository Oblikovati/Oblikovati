// SPDX-License-Identifier: GPL-2.0-only
//! Typed v2 Chamfer creation and editing (equal-distance, distance-angle, two-distances, partial edges, #4046).
use crate::PreparedChamfer;
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        chamfer::ChamferDefinition,
        chamfer_options::{ChamferMode, ChamferOptions, ChamferPartialSpec},
    },
    history::DocumentSnapshot,
};

/// Complete v1/v2 Chamfer inputs shared by preview and commit.
#[derive(Clone, Debug, PartialEq)]
pub struct ChamferV2Inputs {
    pub body: Vec<u8>,
    pub edges: Vec<Vec<u8>>,
    pub mode: ChamferMode,
    pub distance: String,
    pub distance2: Option<String>,
    pub angle_deg: f64,
    pub reference_face: Option<Vec<u8>>,
    pub reversed: bool,
    pub edge_chain_second: bool,
    pub setback_second: bool,
    pub partial: Option<ChamferPartialSpec>,
}

/// Build a validated v1/v2 Chamfer definition from desktop/automation inputs.
pub fn build_chamfer_v2_definition(
    snapshot: &DocumentSnapshot,
    inputs: &ChamferV2Inputs,
) -> Result<ChamferDefinition, Error> {
    if inputs.edges.is_empty() {
        return Err(Error("chamfer: expected at least one edge".into()));
    }
    let distance =
        crate::extrusion::positive_length(snapshot, &inputs.distance, "chamfer distance")?;
    let distance2 = match inputs.mode {
        ChamferMode::TwoDistances => {
            let expr = inputs
                .distance2
                .as_deref()
                .ok_or_else(|| Error("chamfer two-distance: expected Distance2".into()))?;
            Some(crate::extrusion::positive_length(
                snapshot,
                expr,
                "chamfer Distance2",
            )?)
        }
        _ => None,
    };
    let options = ChamferOptions {
        mode: inputs.mode,
        distance2,
        angle_deg: inputs.angle_deg,
        reference_face: inputs.reference_face.clone(),
        reversed: inputs.reversed,
        edge_chain_second: inputs.edge_chain_second,
        setback_second: inputs.setback_second,
        partial: inputs.partial,
    };
    ChamferDefinition::with_options(inputs.body.clone(), inputs.edges.clone(), distance, options)
        .map_err(Error)
}

/// Prepare a validated v2 Chamfer draft for creation.
pub fn prepare_chamfer_v2(
    snapshot: &DocumentSnapshot,
    inputs: &ChamferV2Inputs,
) -> Result<PreparedChamfer, Error> {
    let definition = build_chamfer_v2_definition(snapshot, inputs)?;
    let name = crate::fillet::next_name(snapshot, "Chamfer")?;
    Ok(PreparedChamfer::from_parts(name, definition))
}

/// Replace an existing Chamfer feature in-place inside a candidate or transaction.
pub fn replace_chamfer(
    candidate: &mut DocumentSnapshot,
    id: FeatureId,
    definition: ChamferDefinition,
) -> Result<(), Error> {
    candidate
        .edit_chamfer(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}
