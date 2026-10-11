// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_session::SketchDimensionCreationProjectionInput;
mod inputs;
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: AddDimensionArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let geometry = snapshot.sketches.get(sketch).map_err(api_error)?.geometry();
    let scale = snapshot.parameters.units().working_scale();
    let input = inputs::prepare(geometry, scale, &request)?;
    files
        .commands(document)
        .map_err(api_error)?
        .create_sketch_dimension_projected(sketch, &input, |result| {
            encode_created(result, &request.kind, scale)
        })
        .map_err(api_error)
}
fn encode_created(
    result: SketchDimensionCreationProjectionInput,
    kind: &str,
    scale: f64,
) -> Result<Vec<u8>, Error> {
    let report = result.map_err(api_error)?;
    encode(&AddDimensionResult {
        index: super::indices::constraint_wire_index(report.dimension_index, "dimension")?,
        kind: kind.into(),
        parameter: report.parameter_name,
        value: super::dimensions::canonical_measurement(report.measured, scale)?,
        dof: super::wire_count(report.degrees_of_freedom)?,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::{
        history::SketchDimensionCreationReport,
        quantity::{Dimension, Quantity},
        sketch::SketchConstraintId,
    };
    fn report() -> SketchDimensionCreationReport {
        SketchDimensionCreationReport {
            constraint: SketchConstraintId(1),
            parameter: 1,
            parameter_name: "d0".into(),
            dimension_index: 0,
            measured: Quantity::new(2.5, Dimension::Length),
            degrees_of_freedom: 2,
            unavailable_targets: vec![],
        }
    }
    #[test]
    fn projected_reply_checks_units_and_integer_boundaries_before_adoption() {
        let encoded = encode_created(Ok(report()), "radius", 10.0).unwrap();
        let result: AddDimensionResult = serde_json::from_slice(&encoded).unwrap();
        assert_eq!(result.value, 25.0);
        for field in ["index", "dof"] {
            let mut input = report();
            if field == "index" {
                input.dimension_index = usize::MAX;
            } else {
                input.degrees_of_freedom = usize::MAX;
            }
            if i64::try_from(usize::MAX).is_err() {
                assert!(encode_created(Ok(input), "radius", 1.0).is_err());
            }
        }
        assert!(encode_created(Ok(report()), "radius", f64::MAX).is_err());
    }
}
