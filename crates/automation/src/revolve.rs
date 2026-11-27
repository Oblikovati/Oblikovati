// SPDX-License-Identifier: GPL-2.0-only
//! Resolve Revolve commands and the features.add API through certified Part replay.
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::Revolve},
};
use oblikovati_martensite::{PrismResolutionPolicy, certified_sketch_regions_with_datums};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId,
        sketch_prism::{PrismBoolean, SketchPrismSelection},
        sketch_revolve::{SketchRevolveDefinition, SketchRevolveDirection},
        work_geometry::{DatumAngle, WorkGeometryEvaluation},
    },
    history::{DocumentSnapshot, RecipeReservations},
    quantity::Dimension,
    sketch::SketchEntityId,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
use std::f64::consts::TAU;

const ANGLE_EPSILON: f64 = 1e-9;
/// Owned scalar role name for the primary Revolve Angle A model parameter.
pub const REVOLVE_ANGLE_ROLE: &str = "angle";
/// Owned scalar role name for the Asymmetric Revolve Angle B model parameter.
pub const REVOLVE_SECOND_ANGLE_ROLE: &str = "angle-b";

/// A validated Revolve whose inputs are bound to authored sketch and body identities.
#[derive(Clone)]
pub struct PreparedRevolve {
    name: String,
    definition: SketchRevolveDefinition,
    owned_angle: Option<String>,
    owned_second_angle: Option<String>,
}

impl PreparedRevolve {
    /// Append the recipe to a private candidate. Example: `prepared.create(snapshot, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        let mut definition = self.definition;
        let first = self
            .owned_angle
            .as_deref()
            .map(|source| insert_owned_angle(snapshot, ids, source))
            .transpose()?;
        let second = self
            .owned_second_angle
            .as_deref()
            .map(|source| insert_owned_angle(snapshot, ids, source))
            .transpose()?;
        if let Some(parameter) = first {
            definition = definition
                .with_angle(DatumAngle::parameter(parameter))
                .map_err(api_error)?;
        }
        if let Some(parameter) = second {
            definition = definition
                .with_asymmetric_angle(DatumAngle::parameter(parameter))
                .map_err(api_error)?;
        }
        let feature = snapshot
            .create_sketch_revolve(&self.name, definition, ids)
            .map_err(api_error)?;
        for (role, parameter) in [
            (REVOLVE_ANGLE_ROLE, first),
            (REVOLVE_SECOND_ANGLE_ROLE, second),
        ] {
            if let Some(parameter) = parameter {
                snapshot
                    .work_geometry
                    .assign_scalar_owner(feature, role, parameter)
                    .map_err(api_error)?;
            }
        }
        Ok(feature)
    }

    /// Inspect the prepared authored definition. Example: `prepared.definition()`.
    pub fn definition(&self) -> &SketchRevolveDefinition {
        &self.definition
    }
}

fn insert_owned_angle(
    snapshot: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    source: &str,
) -> Result<u64, Error> {
    let parameter = ids.allocate().map_err(api_error)?;
    snapshot
        .parameters
        .insert_auto_model_scalar(parameter, source, Dimension::Angle)
        .map_err(api_error)?;
    Ok(parameter)
}

/// Normalize a user-typed angle expression, appending the document's preferred angle unit to bare numbers.
pub fn canonical_angle_expression(snapshot: &DocumentSnapshot, expression: &str) -> String {
    let normalized = expression.trim().replace(',', ".");
    if normalized.parse::<f64>().is_ok() {
        let unit = snapshot.parameters.units().preferred_name(Dimension::Angle);
        format!("{normalized} {unit}")
    } else {
        normalized
    }
}

/// Inputs for preparing a typed Revolve definition from desktop or automation callers.
pub struct RevolveInputs<'a> {
    pub sketch: DocumentSketchId,
    pub selection: SketchPrismSelection,
    pub axis: Option<SketchEntityId>,
    pub full: bool,
    pub direction: SketchRevolveDirection,
    pub angle: &'a str,
    pub second_angle: Option<&'a str>,
    pub surface: bool,
    pub boolean_target: Option<(PrismBoolean, Vec<u8>)>,
}

/// Discover the sole visible, unambiguous closed profile sketch region for a new Revolve draft.
/// Example: `automatic_revolve_profile(snapshot, &datums, policy)`.
pub fn automatic_revolve_profile(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    policy: PrismResolutionPolicy,
) -> Option<(DocumentSketchId, SketchPrismSelection)> {
    let collect = |only_unconsumed: bool| {
        let mut found = Vec::new();
        for sketch in snapshot
            .sketches
            .sketches()
            .iter()
            .filter(|sketch| sketch.visible())
            .filter(|sketch| {
                !only_unconsumed
                    || snapshot
                        .work_geometry
                        .sketch_consumers(sketch.id())
                        .next()
                        .is_none()
            })
        {
            if let Ok(regions) =
                certified_sketch_regions_with_datums(snapshot, sketch.id(), policy, datums)
            {
                for region in regions {
                    found.push((sketch.id(), region.selection()));
                }
            }
        }
        found
    };
    let found = collect(true);
    let found = if found.is_empty() {
        collect(false)
    } else {
        found
    };
    if found.len() == 1 {
        found.into_iter().next()
    } else {
        None
    }
}

/// Build and validate a Revolve recipe from authored selections and angle expressions.
/// Example: `prepare_revolve_definition(snapshot, &datums, inputs)?`.
pub fn prepare_revolve_definition(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    inputs: RevolveInputs<'_>,
) -> Result<PreparedRevolve, Error> {
    let (primary, owned_angle) = if inputs.full {
        (DatumAngle::constant(TAU).map_err(api_error)?, None)
    } else {
        let parsed = parse_positive_turn_angle(snapshot, inputs.angle, "revolve angle")?;
        let owned = parsed
            .parameter_id()
            .is_none()
            .then(|| canonical_angle_expression(snapshot, inputs.angle));
        (parsed, owned)
    };
    let mut definition = SketchRevolveDefinition::new(inputs.sketch, inputs.selection, primary)
        .map_err(api_error)?;
    if let Some(axis) = inputs.axis {
        definition = definition.with_axis(axis).map_err(api_error)?;
    }
    let mut owned_second_angle = None;
    if inputs.full {
        definition = definition.with_full_turn();
    } else if inputs.direction == SketchRevolveDirection::Asymmetric {
        let second_expr = inputs
            .second_angle
            .ok_or_else(|| Error("revolve angle B: expected an angular expression".into()))?;
        let second = parse_positive_turn_angle(snapshot, second_expr, "revolve angle B")?;
        let sum = primary.resolve(&snapshot.parameters).map_err(api_error)?
            + second.resolve(&snapshot.parameters).map_err(api_error)?;
        if sum > TAU + ANGLE_EPSILON {
            return Err(Error("Out of Range: combined angle exceeds 360 deg".into()));
        }
        owned_second_angle = second
            .parameter_id()
            .is_none()
            .then(|| canonical_angle_expression(snapshot, second_expr));
        definition = definition
            .with_asymmetric_angle(second)
            .map_err(api_error)?;
    } else {
        definition = definition.with_direction(inputs.direction);
    }
    if inputs.surface {
        definition = definition.with_surface_output(true);
    } else if let Some((operation, key)) = inputs.boolean_target {
        definition = definition
            .with_boolean_target(operation, key)
            .map_err(api_error)?;
    }
    definition
        .resolve_with_datums(snapshot, datums)
        .map_err(api_error)?;
    let prefix = if inputs.surface {
        "RevolutionSrf"
    } else {
        "Revolution"
    };
    Ok(PreparedRevolve {
        name: crate::fillet::next_name(snapshot, prefix)?,
        definition,
        owned_angle,
        owned_second_angle,
    })
}

/// Parse an angular expression and enforce the Revolve (0, 360 deg] bounds.
/// Example: `parse_positive_turn_angle(snapshot, "90 deg", "revolve angle")?`.
pub fn parse_positive_turn_angle(
    snapshot: &DocumentSnapshot,
    expression: &str,
    label: &str,
) -> Result<DatumAngle, Error> {
    let normalized = expression.trim().replace(',', ".");
    let angle = crate::move_body::angle(snapshot, &normalized, label)?;
    let radians = angle.resolve(&snapshot.parameters).map_err(api_error)?;
    if !radians.is_finite() || radians <= ANGLE_EPSILON {
        return Err(Error(format!(
            "{label} {expression:?}: expected a positive angle greater than 0 deg"
        )));
    }
    if radians > TAU + ANGLE_EPSILON {
        return Err(Error(format!(
            "Out of Range: {label} {expression:?} exceeds 360 deg"
        )));
    }
    Ok(angle)
}

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input =
        serde_json::from_value::<Revolve>(request.args.0.unwrap_or_default()).map_err(|cause| {
            Error(format!(
                "revolve arguments: expected the canonical Revolve object: {cause}"
            ))
        })?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("revolve: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("revolve: expected a certified resolution policy".into()))?;
    owner.ensure_part(files, document).map_err(Error)?;
    let datums = owner.datum_evaluation(files, document).map_err(Error)?;
    let boolean_op = parse_operation(&input.operation)?;
    let target = boolean_op
        .map(|op| {
            crate::sweep::sweep_boolean_target(files, owner, document, None).map(|key| (op, key))
        })
        .transpose()?;
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
                "revolve sketch index {}: out of range",
                input.sketch_index
            ))
        })?;
    let regions = certified_sketch_regions_with_datums(snapshot, sketch, policy, &datums)
        .map_err(api_error)?;
    let profile = usize::try_from(input.profile_index)
        .ok()
        .and_then(|index| regions.get(index))
        .ok_or_else(|| {
            Error(format!(
                "revolve profile index {}: expected one of {} closed regions",
                input.profile_index,
                regions.len()
            ))
        })?;
    let axis = if input.about_centerline || input.axis_ref.trim().is_empty() {
        None
    } else {
        Some(parse_axis_entity(&input.axis_ref)?)
    };
    let (direction, second_angle) = parse_direction(&input.direction, &input.angle2)?;
    let full = input.angle.trim().is_empty()
        || parse_positive_turn_angle(snapshot, &input.angle, "revolve angle")
            .and_then(|a| a.resolve(&snapshot.parameters).map_err(api_error))
            .is_ok_and(|rad| (rad - TAU).abs() <= ANGLE_EPSILON && second_angle.is_none());
    let prepared = prepare_revolve_definition(
        snapshot,
        &datums,
        RevolveInputs {
            sketch,
            selection: profile.selection(),
            axis,
            full,
            direction,
            angle: &input.angle,
            second_angle,
            surface: false,
            boolean_target: target,
        },
    )?;
    let prefix = owner.candidate_prefix(files, document).map_err(Error)?;
    let ((reply, candidate), transition) = crate::part_feature_transaction::create(
        files,
        document,
        guid,
        policy,
        prefix,
        "Revolution",
        |snapshot, ids| prepared.create(snapshot, ids),
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

fn parse_operation(operation: &str) -> Result<Option<PrismBoolean>, Error> {
    match operation.trim() {
        "" | "new" => Ok(None),
        "join" => Ok(Some(PrismBoolean::Join)),
        "cut" => Ok(Some(PrismBoolean::Cut)),
        "intersect" => Ok(Some(PrismBoolean::Intersect)),
        other => Err(Error(format!(
            "revolve operation {other:?}: expected new, join, cut, or intersect"
        ))),
    }
}

fn parse_direction<'a>(
    direction: &str,
    angle2: &'a str,
) -> Result<(SketchRevolveDirection, Option<&'a str>), Error> {
    if !angle2.trim().is_empty() {
        return Ok((SketchRevolveDirection::Asymmetric, Some(angle2)));
    }
    match direction.trim() {
        "" | "positive" => Ok((SketchRevolveDirection::Positive, None)),
        "negative" => Ok((SketchRevolveDirection::Negative, None)),
        "symmetric" => Ok((SketchRevolveDirection::Symmetric, None)),
        other => Err(Error(format!(
            "revolve direction {other:?}: expected positive, negative, or symmetric"
        ))),
    }
}

fn parse_axis_entity(axis_ref: &str) -> Result<SketchEntityId, Error> {
    let raw = axis_ref
        .trim()
        .strip_prefix("entity/")
        .unwrap_or(axis_ref.trim());
    let id = raw.parse::<u64>().map_err(|_| {
        Error(format!(
            "revolve axisRef {axis_ref:?}: expected a sketch line entity id"
        ))
    })?;
    Ok(SketchEntityId(id))
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
