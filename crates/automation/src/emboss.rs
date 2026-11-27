// SPDX-License-Identifier: GPL-2.0-only
//! Resolve Emboss commands and the features.add "emboss" API through certified Part replay.
use oblikovati_api::{
    Error,
    wire::{AddFeatureArgs, featureargs::Emboss},
};
use oblikovati_martensite::certified_sketch_regions_with_datums;
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    feature::{
        FeatureId,
        sketch_emboss::{EmbossType, SketchEmbossDefinition, SketchEmbossRecipe},
        sketch_prism::SketchPrismSelection,
        work_geometry::{DatumAngle, DatumLength, WorkGeometryEvaluation},
    },
    history::{DocumentSnapshot, RecipeReservations},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

const EPSILON: f64 = 1e-9;

/// Validated Emboss ready to append to a private candidate snapshot.
#[derive(Clone)]
pub struct PreparedEmboss {
    name: String,
    definition: SketchEmbossDefinition,
}

impl PreparedEmboss {
    /// Append the recipe to a private candidate. Example: `prepared.create(snapshot, ids)?`.
    pub fn create(
        self,
        snapshot: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        snapshot
            .create_sketch_emboss(&self.name, self.definition, ids)
            .map_err(api_error)
    }

    /// Inspect the prepared authored definition. Example: `prepared.definition()`.
    pub fn definition(&self) -> &SketchEmbossDefinition {
        &self.definition
    }
}

/// Typed inputs for preparing an Emboss definition from desktop or automation callers.
pub struct EmbossInputs<'a> {
    pub sketch: DocumentSketchId,
    pub selection: SketchPrismSelection,
    pub depth: &'a str,
    pub emboss_type: EmbossType,
    pub direction_reversed: bool,
    pub taper: &'a str,
    pub wrap_to_face: bool,
    pub wrap_face: Option<Vec<u8>>,
    pub target_body: Vec<u8>,
}

/// Build and validate an Emboss definition from authored inputs and scalar expressions.
pub fn build_emboss_definition(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    inputs: EmbossInputs<'_>,
) -> Result<SketchEmbossDefinition, Error> {
    let depth = parse_positive_length(snapshot, inputs.depth)?;
    let taper = if inputs.taper.trim().is_empty() {
        DatumAngle::constant(0.0).map_err(api_error)?
    } else {
        parse_taper_angle(snapshot, inputs.taper)?
    };
    let definition = SketchEmbossDefinition::from_recipe(&SketchEmbossRecipe {
        version: 1,
        sketch: inputs.sketch,
        selection: inputs.selection,
        depth,
        emboss_type: inputs.emboss_type,
        direction_reversed: inputs.direction_reversed,
        taper,
        wrap_to_face: inputs.wrap_to_face,
        wrap_face: inputs.wrap_face,
        target_body: inputs.target_body,
    })
    .map_err(Error)?;
    definition
        .resolve_for_replay(snapshot, datums)
        .map_err(Error)?;
    Ok(definition)
}

/// Prepare a named Emboss feature with the next unique `EmbossN` name.
pub fn prepare_emboss_definition(
    snapshot: &DocumentSnapshot,
    datums: &WorkGeometryEvaluation,
    inputs: EmbossInputs<'_>,
) -> Result<PreparedEmboss, Error> {
    let definition = build_emboss_definition(snapshot, datums, inputs)?;
    Ok(PreparedEmboss {
        name: crate::fillet::next_name(snapshot, "Emboss")?,
        definition,
    })
}

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    request: AddFeatureArgs,
) -> Result<Vec<u8>, Error> {
    let input =
        serde_json::from_value::<Emboss>(request.args.0.unwrap_or_default()).map_err(|cause| {
            Error(format!(
                "emboss arguments: expected the canonical Emboss object: {cause}"
            ))
        })?;
    let document = crate::part_owner::active_part(files)?;
    let owner =
        replay.ok_or_else(|| Error("emboss: expected a certified part replay owner".into()))?;
    let policy = owner
        .resolution_policy()
        .ok_or_else(|| Error("emboss: expected a certified resolution policy".into()))?;
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
                "emboss sketch index {}: out of range",
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
                "emboss profile index {}: expected one of {} closed regions",
                input.profile_index,
                regions.len()
            ))
        })?;
    let emboss_type = parse_emboss_type(&input.type_, input.engrave)?;
    let wrap_face = if input.wrap_to_face.trim().is_empty() {
        None
    } else {
        Some(decode_face_key(&input.wrap_to_face)?)
    };
    let prepared = prepare_emboss_definition(
        snapshot,
        &datums,
        EmbossInputs {
            sketch,
            selection: profile.selection(),
            depth: &input.depth,
            emboss_type,
            direction_reversed: false,
            taper: &input.taper,
            wrap_to_face: wrap_face.is_some(),
            wrap_face,
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
        "Emboss",
        |snapshot, ids| prepared.create(snapshot, ids),
    )?;
    crate::fillet::publish_blend_candidate(files, Some(owner), document, candidate, &transition)?;
    Ok(reply)
}

fn parse_emboss_type(raw: &str, engrave: bool) -> Result<EmbossType, Error> {
    let trimmed = raw.trim();
    match (trimmed, engrave) {
        ("", false) | ("fromFace", false) => Ok(EmbossType::EmbossFromFace),
        ("", true) | ("engraveFromFace", _) => Ok(EmbossType::EngraveFromFace),
        ("fromPlane", false) => Ok(EmbossType::EmbossEngraveFromPlane),
        ("fromFace", true) | ("fromPlane", true) => Err(Error(format!(
            "emboss: type {trimmed:?} disagrees with engrave=true"
        ))),
        _ => Err(Error(format!(
            "emboss type {trimmed:?}: expected fromFace, engraveFromFace, or fromPlane"
        ))),
    }
}

fn decode_face_key(raw: &str) -> Result<Vec<u8>, Error> {
    let trimmed = raw.trim();
    if !trimmed.len().is_multiple_of(2) {
        return Err(Error(
            "emboss wrapToFace: expected even-length hex key".into(),
        ));
    }
    let mut bytes = Vec::with_capacity(trimmed.len() / 2);
    for pair in trimmed.as_bytes().as_chunks::<2>().0 {
        let text = std::str::from_utf8(pair)
            .map_err(|_| Error("emboss wrapToFace: expected hex ASCII".into()))?;
        let byte = u8::from_str_radix(text, 16)
            .map_err(|_| Error("emboss wrapToFace: expected hex digits".into()))?;
        bytes.push(byte);
    }
    Ok(bytes)
}

fn parse_positive_length(
    snapshot: &DocumentSnapshot,
    expression: &str,
) -> Result<DatumLength, Error> {
    let normalized = expression.trim().replace(',', ".");
    if normalized.is_empty() {
        return Err(Error(
            "The expression is out of the expected range: expected a positive emboss depth".into(),
        ));
    }
    let length = crate::extrusion::positive_length(snapshot, &normalized, "emboss depth").map_err(
        |cause| {
            Error(format!(
                "The expression is out of the expected range: {cause}"
            ))
        },
    )?;
    let value = length.resolve(&snapshot.parameters).map_err(api_error)?;
    if !value.is_finite() || value <= EPSILON {
        return Err(Error(
            "The expression is out of the expected range: expected a positive length greater than 0"
                .into(),
        ));
    }
    Ok(length)
}

fn parse_taper_angle(snapshot: &DocumentSnapshot, expression: &str) -> Result<DatumAngle, Error> {
    let normalized = expression.trim().replace(',', ".");
    let angle = crate::move_body::angle(snapshot, &normalized, "emboss taper")?;
    let radians = angle.resolve(&snapshot.parameters).map_err(api_error)?;
    if !radians.is_finite() || radians.abs() >= std::f64::consts::FRAC_PI_2 - 1e-4 {
        return Err(Error(format!(
            "emboss taper {expression:?}: expected an angle strictly within (-90 deg, 90 deg)"
        )));
    }
    Ok(angle)
}

fn api_error(error: impl std::fmt::Display) -> Error {
    Error(error.to_string())
}
