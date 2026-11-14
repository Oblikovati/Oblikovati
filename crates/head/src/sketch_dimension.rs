// SPDX-License-Identifier: GPL-2.0-only
use crate::sketch_operands as operands;
use crate::{
    HeadSketchEdit,
    sketch_operands::OperandCategory::{Circular, Elliptical, Line, Point},
};
use oblikovati_api::{
    types::{
        DIM_CONSTRAINT_ANGLE, DIM_CONSTRAINT_DISTANCE, DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS,
        DIM_CONSTRAINT_ELLIPSE_RADIUS, DIM_CONSTRAINT_RADIUS, DimensionConstraintKind,
    },
    wire::AddDimensionArgs,
};
use oblikovati_model::sketch::{Sketch2D, SketchEntityId};
use oblikovati_model::{
    sketch::{
        DistanceMeasure, MeasuredDimensionOperands, SeededDimensionOperands,
        SketchDimensionalConstraintCreation,
    },
    units::UnitsOfMeasure,
};
use oblikovati_session::selection::SketchPick;

pub(crate) struct SketchDimensionState {
    pub edit: HeadSketchEdit,
    pub generation: u64,
    pub serial: u64,
    pub placed: Option<[f64; 2]>,
    pub cursor: Option<[f64; 2]>,
    pub proposal: Option<AddDimensionArgs>,
    pub committed: Option<oblikovati_gui::GuiDimensionEdit>,
    pub refusal: Option<String>,
}
impl SketchDimensionState {
    pub(crate) fn wants(
        &self,
        geometry: &Sketch2D,
        picks: &[SketchPick],
        entity: SketchEntityId,
    ) -> Result<bool, String> {
        let selected = operands::resolve(geometry, picks)?;
        if selected.iter().any(|operand| operand.id == entity) {
            return Ok(false);
        }
        let kind = geometry
            .entity_kind(entity)
            .map_err(|cause| cause.to_string())?;
        let Some(category) = operands::category(&kind) else {
            return Ok(false);
        };
        if matches!(category, Circular | Elliptical) && !selected.is_empty() {
            return Ok(false);
        }
        if !dimension_pick_set_ready(&selected) {
            return Ok(true);
        }
        // Go #2022: a first line stays open until placement, so a second can select an angle.
        Ok(selected.len() == 1 && selected[0].category == Line && category == Line)
    }
    pub(crate) fn ready(&self, geometry: &Sketch2D, picks: &[SketchPick]) -> Result<bool, String> {
        Ok(dimension_pick_set_ready(&operands::resolve(
            geometry, picks,
        )?))
    }
    pub(crate) fn request_at(
        &self,
        geometry: &Sketch2D,
        picks: &[SketchPick],
        index: i64,
        units: &UnitsOfMeasure,
        position: [f64; 2],
    ) -> Result<AddDimensionArgs, String> {
        let selected = operands::resolve(geometry, picks)?;
        let (kind, entities, orientation) =
            dimension_selection_recipe(geometry, &selected, position)?;
        let input = dimension_measurement_input(kind.0.as_ref(), &entities, &orientation)?;
        let measured = geometry
            .measure_dimension_input(&input)
            .map_err(|cause| cause.to_string())?;
        let expression = units
            .measured_expression(measured)
            .map_err(|cause| cause.to_string())?;
        Ok(AddDimensionArgs {
            sketch_index: index,
            kind: kind.0.to_string(),
            entities: Some(entities),
            orientation,
            expression,
            text_point: Some(
                position
                    .map(|coordinate| coordinate * units.working_scale())
                    .to_vec(),
            ),
            ..Default::default()
        })
    }
}
fn dimension_pick_set_ready(selected: &[operands::SketchOperand]) -> bool {
    selected
        .iter()
        .any(|operand| matches!(operand.category, Line | Circular | Elliptical))
        || operands::ids(selected, Point).len() >= 2
}
fn dimension_selection_recipe(
    geometry: &Sketch2D,
    selected: &[operands::SketchOperand],
    at: [f64; 2],
) -> Result<(DimensionConstraintKind, Vec<u64>, String), String> {
    if let [operand] = selected
        && matches!(operand.category, Circular | Elliptical)
    {
        let kind = curve_dimension_kind(geometry, *operand, at)?;
        return Ok((kind, vec![operand.id.0], String::new()));
    }
    let lines = operands::ids(selected, Line);
    if let [first, second, ..] = lines.as_slice() {
        return Ok((DIM_CONSTRAINT_ANGLE, vec![first.0, second.0], String::new()));
    }
    let points = dimension_point_pair(geometry, selected, &lines)?;
    let a = geometry
        .point(points[0])
        .map_err(|cause| cause.to_string())?
        .position;
    let b = geometry
        .point(points[1])
        .map_err(|cause| cause.to_string())?
        .position;
    Ok((
        DIM_CONSTRAINT_DISTANCE,
        points.map(|id| id.0).to_vec(),
        dimension_placement_orientation(a, b, at).into(),
    ))
}
fn curve_dimension_kind(
    geometry: &Sketch2D,
    operand: operands::SketchOperand,
    at: [f64; 2],
) -> Result<DimensionConstraintKind, String> {
    match operand.category {
        Circular => Ok(DIM_CONSTRAINT_RADIUS),
        Elliptical => ellipse_dimension_kind(geometry, operand.id, at),
        _ => Err(format!(
            "dimension curve {operand:?}: expected a circular or elliptical curve"
        )),
    }
}
fn ellipse_dimension_kind(
    geometry: &Sketch2D,
    id: SketchEntityId,
    at: [f64; 2],
) -> Result<DimensionConstraintKind, String> {
    let ellipse = geometry.ellipse(id).map_err(|cause| cause.to_string())?;
    let center = geometry
        .point(ellipse.center())
        .map_err(|cause| cause.to_string())?
        .position;
    let [u, v] = ellipse.major_axis();
    let [x, y] = [at[0] - center[0], at[1] - center[1]];
    // Extend Go #2025's perpendicular label-drag choice in the ellipse's own frame.
    // A tie keeps the major axis, matching the existing aligned-first tie policy.
    Ok(if (u * x + v * y).abs() <= (-v * x + u * y).abs() {
        DIM_CONSTRAINT_ELLIPSE_RADIUS
    } else {
        DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS
    })
}
fn dimension_point_pair(
    geometry: &Sketch2D,
    selected: &[operands::SketchOperand],
    lines: &[SketchEntityId],
) -> Result<[SketchEntityId; 2], String> {
    let points = operands::ids(selected, Point);
    if let [first, second, ..] = points.as_slice() {
        return Ok([*first, *second]);
    }
    if let Some(line) = lines.first() {
        return geometry
            .line(*line)
            .map(|line| line.endpoints)
            .map_err(|cause| cause.to_string());
    }
    Err(format!(
        "dimension operands {selected:?}: expected two points, one line, a circular curve or two lines"
    ))
}
fn dimension_placement_orientation(a: [f64; 2], b: [f64; 2], at: [f64; 2]) -> &'static str {
    let delta = [b[0] - a[0], b[1] - a[1]];
    let length = delta[0].hypot(delta[1]);
    if length == 0. {
        return "aligned";
    }
    let drag = [at[0] - a[0].midpoint(b[0]), at[1] - a[1].midpoint(b[1])];
    let mut best = (
        "aligned",
        (delta[0] / length * drag[0] + delta[1] / length * drag[1]).abs(),
    );
    // Go #2025: ties keep aligned; each candidate compares its direction against the label drag.
    for candidate in [("horizontal", drag[0].abs()), ("vertical", drag[1].abs())] {
        if candidate.1 < best.1 {
            best = candidate;
        }
    }
    best.0
}

pub(crate) fn dimension_measurement_input(
    kind: &str,
    entities: &[u64],
    orientation: &str,
) -> Result<SketchDimensionalConstraintCreation, String> {
    let input = match entities {
        [curve] => curve_dimension_input(kind, SketchEntityId(*curve)),
        [first, second] => pair_dimension_input(kind, [*first, *second], orientation),
        _ => None,
    };
    input.ok_or_else(|| format!(
        "dimension {kind:?} operands {entities:?}: expected a distance, circular or elliptical radius, or angle recipe"
    ))
}
fn curve_dimension_input(
    kind: &str,
    curve: SketchEntityId,
) -> Option<SketchDimensionalConstraintCreation> {
    let measured = match kind {
        kind if kind == DIM_CONSTRAINT_RADIUS.0.as_ref() => {
            MeasuredDimensionOperands::Radius { curve }
        }
        kind if kind == DIM_CONSTRAINT_ELLIPSE_RADIUS.0.as_ref() => {
            MeasuredDimensionOperands::EllipseRadius { ellipse: curve }
        }
        kind if kind == DIM_CONSTRAINT_ELLIPSE_MINOR_RADIUS.0.as_ref() => {
            MeasuredDimensionOperands::EllipseMinorRadius { ellipse: curve }
        }
        _ => return None,
    };
    Some(SketchDimensionalConstraintCreation::Measured(measured))
}
fn pair_dimension_input(
    kind: &str,
    entities: [u64; 2],
    orientation: &str,
) -> Option<SketchDimensionalConstraintCreation> {
    use SketchDimensionalConstraintCreation::{Measured, Seeded};
    let points = entities.map(SketchEntityId);
    Some(match kind {
        kind if kind == DIM_CONSTRAINT_ANGLE.0.as_ref() => {
            Seeded(SeededDimensionOperands::Angle(points))
        }
        kind if kind == DIM_CONSTRAINT_DISTANCE.0.as_ref() => {
            Measured(MeasuredDimensionOperands::Distance {
                points,
                measure: dimension_distance_measure(orientation),
            })
        }
        _ => return None,
    })
}
fn dimension_distance_measure(orientation: &str) -> DistanceMeasure {
    match orientation {
        "horizontal" => DistanceMeasure::Horizontal,
        "vertical" => DistanceMeasure::Vertical,
        _ => DistanceMeasure::Aligned,
    }
}
