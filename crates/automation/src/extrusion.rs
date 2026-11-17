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
        sketch_prism::{SketchPrismDefinition, SketchPrismSelection},
        work_geometry::DatumLength,
    },
    history::DocumentSnapshot,
    identity::DocumentGuid,
    quantity::Dimension,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Geometry capability used before committing an Extrude command.
pub trait ExtrudeRegionAccess {
    /// Resolve a current profile index to durable authored lineage; example: `backend.region(snapshot, sketch, 0)?`.
    fn region(
        &self,
        snapshot: &DocumentSnapshot,
        sketch: DocumentSketchId,
        index: usize,
    ) -> Result<SketchPrismSelection, String>;
    /// Validate the solved candidate and return the current body count; example: `backend.validate(candidate, feature)?`.
    fn validate(
        &self,
        candidate: &DocumentSnapshot,
        document: DocumentGuid,
        feature: FeatureId,
    ) -> Result<usize, String>;
    /// Replay a feature and return its failure reason, or an empty string when healthy or suppressed.
    /// Example: `backend.feature_health(candidate, feature)?`.
    fn feature_health(
        &self,
        _candidate: &DocumentSnapshot,
        _document: DocumentGuid,
        feature: FeatureId,
    ) -> Result<String, String> {
        Err(format!(
            "feature {feature:?}: expected a feature replay service"
        ))
    }
}

pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    backend: Option<&dyn ExtrudeRegionAccess>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let request: wire::AddFeatureArgs = crate::decode(wire::METHOD_FEATURES_ADD, encoded)?;
    let input = parse_request(request)?;
    let document = crate::part_owner::active_part(files)?;
    let backend = backend.ok_or_else(|| {
        Error("extrude: expected an injected certified profile and validation service".into())
    })?;
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let prepared = prepare_extrusion(snapshot, backend, &input)?;
    create(files, document, backend, prepared)
}

/// Validated authored inputs shared by preview and commit.
pub struct PreparedExtrusion {
    name: String,
    definition: SketchPrismDefinition,
}
impl PreparedExtrusion {
    /// Create the feature on a caller-owned candidate; example: `prepared.create(candidate, ids)?`.
    pub fn create(
        self,
        candidate: &mut DocumentSnapshot,
        ids: &mut oblikovati_model::history::RecipeReservations<'_>,
    ) -> Result<FeatureId, Error> {
        candidate
            .create_sketch_prism(&self.name, self.definition, ids)
            .map_err(api_error)
    }
}
/// Resolve a canonical Extrude request against current geometry; example: `prepare_extrusion(snapshot, backend, &input)?`.
pub fn prepare_extrusion(
    snapshot: &DocumentSnapshot,
    backend: &dyn ExtrudeRegionAccess,
    input: &Extrude,
) -> Result<PreparedExtrusion, Error> {
    require_supported(input)?;
    let sketch = sketch_index(snapshot, input.sketch_index)?;
    let index = usize::try_from(input.profile_index).map_err(|_| {
        Error(format!(
            "profile index {}: expected a nonnegative index",
            input.profile_index
        ))
    })?;
    let depth = distance(snapshot, &input.distance)?;
    let selection = backend.region(snapshot, sketch, index).map_err(Error)?;
    let definition = SketchPrismDefinition::new(
        sketch,
        selection,
        [DatumLength::Constant(0.), DatumLength::Constant(0.), depth],
    )
    .map_err(api_error)?;
    let name = next_name(snapshot)?;
    Ok(PreparedExtrusion { name, definition })
}

fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    backend: &dyn ExtrudeRegionAccess,
    prepared: PreparedExtrusion,
) -> Result<Vec<u8>, Error> {
    let name = prepared.name.clone();
    let guid = files
        .workspace()
        .identity(document)
        .map_err(api_error)?
        .internal_name;
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Create extrusion",
            |snapshot, ids| prepared.create(snapshot, ids),
            |candidate, id| {
                let bodies = backend.validate(candidate, guid, id).map_err(Error)?;
                let feature = candidate.work_geometry.get(id).map_err(api_error)?;
                crate::encode(&serde_json::json!({
                    "id": id.0, "name": feature.name(), "kind": feature.definition().kind(),
                    "suppressed": candidate.feature_suppressed(id).map_err(api_error)?,
                    "health": "", "feature": name, "bodies": bodies, "healthy": true
                }))
            },
        )
        .map_err(api_error)
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
            &["new", "newbody", "new-body"][..],
        ),
        ("extent", input.extent.as_str(), &["distance"][..]),
        ("direction", input.direction.as_str(), &["positive"][..]),
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
    if !input.second_distance.is_empty()
        || !input.taper.is_empty()
        || !input.to_face.is_empty()
        || input.to_face_geom.is_some()
        || !input.from_face.is_empty()
        || input.from_face_geom.is_some()
        || input
            .profile_seeds
            .as_ref()
            .is_some_and(|seeds| !seeds.is_empty())
    {
        return Err(Error("extrude extent inputs: expected one positive distance without taper, target faces, or profile seeds".into()));
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
    let parameters = &snapshot.parameters;
    let text = source.trim();
    if let Some(id) = parameters.find(text) {
        let quantity = parameters.model_value(id).map_err(api_error)?;
        require_positive(quantity.value, quantity.dimension, source, label)?;
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
    require_positive(quantity.value, quantity.dimension, source, label)?;
    Ok(DatumLength::Constant(quantity.value))
}
/// Check an extrusion expression with the same rules as feature creation.
/// Example: `validate_extrude_distance(&snapshot, "10 mm")?`.
pub fn validate_extrude_distance(snapshot: &DocumentSnapshot, source: &str) -> Result<(), Error> {
    distance(snapshot, source).map(|_| ())
}
fn require_positive(
    value: f64,
    dimension: Dimension,
    source: &str,
    label: &str,
) -> Result<(), Error> {
    if dimension != Dimension::Length || !value.is_finite() || value <= 0. {
        return Err(Error(format!(
            "{label} {source:?} resolved to {value:?} {dimension:?}: expected a finite positive length"
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
