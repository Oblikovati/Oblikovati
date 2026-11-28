// SPDX-License-Identifier: GPL-2.0-only
//! Typed v2 Fillet creation and editing (variable radius, continuity, edge sets, corner setbacks).
use crate::PreparedFillet;
use oblikovati_api::Error;
use oblikovati_model::{
    feature::{
        FeatureId,
        fillet::FilletDefinition,
        fillet_options::{
            FilletAdvancedOptions, FilletContinuity, FilletCornerSetback, FilletEdgeSet,
            FilletMode, FilletOptions, FilletVariablePoint,
        },
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeSet;

/// Typed edge-set input for v2 Fillet creation or editing.
#[derive(Clone, Debug, PartialEq)]
pub struct FilletEdgeSetInput {
    pub edges: Vec<Vec<u8>>,
    pub radius: String,
}

/// Typed variable-radius station input for v2 Fillet creation or editing.
#[derive(Clone, Debug, PartialEq)]
pub struct FilletVariablePointInput {
    pub position: f64,
    pub radius: String,
}

/// Typed corner-setback input for v2 Fillet creation or editing.
#[derive(Clone, Debug, PartialEq)]
pub struct FilletCornerSetbackInput {
    pub vertex: Vec<u8>,
    pub minimal: bool,
    pub distances: Vec<String>,
}

/// Complete v1/v2 Fillet inputs shared by preview and commit.
#[derive(Clone, Debug, PartialEq)]
pub struct FilletV2Inputs {
    pub body: Vec<u8>,
    pub mode: FilletMode,
    pub continuity: FilletContinuity,
    pub smooth_radius_transition: bool,
    pub edge_sets: Vec<FilletEdgeSetInput>,
    pub variable_points: Vec<FilletVariablePointInput>,
    pub corner_setback: Option<FilletCornerSetbackInput>,
    pub advanced: FilletAdvancedOptions,
}

/// Build a validated v1/v2 Fillet definition from desktop/automation inputs.
pub fn build_fillet_v2_definition(
    snapshot: &DocumentSnapshot,
    inputs: &FilletV2Inputs,
) -> Result<FilletDefinition, Error> {
    if inputs.edge_sets.is_empty() {
        return Err(Error("fillet: expected at least one edge set".into()));
    }
    let mut all_edges = Vec::new();
    let mut seen = BTreeSet::new();
    let mut parsed_sets = Vec::with_capacity(inputs.edge_sets.len());
    for set in &inputs.edge_sets {
        if set.edges.is_empty() {
            return Err(Error("fillet edge set: expected at least one edge".into()));
        }
        let radius = crate::extrusion::positive_length(snapshot, &set.radius, "fillet radius")?;
        let mut set_edges = Vec::new();
        for edge in &set.edges {
            if seen.insert(edge.clone()) {
                all_edges.push(edge.clone());
                set_edges.push(edge.clone());
            }
        }
        if !set_edges.is_empty() {
            parsed_sets.push(FilletEdgeSet {
                edges: set_edges,
                radius,
            });
        }
    }
    if all_edges.is_empty() {
        return Err(Error("fillet: expected at least one edge".into()));
    }
    let primary_radius = parsed_sets[0].radius;
    let mut variable_points = Vec::with_capacity(inputs.variable_points.len());
    for pt in &inputs.variable_points {
        let radius =
            crate::extrusion::positive_length(snapshot, &pt.radius, "fillet variable radius")?;
        variable_points.push(FilletVariablePoint {
            position: pt.position,
            radius,
        });
    }
    let corner_setback = match &inputs.corner_setback {
        None => None,
        Some(sb) => {
            let mut distances = Vec::with_capacity(sb.distances.len());
            for dist in &sb.distances {
                distances.push(crate::extrusion::positive_length(
                    snapshot,
                    dist,
                    "fillet setback distance",
                )?);
            }
            Some(FilletCornerSetback {
                vertex: sb.vertex.clone(),
                minimal: sb.minimal,
                distances,
            })
        }
    };
    let edge_sets = if parsed_sets.len() > 1 {
        parsed_sets
    } else {
        Vec::new()
    };
    let options = FilletOptions {
        mode: inputs.mode,
        continuity: inputs.continuity,
        smooth_radius_transition: inputs.smooth_radius_transition,
        edge_sets,
        variable_points,
        corner_setback,
        advanced: inputs.advanced,
    };
    let definition =
        FilletDefinition::new_with_options(inputs.body.clone(), all_edges, primary_radius, options)
            .map_err(Error)?;
    definition.resolve(&snapshot.parameters).map_err(Error)?;
    Ok(definition)
}

/// Prepare a new v1/v2 Fillet feature without mutating the document.
pub fn prepare_fillet_v2(
    snapshot: &DocumentSnapshot,
    inputs: &FilletV2Inputs,
) -> Result<PreparedFillet, Error> {
    let definition = build_fillet_v2_definition(snapshot, inputs)?;
    Ok(PreparedFillet::from_parts(
        crate::fillet::next_name(snapshot, "Fillet")?,
        definition,
    ))
}

/// Replace an existing Fillet feature's definition inside a transaction/preview candidate.
pub fn replace_fillet(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: FilletDefinition,
) -> Result<(), Error> {
    snapshot
        .edit_fillet(id, definition)
        .map_err(|cause| Error(cause.to_string()))
}

/// Evaluate a positive length expression in working units (cm) for Fillet probing.
pub fn evaluate_positive_cm(snapshot: &DocumentSnapshot, expr: &str) -> Result<f64, Error> {
    let len = crate::extrusion::positive_length(snapshot, expr, "fillet radius")?;
    len.resolve(&snapshot.parameters)
        .map_err(|cause| Error(cause.to_string()))
}
