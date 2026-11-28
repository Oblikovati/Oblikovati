// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and automation-input conversion for Part Modify Hole (#4093).
use oblikovati_automation::{HoleFeatureInputs, HoleSeatInput, HoleTerminationInput};
use oblikovati_gui::{GuiHoleEditSeed, GuiHoleRequest, GuiHoleSeatKind, GuiHoleTermination};
use oblikovati_model::{
    feature::{
        FeatureId,
        hole::{HoleDrillPoint, HoleExtent, HolePosition, HoleSeat},
        part::PartFeatureDefinition,
        work_geometry::DatumLength,
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(snapshot: &DocumentSnapshot) -> BTreeMap<FeatureId, GuiHoleEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Hole(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            let diameter = length_text(snapshot, recipe.diameter);
            let (termination, depth, termination_surface) = match &recipe.extent {
                HoleExtent::ThroughAll => (GuiHoleTermination::ThroughAll, "8 mm".into(), None),
                HoleExtent::Distance(len) => (
                    GuiHoleTermination::Distance,
                    length_text(snapshot, *len),
                    None,
                ),
                HoleExtent::ToSurface(surface) => (
                    GuiHoleTermination::ToSurface,
                    "8 mm".into(),
                    Some(surface.clone()),
                ),
            };
            let (seat, seat_diameter, seat_depth, seat_angle) = match &recipe.options.seat {
                HoleSeat::None => (
                    GuiHoleSeatKind::None,
                    diameter.clone(),
                    "2 mm".into(),
                    "90 deg".into(),
                ),
                HoleSeat::Counterbore { diameter, depth } => (
                    GuiHoleSeatKind::Counterbore,
                    length_text(snapshot, *diameter),
                    length_text(snapshot, *depth),
                    "90 deg".into(),
                ),
                HoleSeat::Spotface { diameter, depth } => (
                    GuiHoleSeatKind::Spotface,
                    length_text(snapshot, *diameter),
                    length_text(snapshot, *depth),
                    "90 deg".into(),
                ),
                HoleSeat::Countersink {
                    diameter,
                    angle_deg,
                } => (
                    GuiHoleSeatKind::Countersink,
                    length_text(snapshot, *diameter),
                    "2 mm".into(),
                    format_angle_deg(*angle_deg),
                ),
            };
            let (drill_point_angled, drill_point_angle) = match recipe.options.drill_point {
                HoleDrillPoint::Flat => (false, "118 deg".into()),
                HoleDrillPoint::Angle(deg) => (true, format_angle_deg(deg)),
            };
            let position = match recipe.position {
                HolePosition::FaceCentroid => None,
                HolePosition::ModelPoint(coords) => {
                    let x = coords[0].resolve(&snapshot.parameters).ok()?;
                    let y = coords[1].resolve(&snapshot.parameters).ok()?;
                    let z = coords[2].resolve(&snapshot.parameters).ok()?;
                    Some([x, y, z])
                }
            };
            Some((
                feature.id(),
                GuiHoleEditSeed {
                    body: recipe.body.clone(),
                    face: recipe.face.clone(),
                    position,
                    kind: recipe.options.kind,
                    seat,
                    diameter,
                    depth,
                    seat_diameter,
                    seat_depth,
                    seat_angle,
                    termination,
                    termination_surface,
                    drill_point_angled,
                    drill_point_angle,
                    direction_flipped: recipe.options.direction_flipped,
                    extend_start: recipe.options.extend_start,
                    fastener: recipe.options.fastener.clone(),
                    thread: recipe.options.thread.clone(),
                },
            ))
        })
        .collect()
}

pub(super) fn request_to_inputs(
    request: &GuiHoleRequest,
    body: Vec<u8>,
    face: Vec<u8>,
) -> Result<HoleFeatureInputs, String> {
    let termination = match request.termination {
        GuiHoleTermination::ThroughAll => HoleTerminationInput::ThroughAll,
        GuiHoleTermination::Distance => HoleTerminationInput::Distance(request.depth.clone()),
        GuiHoleTermination::ToSurface => {
            let surface = request
                .termination_surface
                .clone()
                .ok_or("hole: expected a terminating surface")?;
            HoleTerminationInput::ToSurface(surface)
        }
    };
    let seat = match request.seat {
        GuiHoleSeatKind::None => HoleSeatInput::None,
        GuiHoleSeatKind::Counterbore => HoleSeatInput::Counterbore {
            diameter: request.seat_diameter.clone(),
            depth: request.seat_depth.clone(),
        },
        GuiHoleSeatKind::Spotface => HoleSeatInput::Spotface {
            diameter: request.seat_diameter.clone(),
            depth: request.seat_depth.clone(),
        },
        GuiHoleSeatKind::Countersink => HoleSeatInput::Countersink {
            diameter: request.seat_diameter.clone(),
            angle_deg: parse_angle(&request.seat_angle)?,
        },
    };
    let drill_point = if request.drill_point_angled {
        HoleDrillPoint::Angle(parse_angle(&request.drill_point_angle)?)
    } else {
        HoleDrillPoint::Flat
    };
    Ok(HoleFeatureInputs {
        body,
        face,
        position: request.position,
        diameter: request.diameter.clone(),
        termination,
        kind: request.kind,
        seat,
        drill_point,
        direction_flipped: request.direction_flipped,
        extend_start: request.extend_start,
        fastener: request.fastener.clone(),
        thread: request.thread.clone(),
    })
}

fn length_text(snapshot: &DocumentSnapshot, length: DatumLength) -> String {
    if let DatumLength::Parameter(id) = length
        && let Some(param) = snapshot.parameters.get(id)
    {
        return param.name.clone();
    }
    let cm = length.resolve(&snapshot.parameters).unwrap_or(0.4);
    let mm = cm * 10.0;
    let rounded = (mm * 100.0).round() / 100.0;
    if (rounded.fract()).abs() < 1e-6 {
        format!("{} mm", rounded as i64)
    } else if ((rounded * 10.0).fract()).abs() < 1e-6 {
        format!("{rounded:.1} mm")
    } else {
        format!("{rounded:.2} mm")
    }
}

fn format_angle_deg(deg: f64) -> String {
    let rounded = (deg * 100.0).round() / 100.0;
    if rounded.fract().abs() < 1e-6 {
        format!("{} deg", rounded as i64)
    } else {
        format!("{rounded:.1} deg")
    }
}

fn parse_angle(raw: &str) -> Result<f64, String> {
    let cleaned = raw
        .trim()
        .replace(',', ".")
        .replace('\u{00b0}', "")
        .replace("deg", "");
    let value: f64 = cleaned
        .trim()
        .parse()
        .map_err(|_| format!("hole angle {raw:?}: expected a number in degrees"))?;
    if !value.is_finite() || value <= 0.0 || value >= 180.0 {
        return Err(format!("hole angle {value}: expected angle in (0, 180)"));
    }
    Ok(value)
}
