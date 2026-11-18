// SPDX-License-Identifier: GPL-2.0-only
//! Typed Hole creation through the certified part replay owner (#4093).
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::Hole},
};
use oblikovati_model::{
    feature::{FeatureId, hole::HoleDefinition},
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
    let input: Hole = serde_json::from_value(
        request.args.0.unwrap_or_else(|| serde_json::json!({})),
    )
    .map_err(|cause| {
        Error(format!(
            "hole arguments: expected the canonical Hole object: {cause}"
        ))
    })?;
    input::require_supported(&input)?;
    let face = crate::reference_bytes::hex_bytes(&input.face_ref)?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("hole: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("hole: expected a certified part resolution policy".into()))?;
    owner.ensure_part(files, document).map_err(Error)?;
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let body = current_face_body(owner, files, document, guid, &face)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let prepared = prepare_hole(snapshot, body, face, &input)?;
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = create(files, document, guid, policy, prefix, prepared)?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

fn current_face_body<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    owner: &dyn crate::PartUpdateAccess<O, W, F>,
    files: &FileWorkspace<O, W, F>,
    document: u64,
    guid: oblikovati_model::identity::DocumentGuid,
    face: &[u8],
) -> Result<Vec<u8>, Error> {
    let session = files.workspace().session(document).map_err(api_error)?;
    let evaluation = owner.current_part(files, document).map_err(Error)?;
    let mut matches = Vec::new();
    for body in evaluation.ordered_bodies() {
        if body
            .face_reference_keys(session.history(), guid)
            .map_err(Error)?
            .iter()
            .any(|key| key == face)
        {
            matches.push(
                body.body_reference_key(session.history(), guid)
                    .map_err(Error)?
                    .to_vec(),
            );
        }
    }
    match matches.as_slice() {
        [body] => Ok(body.clone()),
        _ => Err(Error(format!(
            "hole face {face:?}, matching bodies {}: expected exactly one current body",
            matches.len()
        ))),
    }
}

/// A validated straight-hole recipe shared by preview and commit.
#[derive(Clone)]
pub struct PreparedHole {
    name: String,
    definition: HoleDefinition,
}
impl PreparedHole {
    /// Add the prepared recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_hole(&self.name, self.definition, ids)
            .map_err(api_error)
    }
}
/// Prepare a straight hole without changing the document.
/// Example: `prepare_hole(snapshot, body_key, face_key, &request)?` before a preview.
pub fn prepare_hole(
    snapshot: &DocumentSnapshot,
    body: Vec<u8>,
    face: Vec<u8>,
    input: &Hole,
) -> Result<PreparedHole, Error> {
    input::require_supported(input)?;
    let recipe = input::recipe(snapshot, body, face, input)?;
    let definition = HoleDefinition::from_recipe(recipe).map_err(Error)?;
    Ok(PreparedHole {
        name: crate::fillet::next_name(snapshot, "Hole")?,
        definition,
    })
}
fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}

mod transaction;
use transaction::create;

#[cfg(test)]
#[path = "../tests/hole/mod.rs"]
mod tests;
