// SPDX-License-Identifier: GPL-2.0-only
//! Resolve Coil commands and the features.add "coil" API through certified Part replay.
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::Coil},
};
use oblikovati_martensite::certified_sketch_regions_with_datums;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId,
        sketch_coil::{
            CoilAxisSource, CoilEndTreatment, CoilHandedness, CoilMethod, SketchCoilDefinition,
            SketchCoilRecipe,
        },
        sketch_prism::{PrismBoolean, SketchPrismSelection},
        sketch_revolve::RevolveBooleanRecipe,
        work_axis::WorkAxisReference,
        work_geometry::{DatumAngle, DatumLength, WorkGeometryEvaluation},
    },
    history::{DocumentSnapshot, RecipeReservations},
    quantity::Dimension,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

const EPSILON: f64 = 1e-9;

/// A validated Coil whose inputs are bound to authored sketch, axis, and body identities.
#[derive(Clone)]
pub struct PreparedCoil {
    name: String,
    definition: SketchCoilDefinition,
}

impl PreparedCoil {
    /// Append the recipe to a private candidate. Example: `prepared.create(snapshot, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_sketch_coil(&self.name, self.definition, ids)
            .map_err(api_error)
    }

    /// Inspect the prepared authored definition. Example: `prepared.definition()`.
    pub fn definition(&self) -> &SketchCoilDefinition {
        &self.definition
    }
}

/// Typed inputs for preparing a Coil definition from desktop or automation callers.
pub struct CoilInputs<'a> {
    pub sketch: DocumentSketchId,
    pub selection: SketchPrismSelection,
    pub axis: CoilAxisSource,
    pub axis_reversed: bool,
    pub method: CoilMethod,
    pub pitch: &'a str,
    pub revolutions: &'a str,
    pub height: &'a str,
    pub taper: &'a str,
    pub handedness: CoilHandedness,
    pub start_end: Option<(&'a str, &'a str)>,
    pub end_end: Option<(&'a str, &'a str)>,
    pub boolean_target: Option<(PrismBoolean, Vec<u8>)>,
}

/// Build and validate a Coil definition from authored inputs and scalar expressions.
pub fn build_coil_definition(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    inputs: CoilInputs<'_>,
) -> Result<SketchCoilDefinition, Error> {
    let pitch = matches!(
        inputs.method,
        CoilMethod::PitchAndRevolution | CoilMethod::PitchAndHeight | CoilMethod::Spiral
    )
    .then(|| parse_positive_length(snapshot, inputs.pitch, "coil pitch"))
    .transpose()?;
    let revolutions = matches!(
        inputs.method,
        CoilMethod::PitchAndRevolution | CoilMethod::RevolutionAndHeight | CoilMethod::Spiral
    )
    .then(|| parse_positive_revolutions(snapshot, inputs.revolutions))
    .transpose()?;
    let height = matches!(
        inputs.method,
        CoilMethod::RevolutionAndHeight | CoilMethod::PitchAndHeight
    )
    .then(|| parse_positive_length(snapshot, inputs.height, "coil height"))
    .transpose()?;
    let taper = if inputs.method == CoilMethod::Spiral || inputs.taper.trim().is_empty() {
        DatumAngle::constant(0.0).map_err(api_error)?
    } else {
        parse_taper_angle(snapshot, inputs.taper)?
    };
    let start_end = inputs
        .start_end
        .map(|(flat, trans)| parse_end_treatment(snapshot, flat, trans, "start"))
        .transpose()?;
    let end_end = inputs
        .end_end
        .map(|(flat, trans)| parse_end_treatment(snapshot, flat, trans, "end"))
        .transpose()?;
    let boolean_target = inputs
        .boolean_target
        .map(|(operation, key)| RevolveBooleanRecipe { operation, key });
    let definition = SketchCoilDefinition::from_recipe(&SketchCoilRecipe {
        version: 1,
        sketch: inputs.sketch,
        selection: inputs.selection,
        axis: inputs.axis,
        axis_reversed: inputs.axis_reversed,
        method: inputs.method,
        pitch,
        revolutions,
        height,
        taper,
        handedness: inputs.handedness,
        start_end,
        end_end,
        boolean_target,
    })
    .map_err(Error)?;
    definition
        .resolve_with_datums(snapshot, datums)
        .map_err(Error)?;
    Ok(definition)
}

/// Build a validated Coil definition and reserve its next unique `CoilN` name.
pub fn prepare_coil_definition(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    inputs: CoilInputs<'_>,
) -> Result<PreparedCoil, Error> {
    let definition = build_coil_definition(snapshot, datums, inputs)?;
    Ok(PreparedCoil {
        name: crate::fillet::next_name(snapshot, "Coil")?,
        definition,
    })
}

fn parse_positive_length(
    snapshot: &DocumentSnapshot,
    expression: &str,
    label: &str,
) -> Result<DatumLength, Error> {
    let normalized = expression.trim().replace(',', ".");
    let length = crate::extrusion::positive_length(snapshot, &normalized, label)?;
    let value = length.resolve(&snapshot.parameters).map_err(api_error)?;
    if !value.is_finite() || value <= EPSILON {
        return Err(Error(format!(
            "{label} {expression:?}: expected a positive length greater than 0"
        )));
    }
    Ok(length)
}

fn parse_positive_revolutions(snapshot: &DocumentSnapshot, expression: &str) -> Result<f64, Error> {
    let normalized = expression.trim().replace(',', ".");
    if normalized.is_empty() {
        return Err(Error("coil revolutions: expected a positive number".into()));
    }
    let value = normalized.parse::<f64>().or_else(|_| {
        snapshot
            .parameters
            .evaluate_scalar_expression(&normalized, Dimension::Unitless)
            .map(|q| q.value)
            .map_err(api_error)
    })?;
    if !value.is_finite() || value <= EPSILON {
        return Err(Error(format!(
            "coil revolutions {expression:?}: expected a positive number greater than 0"
        )));
    }
    Ok(value)
}

fn parse_taper_angle(snapshot: &DocumentSnapshot, expression: &str) -> Result<DatumAngle, Error> {
    let normalized = expression.trim().replace(',', ".");
    let angle = crate::move_body::angle(snapshot, &normalized, "coil taper")?;
    let radians = angle.resolve(&snapshot.parameters).map_err(api_error)?;
    if !radians.is_finite() || radians.abs() >= std::f64::consts::FRAC_PI_2 - 1e-4 {
        return Err(Error(format!(
            "coil taper {expression:?}: expected an angle strictly within (-90 deg, 90 deg)"
        )));
    }
    Ok(angle)
}

fn parse_end_treatment(
    snapshot: &DocumentSnapshot,
    flat: &str,
    transition: &str,
    label: &str,
) -> Result<CoilEndTreatment, Error> {
    let parse_nonneg = |expr: &str, kind: &str| -> Result<DatumAngle, Error> {
        let normalized = expr.trim().replace(',', ".");
        if normalized.is_empty() {
            return DatumAngle::constant(0.0).map_err(api_error);
        }
        let angle =
            crate::move_body::angle(snapshot, &normalized, &format!("coil {label} {kind}"))?;
        let rad = angle.resolve(&snapshot.parameters).map_err(api_error)?;
        if !rad.is_finite() || rad < 0.0 {
            return Err(Error(format!(
                "coil {label} {kind} {expr:?}: expected a non-negative angle"
            )));
        }
        Ok(angle)
    };
    Ok(CoilEndTreatment {
        flat_angle: parse_nonneg(flat, "flat angle")?,
        transition_angle: parse_nonneg(transition, "transition angle")?,
    })
}

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input =
        serde_json::from_value::<Coil>(request.args.0.unwrap_or_default()).map_err(|cause| {
            Error(format!(
                "coil arguments: expected the canonical Coil object: {cause}"
            ))
        })?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("coil: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("coil: expected a certified resolution policy".into()))?;
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
                "coil sketch index {}: out of range",
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
                "coil profile index {}: expected one of {} closed regions",
                input.profile_index,
                regions.len()
            ))
        })?;
    let axis = parse_axis_ref(&input.axis_ref)?;
    let method = parse_method(&input)?;
    let handedness = parse_handedness(&input.handedness)?;
    let start_end = (!input.start_flat_angle.trim().is_empty()
        || !input.start_transition_angle.trim().is_empty())
    .then_some((
        input.start_flat_angle.as_str(),
        input.start_transition_angle.as_str(),
    ));
    let end_end = (!input.end_flat_angle.trim().is_empty()
        || !input.end_transition_angle.trim().is_empty())
    .then_some((
        input.end_flat_angle.as_str(),
        input.end_transition_angle.as_str(),
    ));
    let prepared = prepare_coil_definition(
        snapshot,
        &datums,
        CoilInputs {
            sketch,
            selection: profile.selection(),
            axis,
            axis_reversed: false,
            method,
            pitch: &input.pitch,
            revolutions: &input.revolutions,
            height: &input.height,
            taper: &input.taper,
            handedness,
            start_end,
            end_end,
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
        "Coil",
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
            "coil operation {other:?}: expected new, join, cut, or intersect"
        ))),
    }
}

fn parse_axis_ref(axis_ref: &str) -> Result<CoilAxisSource, Error> {
    match axis_ref.trim().to_ascii_lowercase().as_str() {
        "" | "origin/axis/z" | "z" => Ok(CoilAxisSource::WorkAxis(WorkAxisReference::OriginZ)),
        "origin/axis/x" | "x" => Ok(CoilAxisSource::WorkAxis(WorkAxisReference::OriginX)),
        "origin/axis/y" | "y" => Ok(CoilAxisSource::WorkAxis(WorkAxisReference::OriginY)),
        other => Err(Error(format!(
            "coil axisRef {other:?}: expected origin/axis/x, origin/axis/y, or origin/axis/z"
        ))),
    }
}

fn parse_method(input: &Coil) -> Result<CoilMethod, Error> {
    match input.type_.trim().to_ascii_lowercase().as_str() {
        "spiral" => {
            if !input.height.trim().is_empty() || !input.taper.trim().is_empty() {
                return Err(Error(
                    "coil spiral: height and taper do not apply to a flat spiral".into(),
                ));
            }
            Ok(CoilMethod::Spiral)
        }
        "" | "helical" => {
            let p = !input.pitch.trim().is_empty();
            let r = !input.revolutions.trim().is_empty();
            let h = !input.height.trim().is_empty();
            match (p, r, h) {
                (true, true, false) => Ok(CoilMethod::PitchAndRevolution),
                (false, true, true) => Ok(CoilMethod::RevolutionAndHeight),
                (true, false, true) => Ok(CoilMethod::PitchAndHeight),
                _ => Err(Error(
                    "coil: specify exactly two of pitch, revolutions, and height".into(),
                )),
            }
        }
        other => Err(Error(format!(
            "coil type {other:?}: expected helical or spiral"
        ))),
    }
}

fn parse_handedness(value: &str) -> Result<CoilHandedness, Error> {
    match value.trim().to_ascii_lowercase().as_str() {
        "" | "right" => Ok(CoilHandedness::Right),
        "left" => Ok(CoilHandedness::Left),
        other => Err(Error(format!(
            "coil handedness {other:?}: expected right or left"
        ))),
    }
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
