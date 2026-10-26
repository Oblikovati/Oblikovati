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
    sketch::{Sketch2D, SketchEntityId},
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
    let selection = profile_selection(snapshot, sketch, request)?;
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

fn profile_selection(
    snapshot: &DocumentSnapshot,
    sketch: oblikovati_model::document_sketch::DocumentSketchId,
    request: &Extrude,
) -> Result<SketchPrismSelection, Error> {
    let geometry = snapshot.sketches.get(sketch).map_err(api_error)?.geometry();
    let curves = geometry.profile_curve_ids();
    require_profile_index(request, &curves)?;
    Ok(SketchPrismSelection {
        outer: ordered_profile(geometry, &curves)?,
        holes: vec![],
    })
}

fn require_profile_index(request: &Extrude, curves: &[SketchEntityId]) -> Result<(), Error> {
    if !curves.is_empty() && request.profile_index == 0 {
        return Ok(());
    }
    Err(Error(format!(
        "extrude sketch {} profile {} with {} profile curves: expected profile 0 with at least one non-construction curve",
        request.sketch_index,
        request.profile_index,
        curves.len()
    )))
}

fn ordered_profile(
    geometry: &Sketch2D,
    curves: &[SketchEntityId],
) -> Result<Vec<SketchDirectedCurve>, Error> {
    if let [curve] = curves
        && geometry.circle(*curve).is_ok()
    {
        return Ok(vec![SketchDirectedCurve {
            curve: *curve,
            reversed: false,
        }]);
    }
    let lines = curves
        .iter()
        .map(|curve| geometry.line(*curve).map_err(api_error))
        .collect::<Result<Vec<_>, _>>()?;
    ordered_line_ring(&lines)
}

fn ordered_line_ring(
    lines: &[&oblikovati_model::sketch::SketchLine2D],
) -> Result<Vec<SketchDirectedCurve>, Error> {
    let first = lines.iter().min_by_key(|line| line.id.0).ok_or_else(|| {
        Error("extrude line profile: expected at least one boundary curve".into())
    })?;
    let start = first.endpoints[0].min(first.endpoints[1]);
    let mut result = vec![directed_line(first, start)?];
    while result.len() < lines.len() {
        let end = directed_end(geometry_line(&result, lines)?, result.last().unwrap())?;
        result.push(next_line(lines, &result, end)?);
    }
    require_closed_line_ring(lines, &result, start)?;
    Ok(result)
}

fn geometry_line<'a>(
    selection: &[SketchDirectedCurve],
    lines: &'a [&oblikovati_model::sketch::SketchLine2D],
) -> Result<&'a oblikovati_model::sketch::SketchLine2D, Error> {
    let current = selection.last().expect("nonempty line traversal");
    lines
        .iter()
        .copied()
        .find(|line| line.id == current.curve)
        .ok_or_else(|| {
            Error(format!(
                "extrude curve {:?}: expected selected line",
                current.curve
            ))
        })
}

fn directed_line(
    line: &oblikovati_model::sketch::SketchLine2D,
    start: SketchEntityId,
) -> Result<SketchDirectedCurve, Error> {
    if line.endpoints.contains(&start) {
        return Ok(SketchDirectedCurve {
            curve: line.id,
            reversed: line.endpoints[1] == start,
        });
    }
    Err(Error(format!(
        "extrude line {:?}: expected endpoint {:?}",
        line.id, start
    )))
}

fn directed_end(
    line: &oblikovati_model::sketch::SketchLine2D,
    directed: &SketchDirectedCurve,
) -> Result<SketchEntityId, Error> {
    let endpoints = line.endpoints;
    if directed.reversed {
        return Ok(endpoints[0]);
    }
    Ok(endpoints[1])
}

fn next_line(
    lines: &[&oblikovati_model::sketch::SketchLine2D],
    selected: &[SketchDirectedCurve],
    start: SketchEntityId,
) -> Result<SketchDirectedCurve, Error> {
    let candidates = lines.iter().copied().filter(|line| {
        !selected.iter().any(|directed| directed.curve == line.id)
            && line.endpoints.contains(&start)
    });
    let mut ordered: Vec<_> = candidates.collect();
    ordered.sort_by_key(|line| line.id.0);
    match ordered.as_slice() {
        [line] => directed_line(line, start),
        _ => Err(Error(format!(
            "extrude line profile at point {:?}: expected exactly one unused connected line, got {:?}",
            start,
            ordered.iter().map(|line| line.id).collect::<Vec<_>>()
        ))),
    }
}

fn require_closed_line_ring(
    lines: &[&oblikovati_model::sketch::SketchLine2D],
    selected: &[SketchDirectedCurve],
    start: SketchEntityId,
) -> Result<(), Error> {
    let end = directed_end(geometry_line(selected, lines)?, selected.last().unwrap())?;
    if end == start {
        return Ok(());
    }
    Err(Error(format!(
        "extrude line profile ends at {:?}: expected closed ring at {:?}",
        end, start
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

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ordered_profile_traverses_an_unordered_shared_line_ring() {
        let mut sketch = Sketch2D::default();
        for (id, point) in [(1, [0., 0.]), (2, [1., 0.]), (3, [1., 1.]), (4, [0., 1.])] {
            sketch.add_point(SketchEntityId(id), point).unwrap();
        }
        for (id, endpoints) in [(14, [4, 1]), (12, [2, 3]), (11, [1, 2]), (13, [3, 4])] {
            sketch
                .add_line(SketchEntityId(id), endpoints.map(SketchEntityId))
                .unwrap();
        }
        let profile = ordered_profile(&sketch, &[14, 12, 11, 13].map(SketchEntityId)).unwrap();
        assert_eq!(
            profile,
            vec![
                SketchDirectedCurve {
                    curve: SketchEntityId(11),
                    reversed: false
                },
                SketchDirectedCurve {
                    curve: SketchEntityId(12),
                    reversed: false
                },
                SketchDirectedCurve {
                    curve: SketchEntityId(13),
                    reversed: false
                },
                SketchDirectedCurve {
                    curve: SketchEntityId(14),
                    reversed: false
                },
            ]
        );
    }
}
