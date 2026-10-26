// SPDX-License-Identifier: GPL-2.0-only
use crate::{Error, PartTopologyAccess, encode, part_owner::active_part};
use oblikovati_api::wire::{
    AddFeatureArgs, FeatureInfo, METHOD_FEATURES_ADD, featureargs::Extrude,
};
use oblikovati_model::{
    feature::{
        AuthoredFeature,
        sketch_prism::{SketchDirectedCurve, SketchPrismDefinition, SketchPrismSelection},
        work_geometry::DatumLength,
    },
    history::DocumentSnapshot,
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};

/// Create the native prism form of the public extrude operation and refresh its retained result.
pub(crate) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = active_part(files)?;
    let owner = access
        .as_deref_mut()
        .ok_or_else(|| missing_owner(document))?;
    let request: AddFeatureArgs = crate::decode(METHOD_FEATURES_ADD, encoded)?;
    let extrude = decode_extrude(&request)?;
    let feature = create(files, document, &extrude)?;
    let errors = owner.update(files, document, false).map_err(Error)?;
    encode(&feature_info(files, document, feature, &errors)?)
}

fn decode_extrude(request: &AddFeatureArgs) -> Result<Extrude, Error> {
    if request.kind != "extrude" {
        return Err(Error(format!(
            "features.add kind {:?}: expected \"extrude\"",
            request.kind
        )));
    }
    let value = request.args.0.as_ref().ok_or_else(|| {
        Error("features.add extrude args absent: expected an extrude argument object".into())
    })?;
    serde_json::from_value(value.clone()).map_err(|cause| {
        Error(format!(
            "features.add extrude args {value:?}: expected valid extrude inputs: {cause}"
        ))
    })
}

fn create<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    request: &Extrude,
) -> Result<u64, Error> {
    files
        .commands(document)
        .map_err(api_error)?
        .run("Create Extrude", |snapshot, ids| {
            let definition = definition(snapshot, request)?;
            let id = ids.allocate_feature().map_err(api_error)?;
            let feature = AuthoredFeature::sketch_prism(id, "Extrude", definition, snapshot)
                .map_err(api_error)?;
            snapshot.work_geometry.append(feature).map_err(api_error)?;
            Ok::<u64, Error>(id.0)
        })
        .map_err(api_error)
}

fn definition(
    snapshot: &DocumentSnapshot,
    request: &Extrude,
) -> Result<SketchPrismDefinition, Error> {
    require_supported_extent(request)?;
    let sketch = sketch_at(snapshot, request.sketch_index)?;
    let selection = rectangle_selection(snapshot, sketch, request)?;
    let source = distance_source(snapshot, request)?;
    SketchPrismDefinition::new(
        sketch,
        selection,
        [
            DatumLength::Constant(0.),
            DatumLength::Constant(0.),
            DatumLength::Parameter(source),
        ],
    )
    .map_err(api_error)
}

fn rectangle_selection(
    snapshot: &DocumentSnapshot,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
    request: &Extrude,
) -> Result<SketchPrismSelection, Error> {
    let lines = snapshot
        .sketches
        .get(sketch)
        .map_err(api_error)?
        .geometry()
        .profile_curve_ids();
    require_rectangle_profile(request, &lines)?;
    Ok(SketchPrismSelection {
        outer: lines
            .into_iter()
            .map(|curve| SketchDirectedCurve {
                curve,
                reversed: false,
            })
            .collect(),
        holes: vec![],
    })
}

fn require_rectangle_profile(
    request: &Extrude,
    lines: &[oblikovati_model::sketch::SketchEntityId],
) -> Result<(), Error> {
    if lines.len() == 4 && request.profile_index == 0 {
        return Ok(());
    }
    Err(Error(format!(
        "extrude sketch {} profile {} with {} profile curves: expected profile 0 with one four-line rectangle",
        request.sketch_index,
        request.profile_index,
        lines.len()
    )))
}

fn distance_source(snapshot: &DocumentSnapshot, request: &Extrude) -> Result<u64, Error> {
    snapshot
        .parameters
        .find(request.distance.trim())
        .ok_or_else(|| {
            Error(format!(
                "extrude distance {:?}: expected the name of an existing length parameter",
                request.distance
            ))
        })
}

fn require_supported_extent(request: &Extrude) -> Result<(), Error> {
    let supported_operation = request.operation.is_empty() || request.operation == "new";
    let supported_extent = request.extent.is_empty() || request.extent == "distance";
    let supported_direction = request.direction.is_empty() || request.direction == "positive";
    if supported_operation
        && supported_extent
        && supported_direction
        && request.second_distance.is_empty()
        && request.taper.is_empty()
        && request.to_face.is_empty()
        && request.from_face.is_empty()
        && request.to_face_geom.is_none()
        && request.from_face_geom.is_none()
        && request.profile_seeds.is_none()
    {
        return Ok(());
    }
    Err(Error(format!(
        "extrude {request:?}: expected operation new, distance extent, positive direction, one named distance parameter, and no profile seeds or face targets"
    )))
}

fn sketch_at(
    snapshot: &DocumentSnapshot,
    index: i64,
) -> Result<oblikovati_model::document_sketch::DocumentSketchId, Error> {
    usize::try_from(index)
        .ok()
        .and_then(|position| snapshot.sketches.sketches().get(position))
        .map(|sketch| sketch.id())
        .ok_or_else(|| {
            Error(format!(
                "extrude sketch index {index}: expected an ordered sketch index in 0..{}",
                snapshot.sketches.sketches().len()
            ))
        })
}

fn feature_info<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    id: u64,
    errors: &[crate::FeatureError],
) -> Result<FeatureInfo, Error> {
    let snapshot = files
        .workspace()
        .session(document)
        .map_err(api_error)?
        .snapshot();
    let feature = snapshot
        .work_geometry
        .get(oblikovati_model::feature::FeatureId(id))
        .map_err(api_error)?;
    Ok(FeatureInfo {
        id,
        name: feature.name().into(),
        kind: feature.definition().kind().into(),
        suppressed: feature.suppressed(),
        health: feature_health(id, errors),
    })
}

fn feature_health(id: u64, errors: &[crate::FeatureError]) -> String {
    errors
        .iter()
        .find(|error| error.id == id)
        .map(|error| error.message.clone())
        .unwrap_or_default()
}

fn missing_owner(document: u64) -> Error {
    Error(format!(
        "features.add document {document}: expected an injected exact topology owner"
    ))
}

fn api_error(cause: impl std::fmt::Display) -> Error {
    let message = cause.to_string();
    Error(message)
}
