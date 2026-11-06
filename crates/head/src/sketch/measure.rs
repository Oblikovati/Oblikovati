// SPDX-License-Identifier: GPL-2.0-only
//! What Measure holds and how its selections read as result rows.
use crate::HeadSketchEdit;
use oblikovati_martensite::planar_curve::sketch::{
    SketchCurveMeasure, SketchDistanceRange, measure_sketch_curve, sketch_curve_distance_range,
    sketch_point_curve_distance_range,
};
use oblikovati_model::{
    quantity::{Dimension, Quantity},
    sketch::{Sketch2D, SketchEntityId, SketchPlane},
    units::UnitsOfMeasure,
};

/// One measured operand; a line's midpoint is a point of its own.
#[derive(Clone, Copy, Debug, PartialEq)]
pub(crate) enum MeasurePick {
    Curve(SketchEntityId),
    Point([f64; 2]),
}
/// The running Measure command on one sketch.
pub(crate) struct SketchMeasureState {
    pub edit: HeadSketchEdit,
    pub generation: u64,
    pub picks: Vec<MeasurePick>,
    pub rows: Vec<String>,
}
impl SketchMeasureState {
    /// A third selection starts over, so the panel always compares at most two operands.
    pub(crate) fn add_pick(&mut self, pick: MeasurePick) {
        if self.picks.contains(&pick) {
            return;
        }
        if self.picks.len() == 2 {
            self.picks.clear();
        }
        self.picks.push(pick);
    }
}
/// Everything that reads from the sketch and its document when rows are computed.
pub(crate) struct MeasureContext<'a> {
    pub sketch: &'a Sketch2D,
    pub plane: SketchPlane,
    pub units: &'a UnitsOfMeasure,
}
/// Rows for the current selections; example: a curve's length, then the pair's distances.
pub(crate) fn measure_rows(
    context: &MeasureContext<'_>,
    picks: &[MeasurePick],
) -> Result<Vec<String>, String> {
    let mut rows = Vec::new();
    for (index, pick) in picks.iter().enumerate() {
        pick_rows(context, index + 1, *pick, &mut rows)?;
    }
    if let [first, second] = picks {
        pair_rows(context, [*first, *second], &mut rows)?;
    }
    Ok(rows)
}
fn pick_rows(
    context: &MeasureContext<'_>,
    number: usize,
    pick: MeasurePick,
    rows: &mut Vec<String>,
) -> Result<(), String> {
    match pick {
        MeasurePick::Point(point) => {
            rows.push(format!("Selection {number}: Point"));
            rows.push(format!("Position {}", model_position(context, point)?));
        }
        MeasurePick::Curve(curve) => {
            let measure =
                measure_sketch_curve(context.sketch, curve).map_err(|cause| cause.to_string())?;
            rows.push(format!("Selection {number}: Curve"));
            curve_rows(context, measure, rows)?;
        }
    }
    Ok(())
}
fn curve_rows(
    context: &MeasureContext<'_>,
    measure: SketchCurveMeasure,
    rows: &mut Vec<String>,
) -> Result<(), String> {
    if let Some(circular) = measure.circular {
        rows.push(format!("Radius {}", length(context, circular.radius)?));
        rows.push(format!(
            "Diameter {}",
            length(context, circular.radius * 2.)?
        ));
    }
    rows.push(format!("Length {}", length(context, measure.length)?));
    if let Some(circular) = measure.circular {
        rows.push(format!(
            "Sweep {:.2} deg",
            circular.sweep.abs().to_degrees()
        ));
        rows.push(format!(
            "Center {}",
            model_position(context, circular.center)?
        ));
    }
    Ok(())
}
fn pair_rows(
    context: &MeasureContext<'_>,
    picks: [MeasurePick; 2],
    rows: &mut Vec<String>,
) -> Result<(), String> {
    if let Some(angle) = line_pair_angle(context.sketch, picks)? {
        rows.push(format!("Angle {:.2} deg", angle.to_degrees()));
    }
    let range = pair_range(context.sketch, picks)?;
    rows.push(format!(
        "Minimum distance {}",
        length(context, range.minimum.distance)?
    ));
    rows.push(format!(
        "Maximum distance {}",
        length(context, range.maximum.distance)?
    ));
    rows.push(format!(
        "Minimum delta {}",
        model_delta(context, range.minimum.points)?
    ));
    Ok(())
}
fn pair_range(sketch: &Sketch2D, picks: [MeasurePick; 2]) -> Result<SketchDistanceRange, String> {
    let range = match picks {
        [MeasurePick::Curve(first), MeasurePick::Curve(second)] => {
            sketch_curve_distance_range(sketch, [first, second])
        }
        [MeasurePick::Point(point), MeasurePick::Curve(curve)]
        | [MeasurePick::Curve(curve), MeasurePick::Point(point)] => {
            sketch_point_curve_distance_range(sketch, point, curve)
        }
        [MeasurePick::Point(first), MeasurePick::Point(second)] => {
            let distance = (first[0] - second[0]).hypot(first[1] - second[1]);
            let witness = oblikovati_martensite::planar_curve::sketch::SketchDistanceWitness {
                distance,
                points: [first, second],
            };
            return Ok(SketchDistanceRange {
                minimum: witness,
                maximum: witness,
            });
        }
    };
    range.map_err(|cause| cause.to_string())
}
/// The acute angle between two lines; anything else has no one angle to report.
fn line_pair_angle(sketch: &Sketch2D, picks: [MeasurePick; 2]) -> Result<Option<f64>, String> {
    let [MeasurePick::Curve(first), MeasurePick::Curve(second)] = picks else {
        return Ok(None);
    };
    let directions = [first, second].map(|curve| line_direction(sketch, curve));
    let [Some(a), Some(b)] = directions else {
        return Ok(None);
    };
    let cross = a[0] * b[1] - a[1] * b[0];
    let dot = a[0] * b[0] + a[1] * b[1];
    Ok(Some(cross.abs().atan2(dot.abs())))
}
fn line_direction(sketch: &Sketch2D, curve: SketchEntityId) -> Option<[f64; 2]> {
    let measure = measure_sketch_curve(sketch, curve).ok()?;
    if measure.circular.is_some() {
        return None;
    }
    let [start, end] = measure.endpoints?;
    Some([end[0] - start[0], end[1] - start[1]])
}
fn length(context: &MeasureContext<'_>, value: f64) -> Result<String, String> {
    context
        .units
        .format_display(Quantity::new(value, Dimension::Length))
        .map_err(|cause| cause.to_string())
}
fn model_position(context: &MeasureContext<'_>, point: [f64; 2]) -> Result<String, String> {
    let model = context
        .plane
        .to_model(point)
        .map_err(|cause| cause.to_string())?;
    axis_triple(context, model)
}
fn model_delta(context: &MeasureContext<'_>, points: [[f64; 2]; 2]) -> Result<String, String> {
    let ends = points.map(|point| context.plane.to_model(point));
    let [first, second] = ends;
    let (first, second) = (
        first.map_err(|cause| cause.to_string())?,
        second.map_err(|cause| cause.to_string())?,
    );
    axis_triple(
        context,
        std::array::from_fn(|axis| (first[axis] - second[axis]).abs()),
    )
}
fn axis_triple(context: &MeasureContext<'_>, values: [f64; 3]) -> Result<String, String> {
    let [x, y, z] = values.map(|value| length(context, value));
    let (x, y, z) = (x?, y?, z?);
    let bare = |text: &str| {
        let number = text.split_whitespace().next().unwrap_or(text);
        // A value that rounds to zero reads as zero, never as "-0.000".
        number
            .strip_prefix('-')
            .filter(|rest| rest.chars().all(|c| c == '0' || c == '.'))
            .unwrap_or(number)
            .to_owned()
    };
    let unit = x.split_whitespace().nth(1).unwrap_or("");
    Ok(format!(
        "X {} Y {} Z {} {unit}",
        bare(&x),
        bare(&y),
        bare(&z)
    ))
}

#[cfg(test)]
#[path = "../../tests/sketch/measure_rows.rs"]
mod tests;
