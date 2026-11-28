// SPDX-License-Identifier: GPL-2.0-only
//! Resolve Unwrap commands and features.add "unwrap" through certified Part replay.
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::Unwrap},
};
use oblikovati_model::{
    feature::{
        FeatureId,
        part::PartFeatureDefinition,
        unwrap::{UnwrapAlignment, UnwrapDefinition, UnwrapRecipe},
    },
    history::{DocumentSnapshot, RecipeReservations},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Validated Unwrap ready to append to a private candidate snapshot.
#[derive(Clone)]
pub struct PreparedUnwrap {
    name: String,
    definition: UnwrapDefinition,
}

impl PreparedUnwrap {
    /// Append the recipe to a private candidate. Example: `prepared.create(snapshot, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_unwrap(&self.name, self.definition, ids)
            .map_err(api_error)
    }

    /// Inspect the prepared authored definition. Example: `prepared.definition()`.
    pub fn definition(&self) -> &UnwrapDefinition {
        &self.definition
    }
}

/// Typed inputs for preparing an Unwrap definition from desktop or automation callers.
pub struct UnwrapInputs {
    pub target_body: Vec<u8>,
    pub faces: Vec<Vec<u8>>,
    pub alignment_vertex: Option<Vec<u8>>,
    pub auto_face_chain: bool,
    pub merge_result_body: bool,
    pub alignment: UnwrapAlignment,
    pub linear_result_edges: Vec<Vec<u8>>,
    pub rigid_result_edges: Vec<Vec<u8>>,
    pub seam_edges: Vec<Vec<u8>>,
}

/// Build and validate an Unwrap definition from authored inputs.
pub fn build_unwrap_definition(inputs: UnwrapInputs) -> Result<UnwrapDefinition, Error> {
    UnwrapDefinition::from_recipe(&UnwrapRecipe {
        version: 1,
        target_body: inputs.target_body,
        faces: inputs.faces,
        alignment_vertex: inputs.alignment_vertex,
        auto_face_chain: inputs.auto_face_chain,
        merge_result_body: inputs.merge_result_body,
        alignment: inputs.alignment,
        linear_result_edges: inputs.linear_result_edges,
        rigid_result_edges: inputs.rigid_result_edges,
        seam_edges: inputs.seam_edges,
    })
    .map_err(Error)
}

/// Prepare a named Unwrap feature with the next unique `UnwrapN` name.
pub fn prepare_unwrap_definition(
    snapshot: &DocumentSnapshot,
    inputs: UnwrapInputs,
) -> Result<PreparedUnwrap, Error> {
    let definition = build_unwrap_definition(inputs)?;
    Ok(PreparedUnwrap {
        name: crate::fillet::next_name(snapshot, "Unwrap")?,
        definition,
    })
}

/// Replace an existing Unwrap feature definition in place.
pub fn replace_unwrap(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: UnwrapDefinition,
) -> Result<(), Error> {
    let reads = definition.source_reads();
    snapshot
        .work_geometry
        .replace(
            id,
            PartFeatureDefinition::Unwrap(Box::new(definition)),
            reads,
        )
        .map_err(api_error)
        .map(|_| ())
}

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input = serde_json::from_value::<Unwrap>(request.args.0.unwrap_or_default())
        .map_err(|cause| Error(format!("unwrap arguments: {cause}")))?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("unwrap: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("unwrap: expected a certified resolution policy".into()))?;
    owner.ensure_part(files, document).map_err(Error)?;
    let face = crate::reference_bytes::hex_bytes(&input.face_ref)?;
    let target_body = crate::sweep::sweep_boolean_target(files, owner, document, None)?;
    let workspace = files.workspace();
    let guid = workspace
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let snapshot = workspace.session(document).map_err(api_error)?.snapshot();
    let prepared = prepare_unwrap_definition(
        snapshot,
        UnwrapInputs {
            target_body,
            faces: vec![face],
            alignment_vertex: None,
            auto_face_chain: true,
            merge_result_body: true,
            alignment: UnwrapAlignment::OriginXY,
            linear_result_edges: Vec::new(),
            rigid_result_edges: Vec::new(),
            seam_edges: Vec::new(),
        },
    )?;
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "Unwrap",
        |snapshot, ids| prepared.create(snapshot, ids),
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
