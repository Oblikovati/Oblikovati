// SPDX-License-Identifier: GPL-2.0-only
//! Typed Shell creation through the certified part replay owner (#4106).
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::Shell},
};
use oblikovati_model::{
    feature::{
        FeatureId,
        shell::{ShellDefinition, ShellDirection, ShellFaceThickness},
    },
    history::{DocumentSnapshot, RecipeReservations},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input: Shell = serde_json::from_value(
        request.args.0.unwrap_or_else(|| serde_json::json!({})),
    )
    .map_err(|cause| {
        Error(format!(
            "shell arguments: expected the canonical Shell object: {cause}"
        ))
    })?;
    let openings = opening_keys(&input)?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("shell: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("shell: expected a certified part resolution policy".into()))?;
    owner.ensure_part(files, document).map_err(Error)?;
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let body = opening_body(owner, files, document, guid, &openings, &input)?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let prepared = prepare_shell(snapshot, body, openings, &input)?;
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "shell",
        |candidate, ids| prepared.create(candidate, ids),
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

fn opening_keys(input: &Shell) -> Result<Vec<Vec<u8>>, Error> {
    if input
        .faces_geom
        .as_ref()
        .is_some_and(|faces| !faces.is_empty())
    {
        return Err(Error("shell facesGeom: expected named face references; geometric selectors are not supported".into()));
    }
    let refs = input.face_refs.as_deref().unwrap_or_default();
    if refs.is_empty() {
        return Err(Error(
            "shell faceRefs []: expected at least one opening face".into(),
        ));
    }
    refs.iter()
        .map(|key| crate::reference_bytes::hex_bytes(key))
        .collect()
}

fn opening_body<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    owner: &dyn crate::PartUpdateAccess<O, W, F>,
    files: &FileWorkspace<O, W, F>,
    document: u64,
    guid: oblikovati_model::identity::DocumentGuid,
    openings: &[Vec<u8>],
    input: &Shell,
) -> Result<Vec<u8>, Error> {
    let mut faces = openings.to_vec();
    for entry in input.face_thicknesses.as_deref().unwrap_or_default() {
        faces.push(crate::reference_bytes::hex_bytes(&entry.face_ref)?);
    }
    let mut body = None;
    for face in faces {
        let candidate = crate::hole::current_face_body(owner, files, document, guid, &face)?;
        if body.as_ref().is_some_and(|key| key != &candidate) {
            return Err(Error(format!(
                "shell face {face:?}: expected all faces on the same current body"
            )));
        }
        body = Some(candidate);
    }
    body.ok_or_else(|| Error("shell faceRefs []: expected at least one opening face".into()))
}

/// A Shell recipe that can be evaluated on a private candidate.
#[derive(Clone)]
pub struct PreparedShell {
    name: String,
    definition: ShellDefinition,
}
impl PreparedShell {
    /// Add the recipe to a private candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_shell(&self.name, self.definition, ids)
            .map_err(api_error)
    }
}

/// Prepare exact Shell inputs; example: `prepare_shell(snapshot, body, openings, &input)?`.
pub fn prepare_shell(
    snapshot: &DocumentSnapshot,
    body: Vec<u8>,
    openings: Vec<Vec<u8>>,
    input: &Shell,
) -> Result<PreparedShell, Error> {
    let direction = match input.direction.trim().to_ascii_lowercase().as_str() {
        "" | "inside" => ShellDirection::Inside,
        "outside" => ShellDirection::Outside,
        "both" => ShellDirection::Both,
        _ => {
            return Err(Error(format!(
                "shell direction {:?}: expected inside, outside, or both",
                input.direction
            )));
        }
    };
    let thickness =
        crate::extrusion::positive_length(snapshot, &input.thickness, "shell thickness")?;
    let overrides = input
        .face_thicknesses
        .as_deref()
        .unwrap_or_default()
        .iter()
        .map(|entry| {
            Ok(ShellFaceThickness {
                face: crate::reference_bytes::hex_bytes(&entry.face_ref)?,
                thickness: crate::extrusion::positive_length(
                    snapshot,
                    &entry.thickness,
                    "shell face thickness",
                )?,
            })
        })
        .collect::<Result<Vec<_>, Error>>()?;
    let definition =
        ShellDefinition::new(body, openings, thickness, direction, overrides).map_err(Error)?;
    Ok(PreparedShell {
        name: crate::fillet::next_name(snapshot, "Shell")?,
        definition,
    })
}
fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}

#[cfg(test)]
#[path = "../tests/shell/mod.rs"]
mod tests;
