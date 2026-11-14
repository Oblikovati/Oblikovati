// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchSlotPlacement;
#[derive(Clone, Copy)]
pub(crate) enum SlotPlacementInput {
    Authored(SketchSlotPlacement),
    ThreePoints([[f64; 2]; 3]),
}
#[derive(Clone, Copy)]
pub(crate) struct SlotInput {
    pub placement: SlotPlacementInput,
    pub width: f64,
    pub construction: bool,
}
pub(super) fn prepare(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
) -> Result<PreparedEntity, Error> {
    let points = points::resolve(snapshot, request)?;
    let width = length_expression(snapshot, &request.width)?;
    let placement = placement(snapshot, request, &points)?;
    Ok(PreparedEntity::Slot(SlotInput {
        placement,
        width,
        construction: request.construction,
    }))
}
fn placement(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
    points: &[[f64; 2]],
) -> Result<SlotPlacementInput, Error> {
    let placement = match request.variant.as_str() {
        "arcThreePoint" => return Ok(SlotPlacementInput::ThreePoints(require_points(points)?)),
        "arc" => {
            let [center, start, end] = require_points(points)?;
            SketchSlotPlacement::Arc {
                center,
                endpoints: [start, end],
                counter_clockwise: request.ccw,
            }
        }
        "arcCenterPoint" => center_point(snapshot, request, require_points(points)?)?,
        "overall" => SketchSlotPlacement::Overall(require_points(points)?),
        "slotCenter" => SketchSlotPlacement::SlotCenter(require_points(points)?),
        _ => SketchSlotPlacement::Centers(require_points(points)?),
    };
    Ok(SlotPlacementInput::Authored(placement))
}
fn require_points<const N: usize>(points: &[[f64; 2]]) -> Result<[[f64; 2]; N], Error> {
    points.try_into().map_err(|_| {
        Error(format!(
            "slot positions {points:?}: expected exactly {N} positions"
        ))
    })
}
fn center_point(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
    [center, start]: [[f64; 2]; 2],
) -> Result<SketchSlotPlacement, Error> {
    let sweep = snapshot
        .parameters
        .evaluate_scalar_expression(&request.end_angle, Dimension::Angle)
        .map_err(api_error)?
        .value;
    Ok(SketchSlotPlacement::ArcCenterPoint {
        center,
        start,
        sweep,
    })
}
