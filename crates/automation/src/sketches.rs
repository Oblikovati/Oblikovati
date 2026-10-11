// SPDX-License-Identifier: GPL-2.0-only
use crate::encode;
mod add_constraint;
mod add_entity;
mod constraints;
mod coordinates;
pub(crate) mod create;
mod delete;
mod delete_constraint;
mod delete_dimension;
mod dimensions;
mod drive_dimension;
mod entities;
mod indices;
mod move_dimension;
mod owner;
mod references;
mod transform;
use oblikovati_api::{Error, wire::*};
use oblikovati_model::{
    document_sketch::{DocumentSketchId, DocumentSketchRecompute},
    history::DocumentSnapshot,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{
    SketchSolveProjectionInput, TransactionObserver, files::FileWorkspace,
    workspace::SessionCommands, workspace::WorkspaceObserver,
};
use oblikovati_solve::{
    diagnostics::{ConstraintStatus, ConstraintStructure},
    nonlinear::{NonlinearResult, SolveTermination},
};

/// Geometry commands require a part or assembly; inference preferences use the workspace owner.
pub const SKETCH_COMMAND_METHODS: [&str; 11] = [
    METHOD_SKETCH_CREATE,
    METHOD_SKETCH_SET_INFERENCE_OPTIONS,
    METHOD_SKETCH_SOLVE,
    METHOD_SKETCH_DELETE,
    METHOD_SKETCH_TRANSFORM,
    METHOD_SKETCH_ADD_ENTITY,
    METHOD_SKETCH_ADD_CONSTRAINT,
    METHOD_SKETCH_DELETE_CONSTRAINT,
    METHOD_SKETCH_DELETE_DIMENSION,
    METHOD_SKETCH_MOVE_DIMENSION,
    METHOD_SKETCH_DRIVE_DIMENSION,
];
/// Read-only sketch queries use one current owner and the canonical API DTOs.
pub const SKETCH_QUERY_METHODS: [&str; 6] = [
    METHOD_SKETCH_CONSTRAINTS,
    METHOD_SKETCH_DIMENSIONS,
    METHOD_SKETCH_CONSTRAINT_STATUS,
    METHOD_SKETCH_ENTITIES,
    METHOD_SKETCH_REFERENCE_KEY,
    METHOD_SKETCH_RESOLVE_REFERENCE,
];
pub(super) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    if method == METHOD_SKETCH_DIMENSIONS {
        return dimensions::query(files, crate::decode(method, request)?);
    }
    if method == METHOD_SKETCH_CONSTRAINTS {
        return constraints::query(files, crate::decode(method, request)?);
    }
    if method == METHOD_SKETCH_CONSTRAINT_STATUS {
        return constraint_status(files, crate::decode(method, request)?);
    }
    if method == METHOD_SKETCH_ENTITIES {
        return entities::query(files, crate::decode(method, request)?);
    }
    references::query(files, method, request)
}

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    if method == METHOD_SKETCH_SET_INFERENCE_OPTIONS {
        return files
            .set_sketch_inference_options_projected(
                &crate::decode(method, request)?,
                crate::sketch_inference::inference_reply,
            )
            .map_err(api_error);
    }
    apply_geometry_command(files, method, request)
}
fn apply_geometry_command<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    match method {
        METHOD_SKETCH_TRANSFORM => transform::apply(files, crate::decode(method, request)?),
        METHOD_SKETCH_ADD_ENTITY => add_entity::apply(files, crate::decode(method, request)?),
        METHOD_SKETCH_ADD_CONSTRAINT => {
            add_constraint::apply(files, crate::decode(method, request)?)
        }
        METHOD_SKETCH_DELETE_CONSTRAINT => {
            delete_constraint::apply(files, crate::decode(method, request)?)
        }
        _ => apply_dimension_command(files, method, request),
    }
}
fn apply_dimension_command<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    match method {
        METHOD_SKETCH_DRIVE_DIMENSION => {
            drive_dimension::apply(files, crate::decode(method, request)?)
        }
        METHOD_SKETCH_MOVE_DIMENSION => {
            move_dimension::apply(files, crate::decode(method, request)?)
        }
        METHOD_SKETCH_DELETE_DIMENSION => {
            delete_dimension::apply(files, crate::decode(method, request)?)
        }
        _ => apply_sketch_owner_command(files, method, request),
    }
}
fn apply_sketch_owner_command<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let request: SketchArgs = crate::decode(method, request)?;
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let mut commands = files.commands(document).map_err(api_error)?;
    dispatch_sketch(&mut commands, method, sketch, request.sketch_index)
}
fn dispatch_sketch<O: TransactionObserver>(
    commands: &mut SessionCommands<'_, O>,
    method: &str,
    sketch: DocumentSketchId,
    index: i64,
) -> Result<Vec<u8>, Error> {
    match method {
        METHOD_SKETCH_SOLVE => solve(commands, sketch, index),
        METHOD_SKETCH_DELETE => delete::apply(commands, sketch),
        _ => Err(Error(format!(
            "sketch command {method:?}: expected one of {SKETCH_COMMAND_METHODS:?}"
        ))),
    }
}
fn solve<O: TransactionObserver>(
    commands: &mut SessionCommands<'_, O>,
    sketch: DocumentSketchId,
    index: i64,
) -> Result<Vec<u8>, Error> {
    commands
        .solve_sketch_projected(sketch, || false, |report| solve_reply(report, index))
        .map_err(api_error)?
}
pub(super) fn constraint_status<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    request: SketchArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let structure = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .analyze_sketch(sketch)
        .map_err(api_error)?;
    encode(&constraint_reply(&structure)?)
}
fn active_sketch<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    index: i64,
) -> Result<(u64, DocumentSketchId), Error> {
    let document = owner::active_sketch_host(files)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    Ok((document, sketch_at(snapshot, index)?))
}
fn sketch_at(snapshot: &DocumentSnapshot, index: i64) -> Result<DocumentSketchId, Error> {
    let sketches = snapshot.sketches.sketches();
    usize::try_from(index)
        .ok()
        .and_then(|position| sketches.get(position))
        .map(|sketch| sketch.id())
        .ok_or_else(|| {
            Error(format!(
                "sketch index {index}: expected an ordered index in 0..{}",
                sketches.len()
            ))
        })
}
fn solve_reply(
    report: SketchSolveProjectionInput<'_>,
    index: i64,
) -> Result<Result<Vec<u8>, Error>, Error> {
    let report = report.map_err(api_error)?;
    if report.cancelled {
        return Err(Error(format!(
            "sketch {index} solve cancelled: expected a completed solve before response adoption"
        )));
    }
    let selected = selected_report(report, index)?;
    let outcome = match &selected.result {
        Ok(outcome) => outcome,
        // A local failure still records health; failure to encode a success discards the candidate (#3733).
        Err(cause) => return Ok(Err(api_error(cause))),
    };
    encode(&solve_result(outcome.numerical(), index)?).map(Ok)
}
fn selected_report(
    report: &DocumentSketchRecompute,
    index: i64,
) -> Result<&oblikovati_model::document_sketch::DocumentSketchSolveReport, Error> {
    report.reports.first().ok_or_else(|| {
        Error(format!(
            "sketch {index} solve report {report:?}: expected one completed selected solve"
        ))
    })
}
fn solve_result(numerical: &NonlinearResult, index: i64) -> Result<SolveSketchResult, Error> {
    let converged = numerical.termination == SolveTermination::Converged;
    Ok(SolveSketchResult {
        sketch_index: index,
        dof: wire_count(numerical.structure.degrees_of_freedom)?,
        status: wire_status(numerical.structure.status).into(),
        converged,
        healthy: converged && numerical.structure.status != ConstraintStatus::OverConstrained,
    })
}
fn constraint_reply(structure: &ConstraintStructure) -> Result<ConstraintStatusResult, Error> {
    Ok(ConstraintStatusResult {
        status: wire_status(structure.status).into(),
        dof: wire_count(structure.degrees_of_freedom)?,
        variables: wire_count(structure.variables)?,
        equations: wire_count(structure.equations)?,
        redundant: wire_count(structure.redundant_equations)?,
    })
}
fn wire_status(status: ConstraintStatus) -> &'static str {
    match status {
        ConstraintStatus::WellConstrained => "well",
        ConstraintStatus::UnderConstrained => "under",
        ConstraintStatus::OverConstrained => "over",
    }
}
fn wire_count(count: usize) -> Result<i64, Error> {
    i64::try_from(count).map_err(|_| {
        Error(format!(
            "sketch structural count {count}: expected a nonnegative i64"
        ))
    })
}
fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}

#[cfg(test)]
mod tests;
