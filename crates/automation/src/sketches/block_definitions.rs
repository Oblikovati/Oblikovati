// SPDX-License-Identifier: GPL-2.0-only
//! The canonical block-definition registry on a part document.
use super::*;
use oblikovati_api::types::DOCUMENT_PART;

fn active_part<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<u64, Error> {
    crate::part_owner::require_active_kind(files, &[DOCUMENT_PART], "part document")
}
pub(super) fn list<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    let session = files.workspace().session(document).map_err(api_error)?;
    let definitions = session
        .snapshot()
        .sketches
        .block_definitions()
        .iter()
        .enumerate()
        .map(|(index, definition)| {
            info(
                index,
                definition.name(),
                definition.geometry().entity_ids().len(),
            )
        })
        .collect::<Result<Vec<_>, _>>()?;
    encode(&ListBlockDefinitionsResult {
        definitions: Some(definitions),
    })
}
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    if method == METHOD_SKETCH_BLOCK_DEFINITION_CREATE {
        return create(files, document, crate::decode(method, request)?);
    }
    delete(files, document, crate::decode(method, request)?)
}
fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    request: CreateBlockDefinitionArgs,
) -> Result<Vec<u8>, Error> {
    if request
        .entity_refs
        .as_ref()
        .is_some_and(|ids| !ids.is_empty())
    {
        return Err(Error(format!(
            "block definition {:?} selected entities {:?}: expected an empty selection until block instances are supported",
            request.name, request.entity_refs
        )));
    }
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Create sketch block definition",
            |snapshot, _| {
                snapshot
                    .sketches
                    .define_empty_block(&request.name)
                    .map_err(api_error)
            },
            |_, index| encode(&info(index, &request.name, 0)?),
        )
        .map_err(api_error)
}
fn delete<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    request: DeleteBlockDefinitionArgs,
) -> Result<Vec<u8>, Error> {
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Delete sketch block definition",
            |snapshot, _| {
                snapshot
                    .sketches
                    .delete_block_definition(&request.name)
                    .map_err(api_error)
            },
            |_, ()| encode(&OKResult { ok: true }),
        )
        .map_err(api_error)
}
fn info(index: usize, name: &str, entities: usize) -> Result<SketchBlockDefinitionInfo, Error> {
    Ok(SketchBlockDefinitionInfo {
        index: wire_count(index)?,
        name: name.into(),
        entity_count: wire_count(entities)?,
        instance_count: 0,
    })
}
