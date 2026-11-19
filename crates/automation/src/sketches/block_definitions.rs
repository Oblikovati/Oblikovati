// SPDX-License-Identifier: GPL-2.0-only
//! The canonical block-definition registry on a part document.
use super::*;
use oblikovati_api::types::DOCUMENT_PART;

pub(super) fn active_part<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<u64, Error> {
    crate::part_owner::require_active_kind(files, &[DOCUMENT_PART], "part document")
}
pub(super) fn list<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    let session = files.workspace().session(document).map_err(api_error)?;
    let sketches = &session.snapshot().sketches;
    let definitions = sketches
        .block_definitions()
        .iter()
        .enumerate()
        .map(|(index, definition)| {
            info(
                index,
                definition,
                sketches.block_instance_count(definition.name()),
            )
        })
        .collect::<Result<Vec<_>, _>>()?;
    encode(&ListBlockDefinitionsResult {
        definitions: Some(definitions),
    })
}
pub(super) fn list_instances<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    request: SketchArgs,
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let id = sketch_at(snapshot, request.sketch_index)?;
    let sketch = snapshot.sketches.get(id).map_err(api_error)?;
    let scale = snapshot.parameters.units().working_scale();
    let rows = sketch
        .block_instances()
        .iter()
        .enumerate()
        .map(|(index, instance)| instance_info(snapshot, index, instance, scale))
        .collect::<Result<Vec<_>, Error>>()?;
    encode(&ListBlockInstancesResult {
        instances: Some(rows),
    })
}
pub(super) fn instance_info(
    snapshot: &DocumentSnapshot,
    index: usize,
    instance: &oblikovati_model::document_sketch::SketchBlockInstance,
    scale: f64,
) -> Result<SketchBlockInfo, Error> {
    let definition = snapshot
        .sketches
        .block_definitions()
        .iter()
        .find(|definition| definition.name() == instance.definition())
        .ok_or_else(|| {
            Error(format!(
                "block instance {:?} definition {:?}: expected an existing definition",
                instance.id(),
                instance.definition()
            ))
        })?;
    let position = instance
        .position()
        .into_iter()
        .map(|value| super::entities::centimetres(value, scale))
        .collect::<Result<Vec<_>, _>>()?;
    Ok(SketchBlockInfo {
        index: wire_count(index)?,
        id: instance.id().0,
        definition: instance.definition().into(),
        position: Some(position),
        rotation: instance.rotation(),
        scale: instance.scale(),
        entity_count: wire_count(definition.geometry().entity_ids().len())?,
        color: instance.color().cloned().map(Box::new),
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
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Create sketch block definition",
            |snapshot, reservations| {
                let selected = request.entity_refs.as_deref().unwrap_or_default();
                if selected.is_empty() {
                    return snapshot
                        .sketches
                        .define_empty_block(&request.name)
                        .map_err(api_error);
                }
                let sketch = sketch_at(snapshot, request.source_sketch_index)?;
                let ids: Vec<_> = selected
                    .iter()
                    .copied()
                    .map(oblikovati_model::sketch::SketchEntityId)
                    .collect();
                let instance = reservations.allocate_sketch_entity().map_err(api_error)?;
                snapshot
                    .create_sketch_block(sketch, &request.name, &ids, instance)
                    .map_err(api_error)
            },
            |snapshot, index| {
                let definition = &snapshot.sketches.block_definitions()[index];
                encode(&info(
                    index,
                    definition,
                    snapshot.sketches.block_instance_count(&request.name),
                )?)
            },
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
pub(super) fn info(
    index: usize,
    definition: &oblikovati_model::document_sketch::SketchBlockDefinition,
    instances: usize,
) -> Result<SketchBlockDefinitionInfo, Error> {
    Ok(SketchBlockDefinitionInfo {
        index: wire_count(index)?,
        name: definition.name().into(),
        entity_count: wire_count(definition.geometry().entity_ids().len())?,
        instance_count: wire_count(instances)?,
        color: definition.color().cloned().map(Box::new),
    })
}
