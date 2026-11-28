// SPDX-License-Identifier: GPL-2.0-only
//! Resolve Rib commands and the features.add "rib" API through certified Part replay.
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::Rib},
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId,
        sketch_rib::{
            RibDirection, RibThickenSide, RibThicknessPlane, SketchRibDefinition, SketchRibRecipe,
        },
        work_geometry::{DatumAngle, DatumLength, WorkGeometryEvaluation},
    },
    history::{DocumentSnapshot, RecipeReservations},
    sketch::SketchEntityId,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

const EPSILON: f64 = 1e-9;

/// Validated Rib ready to append to a private candidate snapshot.
#[derive(Clone)]
pub struct PreparedRib {
    name: String,
    definition: SketchRibDefinition,
}

impl PreparedRib {
    /// Append the recipe to a private candidate. Example: `prepared.create(snapshot, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_sketch_rib(&self.name, self.definition, ids)
            .map_err(api_error)
    }

    /// Inspect the prepared authored definition. Example: `prepared.definition()`.
    pub fn definition(&self) -> &SketchRibDefinition {
        &self.definition
    }
}

/// Typed inputs for preparing a Rib definition from desktop or automation callers.
pub struct RibInputs<'a> {
    pub sketch: DocumentSketchId,
    pub profile_curves: Vec<SketchEntityId>,
    pub thickness: &'a str,
    pub depth: Option<&'a str>,
    pub to_next: bool,
    pub direction: RibDirection,
    pub flip_direction: bool,
    pub thicken_side: RibThickenSide,
    pub extend_profile: bool,
    pub draft: &'a str,
    pub thickness_plane: RibThicknessPlane,
    pub target_body: Vec<u8>,
}

/// Build and validate a Rib definition from authored inputs and scalar expressions.
pub fn build_rib_definition(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    inputs: RibInputs<'_>,
) -> Result<SketchRibDefinition, Error> {
    let thickness = parse_positive_length(snapshot, inputs.thickness, "rib thickness")?;
    let depth = match inputs.depth {
        Some(expr) if !expr.trim().is_empty() && !inputs.to_next => {
            Some(parse_positive_length(snapshot, expr, "rib depth")?)
        }
        _ => None,
    };
    let draft = if inputs.draft.trim().is_empty() {
        DatumAngle::constant(0.0).map_err(api_error)?
    } else {
        parse_draft_angle(snapshot, inputs.draft)?
    };
    let definition = SketchRibDefinition::from_recipe(&SketchRibRecipe {
        version: 1,
        sketch: inputs.sketch,
        profile_curves: inputs.profile_curves,
        thickness,
        depth,
        to_next: inputs.to_next,
        direction: inputs.direction,
        flip_direction: inputs.flip_direction,
        thicken_side: inputs.thicken_side,
        extend_profile: inputs.extend_profile,
        draft,
        thickness_plane: inputs.thickness_plane,
        target_body: inputs.target_body,
    })
    .map_err(Error)?;
    definition
        .resolve_for_replay(snapshot, datums)
        .map_err(Error)?;
    Ok(definition)
}

/// Prepare a named Rib feature with the next unique `RibN` name.
pub fn prepare_rib_definition(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    inputs: RibInputs<'_>,
) -> Result<PreparedRib, Error> {
    let definition = build_rib_definition(snapshot, datums, inputs)?;
    Ok(PreparedRib {
        name: crate::fillet::next_name(snapshot, "Rib")?,
        definition,
    })
}

/// Replace an existing Rib feature in place with a validated updated definition.
pub fn replace_rib(
    snapshot: &mut DocumentSnapshot,
    feature: FeatureId,
    definition: SketchRibDefinition,
) -> Result<(), Error> {
    let reads = definition.reads(snapshot).map_err(Error)?;
    snapshot
        .work_geometry
        .replace(
            feature,
            oblikovati_model::feature::part::PartFeatureDefinition::SketchRib(Box::new(definition)),
            reads,
        )
        .map(|_| ())
        .map_err(api_error)
}

/// Collect open non-construction line entities from a document sketch in authored order.
pub fn open_profile_lines(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
) -> Vec<SketchEntityId> {
    let Ok(entry) = snapshot.sketches.get(sketch) else {
        return Vec::new();
    };
    let geometry = entry.geometry();
    geometry
        .lines()
        .iter()
        .filter(|line| !geometry.is_construction(line.id).unwrap_or(false))
        .map(|line| line.id)
        .collect()
}

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input =
        serde_json::from_value::<Rib>(request.args.0.unwrap_or_default()).map_err(|cause| {
            Error(format!(
                "rib arguments: expected the canonical Rib object: {cause}"
            ))
        })?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("rib: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("rib: expected a certified resolution policy".into()))?;
    owner.ensure_part(files, document).map_err(Error)?;
    let datums = owner.datum_evaluation(files, document).map_err(Error)?;
    let target_body = crate::sweep::sweep_boolean_target(files, owner, document, None)?;
    let workspace = files.workspace();
    let guid = workspace
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let snapshot = workspace.session(document).map_err(api_error)?.snapshot();
    let sketch = usize::try_from(input.sketch_index)
        .ok()
        .and_then(|index| snapshot.sketches.sketches().get(index))
        .map(|sketch| sketch.id())
        .ok_or_else(|| {
            Error(format!(
                "rib sketch index {}: out of range",
                input.sketch_index
            ))
        })?;
    let curves = open_profile_lines(snapshot, sketch);
    let curve = usize::try_from(input.profile_index)
        .ok()
        .and_then(|index| curves.get(index).copied())
        .ok_or_else(|| {
            Error(format!(
                "rib profile index {}: expected one of {} open lines",
                input.profile_index,
                curves.len()
            ))
        })?;
    let thicken_side = parse_thicken_side(&input.thicken_side)?;
    let thickness_plane = parse_thickness_plane(&input.thickness_plane)?;
    let to_next = input.to_next || input.depth.trim().is_empty();
    let prepared = prepare_rib_definition(
        snapshot,
        &datums,
        RibInputs {
            sketch,
            profile_curves: vec![curve],
            thickness: &input.thickness,
            depth: (!input.depth.trim().is_empty()).then_some(input.depth.as_str()),
            to_next,
            direction: RibDirection::NormalToSketchPlane,
            flip_direction: false,
            thicken_side,
            extend_profile: input.extend_profile,
            draft: &input.draft,
            thickness_plane,
            target_body,
        },
    )?;
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "Rib",
        |snapshot, ids| prepared.create(snapshot, ids),
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

fn parse_thicken_side(raw: &str) -> Result<RibThickenSide, Error> {
    match raw.trim() {
        "" | "symmetric" => Ok(RibThickenSide::Symmetric),
        "side1" => Ok(RibThickenSide::Side1),
        "side2" => Ok(RibThickenSide::Side2),
        other => Err(Error(format!(
            "rib thickenSide {other:?}: expected symmetric, side1, or side2"
        ))),
    }
}

fn parse_thickness_plane(raw: &str) -> Result<RibThicknessPlane, Error> {
    match raw.trim() {
        "" | "sketch" | "atTop" => Ok(RibThicknessPlane::AtTop),
        "root" | "atRoot" => Ok(RibThicknessPlane::AtRoot),
        other => Err(Error(format!(
            "rib thicknessPlane {other:?}: expected sketch or root"
        ))),
    }
}

fn parse_positive_length(
    snapshot: &DocumentSnapshot,
    expression: &str,
    field: &str,
) -> Result<DatumLength, Error> {
    let normalized = expression.trim().replace(',', ".");
    if normalized.is_empty() {
        return Err(Error(format!(
            "The expression is out of the expected range: expected a positive {field}"
        )));
    }
    let length =
        crate::extrusion::positive_length(snapshot, &normalized, field).map_err(|cause| {
            Error(format!(
                "The expression is out of the expected range: {cause}"
            ))
        })?;
    let value = length.resolve(&snapshot.parameters).map_err(api_error)?;
    if !value.is_finite() || value <= EPSILON {
        return Err(Error(
            "The expression is out of the expected range: expected a positive length greater than 0"
                .into(),
        ));
    }
    Ok(length)
}

fn parse_draft_angle(snapshot: &DocumentSnapshot, expression: &str) -> Result<DatumAngle, Error> {
    let normalized = expression.trim().replace(',', ".");
    let angle = crate::move_body::angle(snapshot, &normalized, "rib draft")?;
    let radians = angle.resolve(&snapshot.parameters).map_err(api_error)?;
    if !radians.is_finite() || radians.abs() >= std::f64::consts::FRAC_PI_2 - 1e-4 {
        return Err(Error(format!(
            "rib draft {expression:?}: expected an angle strictly within (-90 deg, 90 deg)"
        )));
    }
    Ok(angle)
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
