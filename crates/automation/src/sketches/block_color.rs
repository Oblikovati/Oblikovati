// SPDX-License-Identifier: GPL-2.0-only
//! Edit block colors through the shared typed API.
use super::*;
use oblikovati_model::sketch::SketchEntityId;

pub(super) fn definition<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: SetBlockDefinitionColorArgs,
) -> Result<Vec<u8>, Error> {
    let document = block_definitions::active_part(files)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Set block definition color",
            |snapshot, _| {
                snapshot
                    .sketches
                    .set_block_definition_color(
                        &request.name,
                        request.color.clone().map(|color| *color),
                    )
                    .map_err(api_error)
            },
            |snapshot, ()| definition_reply(snapshot, &request.name),
        )
        .map_err(api_error)
}
fn definition_reply(snapshot: &DocumentSnapshot, name: &str) -> Result<Vec<u8>, Error> {
    let (index, definition) = snapshot
        .sketches
        .block_definitions()
        .iter()
        .enumerate()
        .find(|(_, item)| item.name() == name)
        .ok_or_else(|| {
            Error(format!(
                "block definition {name:?}: expected an existing definition"
            ))
        })?;
    encode(&block_definitions::info(
        index,
        definition,
        snapshot.sketches.block_instance_count(name),
    )?)
}
pub(super) fn instance<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: SetBlockInstanceColorArgs,
) -> Result<Vec<u8>, Error> {
    let document = block_definitions::active_part(files)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Set block instance color",
            |snapshot, _| {
                let sketch = sketch_at(snapshot, request.sketch_index)?;
                snapshot
                    .sketches
                    .set_block_instance_color(
                        sketch,
                        SketchEntityId(request.instance_id),
                        request.color.clone().map(|color| *color),
                    )
                    .map_err(api_error)
            },
            |snapshot, ()| instance_reply(snapshot, &request),
        )
        .map_err(api_error)
}
fn instance_reply(
    snapshot: &DocumentSnapshot,
    request: &SetBlockInstanceColorArgs,
) -> Result<Vec<u8>, Error> {
    let sketch = sketch_at(snapshot, request.sketch_index)?;
    let placed = snapshot.sketches.get(sketch).map_err(api_error)?;
    let (index, item) = placed
        .block_instances()
        .iter()
        .enumerate()
        .find(|(_, item)| item.id().0 == request.instance_id)
        .ok_or_else(|| {
            Error(format!(
                "block instance {}: expected an existing placement",
                request.instance_id
            ))
        })?;
    encode(&block_definitions::instance_info(
        snapshot,
        index,
        item,
        snapshot.parameters.units().working_scale(),
    )?)
}
