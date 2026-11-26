// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn operation(input: &Extrude) -> Option<PrismBoolean> {
    match input.operation.trim().to_ascii_lowercase().as_str() {
        "cut" => Some(PrismBoolean::Cut),
        "join" => Some(PrismBoolean::Join),
        _ => None,
    }
}

pub(super) fn target<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    operation: PrismBoolean,
) -> Result<Vec<u8>, Error> {
    let owner = replay.ok_or_else(|| {
        Error(format!(
            "extrude {operation}: expected an active part replay owner"
        ))
    })?;
    owner.ensure_part(files, document).map_err(Error)?;
    let session = files.workspace().session(document).map_err(api_error)?;
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let current = owner.current_part(files, document).map_err(Error)?;
    let solids = || current.ordered_bodies().filter(|body| !body.is_surface());
    let mut bodies = solids();
    match (bodies.next(), bodies.next()) {
        (Some(body), None) => body
            .body_reference_key(session.history(), guid)
            .map(|key| key.to_vec())
            .map_err(Error),
        _ => Err(Error(format!(
            "extrude {operation} target: expected one current body, found {}",
            solids().count()
        ))),
    }
}
