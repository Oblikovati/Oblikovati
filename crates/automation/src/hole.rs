// SPDX-License-Identifier: GPL-2.0-only
//! Typed Hole creation through the certified part replay owner (#4093).
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::Hole},
};
use oblikovati_model::{
    feature::{
        FeatureId,
        hole::{
            HoleDefinition, HoleDrillPoint, HoleExtent, HoleFastenerSpec, HoleKind, HoleOptions,
            HolePosition, HoleRecipe, HoleSeat, HoleTerminationSurface, HoleThreadSpec,
        },
        work_geometry::DatumLength,
    },
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
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "hole",
        |candidate, ids| prepared.create(candidate, ids),
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

pub(crate) fn current_face_body<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
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
        for key in body
            .face_reference_keys(session.history(), guid)
            .map_err(Error)?
        {
            if key == face {
                matches.push(
                    body.body_reference_key(session.history(), guid)
                        .map_err(Error)?
                        .to_vec(),
                );
            }
        }
    }
    match matches.as_slice() {
        [body] => Ok(body.clone()),
        _ => Err(Error(format!(
            "face {face:?}, matching current faces {}: expected exactly one current face",
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
    /// Inspect the validated Hole definition; example: `prepared.definition()`.
    pub fn definition(&self) -> &HoleDefinition {
        &self.definition
    }
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
/// Termination choice for desktop/automation Hole creation and editing.
#[derive(Clone, Debug, PartialEq)]
pub enum HoleTerminationInput {
    ThroughAll,
    Distance(String),
    ToSurface(HoleTerminationSurface),
}

/// Seat parameters for desktop/automation Hole creation and editing.
#[derive(Clone, Debug, Default, PartialEq)]
pub enum HoleSeatInput {
    #[default]
    None,
    Counterbore {
        diameter: String,
        depth: String,
    },
    Spotface {
        diameter: String,
        depth: String,
    },
    Countersink {
        diameter: String,
        angle_deg: f64,
    },
}

/// Typed desktop/automation inputs for creating or editing any v1/v2 Hole.
#[derive(Clone, Debug, PartialEq)]
pub struct HoleFeatureInputs {
    pub body: Vec<u8>,
    pub face: Vec<u8>,
    pub position: Option<[f64; 3]>,
    pub diameter: String,
    pub termination: HoleTerminationInput,
    pub kind: HoleKind,
    pub seat: HoleSeatInput,
    pub drill_point: HoleDrillPoint,
    pub direction_flipped: bool,
    pub extend_start: bool,
    pub fastener: Option<HoleFastenerSpec>,
    pub thread: Option<HoleThreadSpec>,
}

/// Build a validated v1/v2 Hole definition from desktop/automation inputs.
pub fn build_hole_definition(
    snapshot: &DocumentSnapshot,
    inputs: &HoleFeatureInputs,
) -> Result<HoleDefinition, Error> {
    let diameter = crate::extrusion::positive_length(snapshot, &inputs.diameter, "hole diameter")?;
    let extent = match &inputs.termination {
        HoleTerminationInput::ThroughAll => HoleExtent::ThroughAll,
        HoleTerminationInput::Distance(expr) => HoleExtent::Distance(
            crate::extrusion::positive_length(snapshot, expr, "hole depth")?,
        ),
        HoleTerminationInput::ToSurface(surface) => HoleExtent::ToSurface(surface.clone()),
    };
    let seat = match &inputs.seat {
        HoleSeatInput::None => HoleSeat::None,
        HoleSeatInput::Counterbore { diameter, depth } => HoleSeat::Counterbore {
            diameter: crate::extrusion::positive_length(snapshot, diameter, "seat diameter")?,
            depth: crate::extrusion::positive_length(snapshot, depth, "seat depth")?,
        },
        HoleSeatInput::Spotface { diameter, depth } => HoleSeat::Spotface {
            diameter: crate::extrusion::positive_length(snapshot, diameter, "seat diameter")?,
            depth: crate::extrusion::positive_length(snapshot, depth, "seat depth")?,
        },
        HoleSeatInput::Countersink {
            diameter,
            angle_deg,
        } => HoleSeat::Countersink {
            diameter: crate::extrusion::positive_length(snapshot, diameter, "seat diameter")?,
            angle_deg: *angle_deg,
        },
    };
    let position = match inputs.position {
        Some(point) => HolePosition::ModelPoint(point.map(DatumLength::Constant)),
        None => HolePosition::FaceCentroid,
    };
    let options = HoleOptions {
        kind: inputs.kind,
        seat,
        drill_point: inputs.drill_point,
        direction_flipped: inputs.direction_flipped,
        extend_start: inputs.extend_start,
        fastener: inputs.fastener.clone(),
        thread: inputs.thread.clone(),
    };
    let mut recipe = HoleRecipe {
        version: 1,
        body: inputs.body.clone(),
        face: inputs.face.clone(),
        position,
        diameter,
        extent,
        options,
    };
    recipe.version = recipe.canonical_version();
    let definition = HoleDefinition::from_recipe(recipe).map_err(Error)?;
    definition
        .resolve(&snapshot.parameters)
        .map_err(|cause| Error(cause.to_string()))?;
    Ok(definition)
}

/// Prepare a new v1/v2 Hole feature without mutating the document.
pub fn prepare_hole_definition(
    snapshot: &DocumentSnapshot,
    inputs: &HoleFeatureInputs,
) -> Result<PreparedHole, Error> {
    let definition = build_hole_definition(snapshot, inputs)?;
    Ok(PreparedHole {
        name: crate::fillet::next_name(snapshot, "Hole")?,
        definition,
    })
}

/// Replace an existing Hole feature's definition inside a transaction/preview candidate.
pub fn replace_hole(
    snapshot: &mut DocumentSnapshot,
    id: FeatureId,
    definition: HoleDefinition,
) -> Result<(), Error> {
    snapshot.edit_hole(id, definition).map_err(api_error)
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}

#[cfg(test)]
#[path = "../tests/hole/mod.rs"]
mod tests;
