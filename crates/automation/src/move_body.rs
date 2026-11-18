// SPDX-License-Identifier: GPL-2.0-only
//! Typed body motion through the certified part replay owner (#4097).
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::MoveBody},
};
use oblikovati_model::{
    feature::{FeatureId, move_body::MoveBodyDefinition},
    history::{DocumentSnapshot, RecipeReservations},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
mod input;

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input: MoveBody = serde_json::from_value(
        request.args.0.unwrap_or_else(|| serde_json::json!({})),
    )
    .map_err(|cause| {
        Error(format!(
            "moveBody arguments: expected the canonical MoveBody object: {cause}"
        ))
    })?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("moveBody: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("moveBody: expected a certified part resolution policy".into()))?;
    owner.ensure_part(files, document).map_err(Error)?;
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let session = files.workspace().session(document).map_err(api_error)?;
    let evaluation = owner.current_part(files, document).map_err(Error)?;
    let body = usize::try_from(input.body_index)
        .ok()
        .and_then(|index| evaluation.ordered_bodies().nth(index))
        .ok_or_else(|| {
            Error(format!(
                "moveBody bodyIndex {}: expected a current body index in 0..{}",
                input.body_index,
                evaluation.ordered_bodies().count()
            ))
        })?;
    let key = body
        .body_reference_key(session.history(), guid)
        .map_err(Error)?
        .to_vec();
    let prepared = prepare_move_body(session.snapshot(), key, &input)?;
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "move body",
        |candidate, ids| prepared.create(candidate, ids),
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}
/// A validated body motion recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedMoveBody {
    name: String,
    definition: MoveBodyDefinition,
}
impl PreparedMoveBody {
    /// Add a motion to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_move_body(&self.name, self.definition, ids)
            .map_err(api_error)
    }
}
/// Prepare body motion without changing the document; example: `prepare_move_body(snapshot, body, &request)?`.
pub fn prepare_move_body(
    snapshot: &DocumentSnapshot,
    body: Vec<u8>,
    input: &MoveBody,
) -> Result<PreparedMoveBody, Error> {
    let definition =
        MoveBodyDefinition::new(body, input::operations(snapshot, input)?).map_err(Error)?;
    Ok(PreparedMoveBody {
        name: crate::fillet::next_name(snapshot, "Move")?,
        definition,
    })
}
fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
#[cfg(test)]
#[path = "../tests/move_body/mod.rs"]
mod tests;
