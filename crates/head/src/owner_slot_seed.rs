// SPDX-License-Identifier: GPL-2.0-only
//! Edit-seed hydration and automation-input conversion for Part Modify Slot.
use oblikovati_automation::{HoleTerminationInput, SlotFeatureInputs, SlotSeatInput};
use oblikovati_gui::{GuiHoleSeatKind, GuiHoleTermination, GuiSlotEditSeed, GuiSlotRequest};
use oblikovati_model::{
    feature::{
        FeatureId,
        hole::{HoleExtent, HolePosition},
        part::PartFeatureDefinition,
        slot::{SlotEndMill, SlotSeat},
        work_geometry::DatumLength,
    },
    history::DocumentSnapshot,
};
use std::collections::BTreeMap;

pub(super) fn edit_seeds(snapshot: &DocumentSnapshot) -> BTreeMap<FeatureId, GuiSlotEditSeed> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter_map(|feature| {
            let PartFeatureDefinition::Slot(definition) = feature.definition() else {
                return None;
            };
            let recipe = definition.recipe();
            let length = length_text(snapshot, recipe.length);
            let width = length_text(snapshot, recipe.width);
            let rotation = format_angle_deg(recipe.rotation_deg);
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
            let (seat, seat_width, seat_depth, seat_angle) = match &recipe.seat {
                SlotSeat::None => (
                    GuiHoleSeatKind::None,
                    width.clone(),
                    "2 mm".into(),
                    "90 deg".into(),
                ),
                SlotSeat::Counterbore { width, depth } => (
                    GuiHoleSeatKind::Counterbore,
                    length_text(snapshot, *width),
                    length_text(snapshot, *depth),
                    "90 deg".into(),
                ),
                SlotSeat::Spotface { width, depth } => (
                    GuiHoleSeatKind::Spotface,
                    length_text(snapshot, *width),
                    length_text(snapshot, *depth),
                    "90 deg".into(),
                ),
                SlotSeat::Countersink { width, angle_deg } => (
                    GuiHoleSeatKind::Countersink,
                    length_text(snapshot, *width),
                    "2 mm".into(),
                    format_angle_deg(*angle_deg),
                ),
            };
            let (end_mill_pointed, end_mill_angle) = match recipe.end_mill {
                SlotEndMill::Flat => (false, "90 deg".into()),
                SlotEndMill::Point { angle_deg } => (true, format_angle_deg(angle_deg)),
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
                GuiSlotEditSeed {
                    body: recipe.body.clone(),
                    face: recipe.face.clone(),
                    position,
                    shape: recipe.shape,
                    kind: recipe.kind,
                    seat,
                    length,
                    width,
                    rotation,
                    depth,
                    seat_width,
                    seat_depth,
                    seat_angle,
                    termination,
                    termination_surface,
                    end_mill_pointed,
                    end_mill_angle,
                    center_marks: recipe.center_marks,
                    direction_flipped: recipe.direction_flipped,
                    extend_start: recipe.extend_start,
                    imate: recipe.imate,
                    fastener: recipe.fastener.clone(),
                },
            ))
        })
        .collect()
}

pub(super) fn request_to_inputs(
    request: &GuiSlotRequest,
    body: Vec<u8>,
    face: Vec<u8>,
) -> Result<SlotFeatureInputs, String> {
    let termination = match request.termination {
        GuiHoleTermination::ThroughAll => HoleTerminationInput::ThroughAll,
        GuiHoleTermination::Distance => HoleTerminationInput::Distance(request.depth.clone()),
        GuiHoleTermination::ToSurface => {
            let surface = request
                .termination_surface
                .clone()
                .ok_or("slot: expected a terminating surface")?;
            HoleTerminationInput::ToSurface(surface)
        }
    };
    let seat = match request.seat {
        GuiHoleSeatKind::None => SlotSeatInput::None,
        GuiHoleSeatKind::Counterbore => SlotSeatInput::Counterbore {
            width: request.seat_width.clone(),
            depth: request.seat_depth.clone(),
        },
        GuiHoleSeatKind::Spotface => SlotSeatInput::Spotface {
            width: request.seat_width.clone(),
            depth: request.seat_depth.clone(),
        },
        GuiHoleSeatKind::Countersink => SlotSeatInput::Countersink {
            width: request.seat_width.clone(),
            angle_deg: parse_open_angle(&request.seat_angle)?,
        },
    };
    let end_mill = if request.end_mill_pointed {
        SlotEndMill::Point {
            angle_deg: parse_open_angle(&request.end_mill_angle)?,
        }
    } else {
        SlotEndMill::Flat
    };
    Ok(SlotFeatureInputs {
        body,
        face,
        position: request.position,
        shape: request.shape,
        length: request.length.clone(),
        width: request.width.clone(),
        rotation_deg: parse_rotation(&request.rotation)?,
        termination,
        kind: request.kind,
        seat,
        end_mill,
        center_marks: request.center_marks,
        direction_flipped: request.direction_flipped,
        extend_start: request.extend_start,
        imate: request.imate,
        fastener: request.fastener.clone(),
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
    if rounded.fract().abs() < 1e-6 {
        format!("{} mm", rounded as i64)
    } else if (rounded * 10.0).fract().abs() < 1e-6 {
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

fn parse_rotation(raw: &str) -> Result<f64, String> {
    let cleaned = raw
        .trim()
        .replace(',', ".")
        .replace('\u{00b0}', "")
        .replace("deg", "");
    let value: f64 = cleaned
        .trim()
        .parse()
        .map_err(|_| format!("slot rotation {raw:?}: expected degrees"))?;
    if !value.is_finite() {
        return Err(format!("slot rotation {value}: expected finite degrees"));
    }
    Ok(value)
}

fn parse_open_angle(raw: &str) -> Result<f64, String> {
    let value = parse_rotation(raw)?;
    if value <= 0.0 || value >= 180.0 {
        return Err(format!("slot angle {value}: expected angle in (0, 180)"));
    }
    Ok(value)
}
