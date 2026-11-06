// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_session::SketchGeometricCreationProjectionInput;
mod inputs;
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: AddConstraintArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let geometry = snapshot.sketches.get(sketch).map_err(api_error)?.geometry();
    let input = inputs::prepare(geometry, &request)?;
    files
        .commands(document)
        .map_err(api_error)?
        .create_sketch_geometric_constraint_projected(sketch, &input, encode_created)
        .map_err(api_error)
}
fn encode_created(result: SketchGeometricCreationProjectionInput) -> Result<Vec<u8>, Error> {
    let report = result.map_err(api_error)?;
    let (kind, _) = super::constraints::shape::require(&report.constraint.relation)?;
    let dof = i64::try_from(report.degrees_of_freedom).map_err(|_| {
        Error(format!(
            "constraint DOF {}: expected a nonnegative signed wire integer",
            report.degrees_of_freedom
        ))
    })?;
    encode(&AddConstraintResult {
        index: super::indices::geometric_wire_index(report.geometric_index)?,
        kind: kind.0.to_string(),
        dof,
    })
}
