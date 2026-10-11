// SPDX-License-Identifier: GPL-2.0-only
use crate::encode;
mod delete;
use oblikovati_api::{Error, types::DOCUMENT_PART, wire::*};
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

/// Selected sketch commands require a retained part-document kind.
pub const SKETCH_COMMAND_METHODS: [&str; 2] = [METHOD_SKETCH_SOLVE, METHOD_SKETCH_DELETE];

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
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
    let document = active_part(files)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    Ok((document, sketch_at(snapshot, index)?))
}
fn active_part<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<u64, Error> {
    let document = files.workspace().active_document().ok_or_else(|| {
        Error("active document is absent: expected an active part document".into())
    })?;
    let kind = files.document_type(document).map_err(api_error)?;
    if kind != DOCUMENT_PART {
        return Err(Error(format!(
            "document {document} kind {kind:?}: expected {DOCUMENT_PART:?} for a sketch query or command"
        )));
    }
    Ok(document)
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
