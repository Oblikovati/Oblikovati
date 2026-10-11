// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_session::SketchDeleteProjectionInput;

pub(super) fn apply<O: TransactionObserver>(
    commands: &mut SessionCommands<'_, O>,
    sketch: DocumentSketchId,
) -> Result<Vec<u8>, Error> {
    commands
        .delete_sketch_projected(sketch, deletion_reply)
        .map_err(api_error)
}
fn deletion_reply(report: SketchDeleteProjectionInput<'_>) -> Result<Vec<u8>, Error> {
    report.map_err(api_error)?;
    encode(&OKResult { ok: true })
}
