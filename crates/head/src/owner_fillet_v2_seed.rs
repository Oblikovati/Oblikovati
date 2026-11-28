// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and automation-input conversion for Part Modify Fillet v2.
use oblikovati_automation::{
    FilletCornerSetbackInput, FilletEdgeSetInput, FilletV2Inputs, FilletVariablePointInput,
};
use oblikovati_gui::{
    GuiFilletCornerSetback, GuiFilletEdgeSet, GuiFilletV2EditSeed, GuiFilletV2Request,
    GuiFilletVariablePoint,
};
use oblikovati_model::{
    feature::{FeatureId, part::PartFeatureDefinition, work_geometry::DatumLength},
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(
    snapshot: &DocumentSnapshot,
    generation: u64,
) -> BTreeMap<FeatureId, GuiFilletV2EditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Fillet(definition) = feature.definition() else {
                return None;
            };
            let opts = definition.options();
            let primary_radius = length_text(snapshot, definition.radius());
            let edge_sets = if opts.edge_sets.is_empty() {
                vec![GuiFilletEdgeSet {
                    edges: definition.edges().to_vec(),
                    radius: primary_radius,
                    is_loop: false,
                }]
            } else {
                opts.edge_sets
                    .iter()
                    .map(|set| GuiFilletEdgeSet {
                        edges: set.edges.clone(),
                        radius: length_text(snapshot, set.radius),
                        is_loop: false,
                    })
                    .collect()
            };
            let variable_points = opts
                .variable_points
                .iter()
                .map(|pt| GuiFilletVariablePoint {
                    position: format_pos(pt.position),
                    radius: length_text(snapshot, pt.radius),
                })
                .collect();
            let corner_setback = opts
                .corner_setback
                .as_ref()
                .map(|sb| GuiFilletCornerSetback {
                    vertex: sb.vertex.clone(),
                    minimal: sb.minimal,
                    distances: sb
                        .distances
                        .iter()
                        .map(|d| length_text(snapshot, *d))
                        .collect(),
                });
            Some((
                feature.id(),
                GuiFilletV2EditSeed {
                    generation,
                    body: definition.body().to_vec(),
                    mode: opts.mode,
                    continuity: opts.continuity,
                    smooth_radius_transition: opts.smooth_radius_transition,
                    edge_sets,
                    variable_points,
                    corner_setback,
                    advanced: opts.advanced,
                },
            ))
        })
        .collect()
}

pub(super) fn request_to_inputs(
    request: &GuiFilletV2Request,
    body: Vec<u8>,
) -> Result<FilletV2Inputs, String> {
    let edge_sets = request
        .edge_sets
        .iter()
        .filter(|set| !set.edges.is_empty())
        .map(|set| FilletEdgeSetInput {
            edges: set.edges.clone(),
            radius: set.radius.clone(),
        })
        .collect();
    let mut variable_points = Vec::with_capacity(request.variable_points.len());
    for pt in &request.variable_points {
        let pos: f64 = pt.position.trim().replace(',', ".").parse().map_err(|_| {
            format!(
                "fillet variable position {:?}: expected number",
                pt.position
            )
        })?;
        variable_points.push(FilletVariablePointInput {
            position: pos,
            radius: pt.radius.clone(),
        });
    }
    let corner_setback = request
        .corner_setback
        .as_ref()
        .map(|sb| FilletCornerSetbackInput {
            vertex: sb.vertex.clone(),
            minimal: sb.minimal,
            distances: sb.distances.clone(),
        });
    Ok(FilletV2Inputs {
        body,
        mode: request.mode,
        continuity: request.continuity,
        smooth_radius_transition: request.smooth_radius_transition,
        edge_sets,
        variable_points,
        corner_setback,
        advanced: request.advanced,
    })
}

fn length_text(snapshot: &DocumentSnapshot, length: DatumLength) -> String {
    if let DatumLength::Parameter(id) = length
        && let Some(param) = snapshot.parameters.get(id)
    {
        return param.name.clone();
    }
    let cm = length.resolve(&snapshot.parameters).unwrap_or(0.2);
    let mm = cm * 10.0;
    let rounded = (mm * 100.0).round() / 100.0;
    if rounded.fract().abs() < 1e-6 {
        format!("{} mm", rounded as i64)
    } else if (rounded * 10.0).fract().abs() < 1e-6 {
        format!("{rounded:.1} mm")
    } else {
        format!("{rounded:.2} mm")
    }
}

fn format_pos(pos: f64) -> String {
    let rounded = (pos * 100.0).round() / 100.0;
    if rounded.fract().abs() < 1e-6 {
        format!("{rounded:.1}")
    } else {
        format!("{rounded}")
    }
}
