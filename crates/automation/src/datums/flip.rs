// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    args: FlipWorkPlaneArgs,
) -> Result<Vec<u8>, Error> {
    let (document, snapshot) = datum_snapshot(files)?;
    let id = redefine::placed_plane(snapshot, args.index)?.id();
    files
        .commands(document)
        .map_err(api_error)?
        .run("Flip WorkPlane normal", |snapshot, _| {
            install(snapshot, id, args.index)
        })
        .map_err(api_error)
}
fn install(snapshot: &mut DocumentSnapshot, id: FeatureId, index: i64) -> Result<Vec<u8>, Error> {
    snapshot
        .work_geometry
        .flip_plane_normal(id)
        .map_err(api_error)?;
    let plane = planes::current_info(snapshot, id, index)?;
    encode(&FlipWorkPlaneResult { plane })
}
