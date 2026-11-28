// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and automation-input conversion for Part Modify Chamfer v2 (#4046).
use oblikovati_automation::ChamferV2Inputs;
use oblikovati_gui::{GuiChamferPartialState, GuiChamferV2EditSeed, GuiChamferV2Request};
use oblikovati_model::{
    feature::{
        FeatureId, chamfer_options::ChamferPartialSpec, part::PartFeatureDefinition,
        work_geometry::DatumLength,
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(
    snapshot: &DocumentSnapshot,
    generation: u64,
) -> BTreeMap<FeatureId, GuiChamferV2EditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Chamfer(definition) = feature.definition() else {
                return None;
            };
            let opts = definition.options();
            let distance = length_text(snapshot, definition.distance());
            let distance2 = opts
                .distance2
                .map_or_else(|| "2 mm".into(), |d| length_text(snapshot, d));
            let angle = format_deg(opts.angle_deg);
            let partial = opts.partial.map(|p| GuiChamferPartialState {
                edge_length_cm: p.edge_length_cm,
                to_start: format_mm(p.to_start_cm * 10.0),
                chamfer_length: format_mm(p.chamfer_length_cm * 10.0),
                to_end: format_mm(p.to_end_cm * 10.0),
                driven: p.driven,
            });
            Some((
                feature.id(),
                GuiChamferV2EditSeed {
                    generation,
                    body: definition.body().to_vec(),
                    edges: definition.edges().to_vec(),
                    mode: opts.mode,
                    distance,
                    distance2,
                    angle,
                    reference_face: opts.reference_face.clone(),
                    reversed: opts.reversed,
                    edge_chain_second: opts.edge_chain_second,
                    setback_second: opts.setback_second,
                    partial,
                },
            ))
        })
        .collect()
}

pub(super) fn request_to_inputs(
    request: &GuiChamferV2Request,
    body: Vec<u8>,
) -> Result<ChamferV2Inputs, String> {
    let angle_deg = parse_angle_deg(&request.angle)?;
    let partial = match &request.partial {
        None => None,
        Some(p) => {
            let to_start_cm = parse_non_negative_cm(&p.to_start)?;
            let chamfer_length_cm = parse_non_negative_cm(&p.chamfer_length)?;
            let to_end_cm = parse_non_negative_cm(&p.to_end)?;
            Some(ChamferPartialSpec {
                edge_length_cm: p.edge_length_cm,
                to_start_cm,
                chamfer_length_cm,
                to_end_cm,
                driven: p.driven,
            })
        }
    };
    Ok(ChamferV2Inputs {
        body,
        edges: request.edges.clone(),
        mode: request.mode,
        distance: request.distance.clone(),
        distance2: Some(request.distance2.clone()),
        angle_deg,
        reference_face: request.reference_face.clone(),
        reversed: request.reversed,
        edge_chain_second: request.edge_chain_second,
        setback_second: request.setback_second,
        partial,
    })
}

fn length_text(snapshot: &DocumentSnapshot, length: DatumLength) -> String {
    match length {
        DatumLength::Constant(cm) => format_mm(cm * 10.0),
        DatumLength::Parameter(id) => snapshot
            .parameters
            .get(id)
            .map(|p| p.name.clone())
            .unwrap_or_else(|| "1 mm".into()),
    }
}

fn format_mm(mm: f64) -> String {
    let rounded = (mm * 100.0).round() / 100.0;
    if (rounded - rounded.round()).abs() < 1e-6 {
        format!("{} mm", rounded.round() as i64)
    } else {
        format!("{rounded:.2} mm")
    }
}

fn format_deg(deg: f64) -> String {
    let rounded = (deg * 100.0).round() / 100.0;
    if (rounded - rounded.round()).abs() < 1e-6 {
        format!("{} deg", rounded.round() as i64)
    } else {
        format!("{rounded:.2} deg")
    }
}

fn parse_angle_deg(text: &str) -> Result<f64, String> {
    let trimmed = text.trim().replace(',', ".");
    let num = trimmed
        .strip_suffix("deg")
        .or_else(|| trimmed.strip_suffix('°'))
        .unwrap_or(&trimmed)
        .trim();
    let val: f64 = num
        .parse()
        .map_err(|_| format!("chamfer angle {text:?}: expected a finite angle in degrees"))?;
    if !val.is_finite() || val <= 0.0 || val >= 90.0 {
        return Err(format!(
            "chamfer angle {val}: expected angle in (0, 90) degrees"
        ));
    }
    Ok(val)
}

fn parse_non_negative_cm(text: &str) -> Result<f64, String> {
    let trimmed = text.trim().replace(',', ".");
    let (num, factor) = if let Some(s) = trimmed.strip_suffix("mm") {
        (s.trim(), 0.1)
    } else if let Some(s) = trimmed.strip_suffix("cm") {
        (s.trim(), 1.0)
    } else if let Some(s) = trimmed.strip_suffix("in") {
        (s.trim(), 2.54)
    } else {
        (trimmed.as_str(), 0.1)
    };
    let val: f64 = num
        .parse()
        .map_err(|_| format!("chamfer length {text:?}: expected a non-negative length"))?;
    if !val.is_finite() || val < 0.0 {
        return Err(format!(
            "chamfer length {text:?}: expected a non-negative length"
        ));
    }
    Ok(val * factor)
}
