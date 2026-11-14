// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::{Sketch2D, SketchConstraint};
pub(super) mod shape;
use super::indices::geometric_wire_index;
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
    encode(&ListConstraintsResult {
        constraints: Some(constraint_rows(geometry)?),
    })
}
fn constraint_rows(geometry: &Sketch2D) -> Result<Vec<ConstraintInfo>, Error> {
    geometry
        .geometric_constraints()
        .enumerate()
        .map(|(index, constraint)| constraint_row(index, constraint))
        .collect()
}
fn constraint_row(index: usize, constraint: &SketchConstraint) -> Result<ConstraintInfo, Error> {
    let (kind, entities) = shape::require(&constraint.relation)?;
    Ok(ConstraintInfo {
        index: geometric_wire_index(index)?,
        kind: kind.0.to_string(),
        entities: Some(entities.into_iter().map(|id| id.0).collect()),
        // Structural projection/arc sources are outside this authored enumeration (#3780).
        deletable: true,
        ..Default::default()
    })
}
