// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn target<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
) -> Result<Vec<u8>, Error> {
    let owner =
        replay.ok_or_else(|| Error("extrude Cut: expected an active part replay owner".into()))?;
    owner.ensure_part(files, document).map_err(Error)?;
    let session = files.workspace().session(document).map_err(api_error)?;
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let current = owner.current_part(files, document).map_err(Error)?;
    let mut bodies = current.ordered_bodies();
    match (bodies.next(), bodies.next()) {
        (Some(body), None) => body
            .body_reference_key(session.history(), guid)
            .map(|key| key.to_vec())
            .map_err(Error),
        _ => Err(Error(format!(
            "extrude Cut target: expected one current body, found {}",
            current.ordered_bodies().count()
        ))),
    }
}
