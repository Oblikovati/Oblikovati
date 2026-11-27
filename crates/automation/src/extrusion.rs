// SPDX-License-Identifier: GPL-2.0-only
//! Typed feature creation through a document transaction and an injected geometry owner.
use oblikovati_api::{
    Error,
    wire::{self, featureargs::Extrude},
};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    expression::Expression,
    feature::{
        FeatureId,
        sketch_prism::{
            PrismBoolean, SketchPrismDefinition, SketchPrismDirection, SketchPrismSelection,
        },
        work_geometry::{DatumLength, WorkGeometryEvaluation},
    },
    history::{DocumentSnapshot, RecipeReservations},
    identity::DocumentGuid,
    quantity::Dimension,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{
    TransactionObserver, files::FileWorkspace, part_evaluation::CurrentPartDatums,
    workspace::WorkspaceObserver,
};

mod boolean;
mod extent;
mod region_access;
pub use region_access::ExtrudeRegionAccess;
mod replay;
use replay::{ExtrudeProjection, adopt_extrude_candidate, prepare_owned_prefix};

pub(super) const DISTANCE_ROLE: &str = "distance";
pub(super) const SECOND_DISTANCE_ROLE: &str = "distance-b";

/// A current named result prefix for certified extrusion validation.
pub type ExtrudeReplayPrefix = oblikovati_session::OwnedPartReplay;
/// Detached named bodies produced by an extrusion candidate.
pub type CapturedExtrudeCandidate = oblikovati_session::OwnedPartCandidate;
/// The candidate body count and its captured results.
pub type ExtrudePrefixReplay = (usize, CapturedExtrudeCandidate);
type PolicyExtrudePrefix = (
    oblikovati_martensite::PrismResolutionPolicy,
    ExtrudeReplayPrefix,
);

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn ExtrudeRegionAccess>,
    replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let request: wire::AddFeatureArgs = crate::decode(wire::METHOD_FEATURES_ADD, encoded)?;
    let input = parse_request(request)?;
    let document = crate::part_owner::active_part(files)?;
    let backend = backend.ok_or_else(|| {
        Error("extrude: expected an injected certified profile and validation service".into())
    })?;
    let prepared = prepare_owned_extrusion(files, document, backend, &input, replay)?;
    create(files, document, backend, prepared)
}

/// Resolve an extrusion for preview or commit, including named targets and sketch datums.
/// Example: `prepare_owned_extrusion(files, document, backend, &request, Some(owner))?`.
pub fn prepare_owned_extrusion<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    backend: &dyn ExtrudeRegionAccess,
    input: &Extrude,
    mut replay: Option<&mut dyn crate::PartUpdateAccess<O, W, F>>,
) -> Result<PreparedExtrusion, Error> {
    let profile_datums = match replay.as_mut() {
        Some(owner) => current_profile_datums(files, document, input, &mut **owner)?,
        None => None,
    };
    let target = boolean::operation(input)
        .map(|operation| {
            boolean::target(files, document, replay, operation).map(|key| (operation, key))
        })
        .transpose()?;
    let session = files.workspace().session(document).map_err(api_error)?;
    let datums = profile_datums
        .as_ref()
        .map(|current| current.current(session))
        .transpose()
        .map_err(api_error)?;
    prepare_extrusion_for_target(session.snapshot(), backend, input, target, datums)
}

fn current_profile_datums<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    input: &Extrude,
    owner: &mut dyn crate::PartUpdateAccess<O, W, F>,
) -> Result<Option<CurrentPartDatums>, Error> {
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    if !crate::requires_part_replay(snapshot) {
        return Ok(None);
    }
    let sketch = sketch_index(snapshot, input.sketch_index)?;
    if !snapshot
        .sketch_inspection_needs_work_geometry(sketch)
        .map_err(api_error)?
    {
        return Ok(None);
    }
    owner
        .current_part_datums(files, document)
        .map(Some)
        .map_err(Error)
}

/// Validated authored inputs shared by preview and commit.
pub struct PreparedExtrusion {
    name: String,
    definition: SketchPrismDefinition,
    owned_distance: Option<String>,
    owned_second_distance: Option<String>,
}
impl PreparedExtrusion {
    /// Create the feature on a caller-owned candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        candidate: &mut DocumentSnapshot,
        ids: &mut RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        let mut definition = self.definition;
        let first = self
            .owned_distance
            .as_deref()
            .map(|source| insert_owned_distance(candidate, ids, source))
            .transpose()?;
        let second = self
            .owned_second_distance
            .as_deref()
            .map(|source| insert_owned_distance(candidate, ids, source))
            .transpose()?;
        if let Some(parameter) = first {
            definition = definition
                .with_normal_source(DatumLength::Parameter(parameter))
                .map_err(api_error)?;
        }
        if let Some(parameter) = second {
            definition = definition
                .with_second_depth(DatumLength::Parameter(parameter))
                .map_err(api_error)?;
        }
        let feature = candidate
            .create_sketch_prism(&self.name, definition, ids)
            .map_err(api_error)?;
        for (role, parameter) in [(DISTANCE_ROLE, first), (SECOND_DISTANCE_ROLE, second)] {
            if let Some(parameter) = parameter {
                candidate
                    .work_geometry
                    .assign_scalar_owner(feature, role, parameter)
                    .map_err(api_error)?;
            }
        }
        Ok(feature)
    }
}

fn insert_owned_distance(
    candidate: &mut DocumentSnapshot,
    ids: &mut RecipeReservations<'_>,
    source: &str,
) -> Result<u64, Error> {
    let parameter = ids.allocate().map_err(api_error)?;
    candidate
        .parameters
        .insert_auto_model_scalar(parameter, source, Dimension::Length)
        .map_err(api_error)?;
    Ok(parameter)
}
/// Resolve a canonical Extrude request against current geometry; example: `prepare_extrusion(snapshot, backend, &input)?`.
pub fn prepare_extrusion(
    snapshot: &DocumentSnapshot,
    backend: &dyn ExtrudeRegionAccess,
    input: &Extrude,
) -> Result<PreparedExtrusion, Error> {
    prepare_extrusion_for_target(snapshot, backend, input, None, None)
}

fn prepare_extrusion_for_target(
    snapshot: &DocumentSnapshot,
    backend: &dyn ExtrudeRegionAccess,
    input: &Extrude,
    target: Option<(PrismBoolean, Vec<u8>)>,
    datums: Option<&WorkGeometryEvaluation>,
) -> Result<PreparedExtrusion, Error> {
    require_supported(input)?;
    if let Some(operation) = boolean::operation(input).filter(|_| target.is_none()) {
        return Err(Error(format!(
            "extrude {operation}: expected a named target body"
        )));
    }
    let mut prepared = prepare_prism(snapshot, backend, input, datums)?;
    if let Some((operation, key)) = target {
        prepared.definition = prepared
            .definition
            .with_boolean_target(operation, key)
            .map_err(api_error)?;
    }
    Ok(prepared)
}

fn prepare_prism(
    snapshot: &DocumentSnapshot,
    backend: &dyn ExtrudeRegionAccess,
    input: &Extrude,
    datums: Option<&WorkGeometryEvaluation>,
) -> Result<PreparedExtrusion, Error> {
    let extent = extent::reference_extent(snapshot, input)?;
    let sketch = sketch_index(snapshot, input.sketch_index)?;
    let index = usize::try_from(input.profile_index).map_err(|_| {
        Error(format!(
            "profile index {}: expected a nonnegative index",
            input.profile_index
        ))
    })?;
    let resolve_profile = || {
        match datums {
            Some(datums) => backend.region_with_datums(snapshot, sketch, index, datums),
            None => backend.region(snapshot, sketch, index),
        }
        .map_err(Error)
    };
    if let Some(extent) = extent {
        if !input.taper.trim().is_empty() {
            return Err(Error("Extrude Taper A requires a distance extent.".into()));
        }
        return Ok(PreparedExtrusion {
            name: next_name(snapshot)?,
            definition: SketchPrismDefinition::between_planes(sketch, resolve_profile()?, extent)
                .map_err(api_error)?,
            owned_distance: None,
            owned_second_distance: None,
        });
    }
    prepare_distance_prism(snapshot, input, sketch, resolve_profile)
}

fn prepare_distance_prism(
    snapshot: &DocumentSnapshot,
    input: &Extrude,
    sketch: DocumentSketchId,
    resolve_profile: impl FnOnce() -> Result<SketchPrismSelection, Error>,
) -> Result<PreparedExtrusion, Error> {
    let depth = distance(snapshot, &input.distance)?;
    let owned_distance =
        matches!(depth, DatumLength::Constant(_)).then(|| input.distance.trim().to_owned());
    let definition = SketchPrismDefinition::new(
        sketch,
        resolve_profile()?,
        [DatumLength::Constant(0.), DatumLength::Constant(0.), depth],
    )
    .map_err(api_error)?;
    let direction = match input.direction.trim().to_ascii_lowercase().as_str() {
        "negative" => SketchPrismDirection::Negative,
        "symmetric" => SketchPrismDirection::Symmetric,
        _ => SketchPrismDirection::Positive,
    };
    let second = (!input.second_distance.is_empty())
        .then(|| positive_length(snapshot, &input.second_distance, "extrude Distance B"))
        .transpose()?;
    let owned_second_distance = matches!(second, Some(DatumLength::Constant(_)))
        .then(|| input.second_distance.trim().to_owned());
    let taper = (!input.taper.trim().is_empty())
        .then(|| crate::move_body::angle(snapshot, &input.taper, "Extrude Taper A"))
        .transpose()?;
    let definition = match second {
        Some(depth) => definition.with_second_depth(depth),
        None => definition.with_direction(direction),
    }
    .and_then(|definition| definition.with_taper(taper))
    .map_err(api_error)?;
    let name = next_name(snapshot)?;
    Ok(PreparedExtrusion {
        name,
        definition,
        owned_distance,
        owned_second_distance,
    })
}

fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    backend: &dyn ExtrudeRegionAccess,
    prepared: PreparedExtrusion,
) -> Result<Vec<u8>, Error> {
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    let prefix = prepare_owned_prefix(files, document, backend)?;
    let name = prepared.name.clone();
    let projection = ExtrudeProjection {
        backend,
        guid,
        name: &name,
        prefix: prefix.as_ref(),
    };
    let ((response, captured), transition) = files
        .commands(document)
        .map_err(api_error)?
        .run_private_candidate(
            "Create extrusion",
            |snapshot, ids| prepared.create(snapshot, ids),
            |candidate, id, history| projection.project(candidate, id, history),
        )
        .map_err(api_error)?;
    if let (Some((policy, _)), Some(captured)) = (prefix, captured) {
        adopt_extrude_candidate(files, document, policy, captured, &transition)?;
    }
    Ok(response)
}

fn parse_request(request: wire::AddFeatureArgs) -> Result<Extrude, Error> {
    if request.kind != wire::featureargs::KIND_EXTRUDE {
        return Err(Error(format!(
            "feature kind {:?}: expected extrude",
            request.kind
        )));
    }
    let value = request.args.0.unwrap_or_else(|| serde_json::json!({}));
    let input: Extrude = serde_json::from_value(value).map_err(|cause| {
        Error(format!(
            "extrude arguments: expected the canonical Extrude object: {cause}"
        ))
    })?;
    Ok(input)
}
fn require_supported(input: &Extrude) -> Result<(), Error> {
    for (field, value, supported) in [
        (
            "operation",
            input.operation.as_str(),
            &["new", "newbody", "new-body", "cut", "join"][..],
        ),
        (
            "direction",
            input.direction.as_str(),
            &["positive", "negative", "symmetric"][..],
        ),
    ] {
        let normalized = value.trim();
        if !normalized.is_empty()
            && !supported
                .iter()
                .any(|allowed| normalized.eq_ignore_ascii_case(allowed))
        {
            return Err(Error(format!(
                "extrude {field} {value:?}: expected one of {supported:?}"
            )));
        }
    }
    if input.to_face_geom.is_some()
        || input.from_face_geom.is_some()
        || input
            .profile_seeds
            .as_ref()
            .is_some_and(|seeds| !seeds.is_empty())
    {
        return Err(Error("extrude extent inputs: expected an extent without geometric face selectors or profile seeds".into()));
    }
    Ok(())
}
fn sketch_index(snapshot: &DocumentSnapshot, index: i64) -> Result<DocumentSketchId, Error> {
    usize::try_from(index)
        .ok()
        .and_then(|index| snapshot.sketches.sketches().get(index))
        .map(|sketch| sketch.id())
        .ok_or_else(|| {
            Error(format!(
                "sketch index {index}: expected an index in 0..{}",
                snapshot.sketches.sketches().len()
            ))
        })
}
pub(super) fn distance(snapshot: &DocumentSnapshot, source: &str) -> Result<DatumLength, Error> {
    positive_length(snapshot, source, "extrude distance")
}
pub(crate) fn positive_length(
    snapshot: &DocumentSnapshot,
    source: &str,
    label: &str,
) -> Result<DatumLength, Error> {
    let length = finite_length(snapshot, source, label)?;
    let quantity = length.resolve(&snapshot.parameters).map_err(api_error)?;
    require_positive(quantity, Dimension::Length, source, label)?;
    Ok(length)
}
pub(crate) fn finite_length(
    snapshot: &DocumentSnapshot,
    source: &str,
    label: &str,
) -> Result<DatumLength, Error> {
    let parameters = &snapshot.parameters;
    let text = source.trim();
    if let Some(id) = parameters.find(text) {
        let quantity = parameters.model_value(id).map_err(api_error)?;
        require_finite_length(quantity.value, quantity.dimension, source, label)?;
        return Ok(DatumLength::Parameter(id));
    }
    let expression = Expression::parse(text, parameters.units()).map_err(api_error)?;
    if !expression.references().is_empty() {
        return Err(Error(format!(
            "{label} {source:?}: expected a parameter name or a reference-free length expression"
        )));
    }
    let mut quantity = expression.evaluate().map_err(api_error)?;
    if quantity.dimension == Dimension::Unitless {
        quantity = parameters
            .units()
            .from_preferred(quantity.value, Dimension::Length);
    }
    require_finite_length(quantity.value, quantity.dimension, source, label)?;
    Ok(DatumLength::Constant(quantity.value))
}
/// Check an extrusion expression with the same rules as feature creation.
/// Example: `validate_extrude_distance(&snapshot, "10 mm")?`.
pub fn validate_extrude_distance(snapshot: &DocumentSnapshot, source: &str) -> Result<(), Error> {
    distance(snapshot, source).map(|_| ())
}
/// Evaluate the current distance in database units using the feature's expression rules.
pub fn evaluated_extrude_distance(snapshot: &DocumentSnapshot, source: &str) -> Result<f64, Error> {
    distance(snapshot, source)?
        .resolve(&snapshot.parameters)
        .map_err(api_error)
}
fn require_positive(
    value: f64,
    dimension: Dimension,
    source: &str,
    label: &str,
) -> Result<(), Error> {
    require_finite_length(value, dimension, source, label)?;
    if value <= 0. {
        return Err(Error(format!(
            "{label} {source:?} resolved to {value:?} {dimension:?}: expected a finite positive length"
        )));
    }
    Ok(())
}
fn require_finite_length(
    value: f64,
    dimension: Dimension,
    source: &str,
    label: &str,
) -> Result<(), Error> {
    if dimension != Dimension::Length || !value.is_finite() {
        return Err(Error(format!(
            "{label} {source:?} resolved to {value:?} {dimension:?}: expected a finite length"
        )));
    }
    Ok(())
}
fn next_name(snapshot: &DocumentSnapshot) -> Result<String, Error> {
    let limit = snapshot
        .work_geometry
        .features()
        .len()
        .checked_add(1)
        .ok_or_else(|| Error("feature count: expected room for another feature name".into()))?;
    for index in 1..=limit {
        let candidate = format!("Extrusion{index}");
        if snapshot
            .work_geometry
            .features()
            .iter()
            .all(|feature| feature.name() != candidate)
        {
            return Ok(candidate);
        }
    }
    Err(Error(format!(
        "feature names through {limit}: expected an available extrusion name"
    )))
}
fn api_error(cause: impl std::fmt::Display) -> Error {
    Error(cause.to_string())
}

#[cfg(test)]
#[path = "extrusion/validation_tests.rs"]
mod validation_tests;
