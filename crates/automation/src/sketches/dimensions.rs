// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    parameter::ParameterState,
    quantity::{Dimension, Quantity},
    sketch::{Sketch2D, SketchConstraint},
};
mod shape;
pub(super) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    request: SketchArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let geometry = snapshot.sketches.get(sketch).map_err(api_error)?.geometry();
    let rows = geometry
        .dimensional_constraints()
        .enumerate()
        .map(|(index, (constraint, parameter))| {
            dimension_row(snapshot, geometry, index, constraint, parameter)
        })
        .collect::<Result<Vec<_>, _>>()?;
    encode(&ListDimensionsResult {
        dimensions: Some(rows),
    })
}
fn dimension_row(
    snapshot: &DocumentSnapshot,
    geometry: &Sketch2D,
    index: usize,
    constraint: &SketchConstraint,
    parameter: u64,
) -> Result<DimensionInfo, Error> {
    let measured = geometry
        .measure_dimension(constraint.id)
        .map_err(api_error)?;
    let state = require_parameter(snapshot, parameter, measured.dimension)?;
    let (kind, orientation, linear_diameter) = shape::require(&constraint.relation)?;
    Ok(DimensionInfo {
        index: super::indices::constraint_wire_index(index, "dimensional constraint")?,
        kind: kind.0.to_string(),
        name: state.name.clone(),
        expression: state.source.clone(),
        value: canonical_measurement(measured, snapshot.parameters.units().working_scale())?,
        orientation: orientation.into(),
        linear_diameter,
        // Current authored dimensions constrain geometry; reference modes and text placement are separate port work (#3782).
        ..Default::default()
    })
}
fn require_parameter(
    snapshot: &DocumentSnapshot,
    parameter: u64,
    expected: Dimension,
) -> Result<&ParameterState<u64>, Error> {
    let state = snapshot.parameters.get(parameter).ok_or_else(|| {
        Error(format!(
            "dimension parameter {parameter}: expected an owned healthy {expected:?} source"
        ))
    })?;
    let value = healthy_source_value(snapshot, state, parameter, expected)?;
    if value.dimension != expected || !value.value.is_finite() {
        return Err(Error(format!(
            "dimension parameter {parameter} {:?} value {value:?}: expected a finite healthy {expected:?} source",
            state.name
        )));
    }
    Ok(state)
}
fn healthy_source_value(
    snapshot: &DocumentSnapshot,
    state: &ParameterState<u64>,
    parameter: u64,
    expected: Dimension,
) -> Result<Quantity, Error> {
    snapshot.parameters.model_value(parameter).map_err(|cause| {
        Error(format!(
            "dimension parameter {parameter} {:?}: expected a finite healthy {expected:?} source: {cause}",
            state.name
        ))
    })
}
fn canonical_measurement(measured: Quantity, scale: f64) -> Result<f64, Error> {
    let value = match measured.dimension {
        Dimension::Length => measured.value * scale,
        Dimension::Angle => measured.value,
        _ => {
            return Err(Error(format!(
                "dimension measurement {measured:?}: expected a length or angular quantity"
            )));
        }
    };
    if !value.is_finite() {
        return Err(Error(format!(
            "dimension measurement {measured:?} scale {scale:?}: expected a finite centimetre or radian value"
        )));
    }
    Ok(value)
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn overflow_cannot_be_reported_as_a_finite_length_measurement() {
        let measured = Quantity::new(f64::MAX, Dimension::Length);
        let cause = canonical_measurement(measured, 2.0).unwrap_err().0;
        assert!(cause.contains("scale 2") && cause.contains("finite centimetre or radian"));
    }
}

#[cfg(test)]
mod category_tests {
    use super::*;
    #[test]
    fn other_quantity_categories_cannot_be_encoded_as_length_or_angle_measurements() {
        let measured = Quantity::new(2.0, Dimension::Area);
        let cause = canonical_measurement(measured, 1.0).unwrap_err().0;
        assert!(cause.contains("Area") && cause.contains("length or angular"));
    }
}
