// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    parameter::ParameterState,
    quantity::{Dimension, Quantity},
    sketch::{Sketch2D, SketchConstraint, SketchConstraintId},
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
    let id = constraint.id;
    let measured = geometry.measure_dimension(id).map_err(api_error)?;
    let reference = geometry.dimension_is_reference(id).map_err(api_error)?;
    let state = dimension_parameter_state(snapshot, parameter, measured.dimension, reference)?;
    let scale = snapshot.parameters.units().working_scale();
    let (kind, orientation, linear_diameter) = shape::require(&constraint.relation)?;
    Ok(DimensionInfo {
        index: super::indices::constraint_wire_index(index, "dimensional constraint")?,
        kind: kind.0.to_string(),
        name: state.name.clone(),
        expression: state.source.clone(),
        value: canonical_measurement(measured, scale)?,
        orientation: orientation.into(),
        linear_diameter,
        text_point: canonical_text_point(geometry, constraint.id, scale)?,
        driven: reference,
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

pub(super) fn selected_constraint(
    geometry: &Sketch2D,
    index: i64,
) -> Result<SketchConstraintId, Error> {
    usize::try_from(index)
        .ok()
        .and_then(|position| geometry.dimensional_constraints().nth(position))
        .map(|(constraint, _)| constraint.id)
        .ok_or_else(|| {
            Error(format!(
                "dimension index {index}: expected an owned row in 0..{}",
                geometry.dimensional_constraints().count()
            ))
        })
}

fn canonical_text_point(
    geometry: &Sketch2D,
    dimension: SketchConstraintId,
    scale: f64,
) -> Result<Option<Vec<f64>>, Error> {
    geometry
        .dimension_text_point(dimension)
        .map_err(api_error)?
        .map(|position| {
            position
                .into_iter()
                .map(|value| canonical_measurement(Quantity::new(value, Dimension::Length), scale))
                .collect()
        })
        .transpose()
}

#[cfg(test)]
mod placement_tests {
    use super::*;
    #[test]
    fn annotation_conversion_cannot_report_infinite_centimetre_coordinates() {
        let mut geometry = positioned_dimension();
        geometry
            .set_dimension_text_point(SketchConstraintId(1), [f64::MAX, 0.0])
            .unwrap();
        let cause = canonical_text_point(&geometry, SketchConstraintId(1), 2.0)
            .unwrap_err()
            .0;
        assert!(cause.contains("scale 2") && cause.contains("finite centimetre"));
    }
    fn positioned_dimension() -> Sketch2D {
        use oblikovati_model::sketch::{PlanarConstraint, SketchEntityId};
        let mut geometry = Sketch2D::default();
        geometry.add_point(SketchEntityId(1), [0.0, 0.0]).unwrap();
        geometry
            .add_circle(SketchEntityId(2), SketchEntityId(1), 1.0)
            .unwrap();
        let relation = PlanarConstraint::Radius {
            curve: SketchEntityId(2),
            parameter: 1,
        };
        geometry
            .add_constraint(SketchConstraintId(1), relation)
            .unwrap();
        geometry
    }
}

fn dimension_parameter_state(
    snapshot: &DocumentSnapshot,
    parameter: u64,
    category: Dimension,
    reference: bool,
) -> Result<&ParameterState<u64>, Error> {
    if !reference {
        return require_parameter(snapshot, parameter, category);
    }
    snapshot.parameters.get(parameter).ok_or_else(|| Error(format!(
        "reference dimension parameter {parameter}: expected an owned source for name and expression inspection"
    )))
}
