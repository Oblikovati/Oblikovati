// SPDX-License-Identifier: GPL-2.0-only
//! Place a block definition through the typed sketch API.
use super::*;
use oblikovati_model::{document_sketch::SketchBlockInstance, sketch::SketchEntityId};

struct PreparedBlockPlacement {
    sketch: DocumentSketchId,
    definition: String,
    position: [f64; 2],
    rotation: f64,
    scale: f64,
}
pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: AddSketchBlockArgs,
) -> Result<Vec<u8>, Error> {
    let document = super::block_definitions::active_part(files)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let placement = prepare(snapshot, request)?;
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Place sketch block",
            |snapshot, reservations| placement.commit(snapshot, reservations),
            |_, id| encode(&AddSketchBlockResult { entity_id: id.0 }),
        )
        .map_err(api_error)
}
fn prepare(
    snapshot: &DocumentSnapshot,
    request: AddSketchBlockArgs,
) -> Result<PreparedBlockPlacement, Error> {
    let sketch = sketch_at(snapshot, request.sketch_index)?;
    require_definition(snapshot, &request.definition)?;
    let position = super::coordinates::working_pair(
        request.position.as_deref().unwrap_or_default(),
        snapshot.parameters.units().working_scale(),
        "block instance position",
    )?;
    let rotation = rotation(snapshot, &request.rotation_angle)?;
    let scale = if request.scale == 0. {
        1.
    } else {
        request.scale
    };
    let placement = PreparedBlockPlacement {
        sketch,
        definition: request.definition,
        position,
        rotation,
        scale,
    };
    placement.instance(SketchEntityId(1))?;
    Ok(placement)
}
fn require_definition(snapshot: &DocumentSnapshot, name: &str) -> Result<(), Error> {
    if snapshot
        .sketches
        .block_definitions()
        .iter()
        .any(|item| item.name() == name)
    {
        return Ok(());
    }
    Err(Error(format!(
        "block instance definition {name:?}: expected an existing block definition"
    )))
}
fn rotation(snapshot: &DocumentSnapshot, source: &str) -> Result<f64, Error> {
    if source.is_empty() {
        return Ok(0.);
    }
    super::transform::rotation_angle(&snapshot.parameters, source)
        .map_err(|cause| Error(format!("block instance rotation {source:?}: {cause}")))
}
impl PreparedBlockPlacement {
    fn commit(
        &self,
        snapshot: &mut DocumentSnapshot,
        reservations: &mut oblikovati_model::history::RecipeReservations<'_>,
    ) -> Result<SketchEntityId, Error> {
        let id = reservations.allocate_sketch_entity().map_err(api_error)?;
        snapshot
            .sketches
            .insert_block_instance(self.sketch, self.instance(id)?)
            .map_err(api_error)?;
        Ok(id)
    }
    fn instance(&self, id: SketchEntityId) -> Result<SketchBlockInstance, Error> {
        SketchBlockInstance::new(
            id,
            &self.definition,
            self.position,
            self.rotation,
            self.scale,
        )
        .map_err(api_error)
    }
}
