// SPDX-License-Identifier: GPL-2.0-only
//! Resolve Decal commands and the features.add "decal" API through certified Part replay.
use oblikovati_api::{Error, wire::AddFeatureArgs};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId,
        sketch_decal::{SketchDecalDefinition, SketchDecalRecipe},
        work_geometry::WorkGeometryEvaluation,
    },
    history::{DocumentSnapshot, RecipeReservations},
    sketch::SketchEntityId,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Validated Decal ready to append to a private candidate snapshot.
#[derive(Clone)]
pub struct PreparedDecal {
    name: String,
    definition: SketchDecalDefinition,
}

impl PreparedDecal {
    /// Append the recipe to a private candidate. Example: `prepared.create(snapshot, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_sketch_decal(&self.name, self.definition, ids)
            .map_err(api_error)
    }

    /// Inspect the prepared authored definition. Example: `prepared.definition()`.
    pub fn definition(&self) -> &SketchDecalDefinition {
        &self.definition
    }
}

/// Typed inputs for preparing a Decal definition from desktop or automation callers.
pub struct DecalInputs {
    pub sketch: DocumentSketchId,
    pub image: SketchEntityId,
    pub target_face: Vec<u8>,
    pub target_body: Vec<u8>,
    pub chain_faces: bool,
    pub wrap_to_face: bool,
}

/// Build and validate a Decal definition from authored inputs.
pub fn build_decal_definition(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    inputs: DecalInputs,
) -> Result<SketchDecalDefinition, Error> {
    let definition = SketchDecalDefinition::from_recipe(&SketchDecalRecipe {
        version: 1,
        sketch: inputs.sketch,
        image: inputs.image,
        target_face: inputs.target_face,
        target_body: inputs.target_body,
        chain_faces: inputs.chain_faces,
        wrap_to_face: inputs.wrap_to_face,
    })
    .map_err(Error)?;
    definition
        .resolve_for_replay(snapshot, datums)
        .map_err(Error)?;
    Ok(definition)
}

/// Prepare a named Decal feature with the next unique `DecalN` name.
pub fn prepare_decal_definition(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    inputs: DecalInputs,
) -> Result<PreparedDecal, Error> {
    let definition = build_decal_definition(snapshot, datums, inputs)?;
    Ok(PreparedDecal {
        name: crate::fillet::next_name(snapshot, "Decal")?,
        definition,
    })
}

/// Replace an existing Decal feature in place with a validated updated definition.
pub fn replace_decal(
    snapshot: &mut DocumentSnapshot,
    feature: FeatureId,
    definition: SketchDecalDefinition,
) -> Result<(), Error> {
    let reads = definition.reads(snapshot).map_err(Error)?;
    snapshot
        .work_geometry
        .replace(
            feature,
            oblikovati_model::feature::part::PartFeatureDefinition::SketchDecal(Box::new(
                definition,
            )),
            reads,
        )
        .map(|_| ())
        .map_err(api_error)
}

#[derive(serde::Deserialize)]
#[serde(rename_all = "camelCase")]
struct DecalArgs {
    sketch_index: i64,
    #[serde(default)]
    image_id: Option<u64>,
    face_ref: String,
    #[serde(default = "default_true")]
    chain_faces: bool,
    #[serde(default)]
    wrap_to_face: bool,
}

fn default_true() -> bool {
    true
}

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input = serde_json::from_value::<DecalArgs>(request.args.0.unwrap_or_default()).map_err(
        |cause| {
            Error(format!(
                "decal arguments: expected sketchIndex and faceRef: {cause}"
            ))
        },
    )?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("decal: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("decal: expected a certified resolution policy".into()))?;
    owner.ensure_part(files, document).map_err(Error)?;
    let datums = owner.datum_evaluation(files, document).map_err(Error)?;
    let target_face = crate::reference_bytes::hex_bytes(&input.face_ref)?;
    let target_body = crate::sweep::sweep_boolean_target(files, owner, document, None)?;
    let workspace = files.workspace();
    let guid = workspace
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let snapshot = workspace.session(document).map_err(api_error)?.snapshot();
    let placed = usize::try_from(input.sketch_index)
        .ok()
        .and_then(|index| snapshot.sketches.sketches().get(index))
        .ok_or_else(|| {
            Error(format!(
                "decal sketch index {}: out of range",
                input.sketch_index
            ))
        })?;
    let image = match input.image_id {
        Some(id) => SketchEntityId(id),
        None => placed
            .images()
            .first()
            .map(|item| item.id)
            .ok_or_else(|| Error("decal sketch: expected an inserted image".into()))?,
    };
    let prepared = prepare_decal_definition(
        snapshot,
        &datums,
        DecalInputs {
            sketch: placed.id(),
            image,
            target_face,
            target_body,
            chain_faces: input.chain_faces,
            wrap_to_face: input.wrap_to_face,
        },
    )?;
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "Decal",
        |snapshot, ids| prepared.create(snapshot, ids),
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
