// SPDX-License-Identifier: GPL-2.0-only
use super::*;
type DatumEntry = AuthoredFeature<WorkGeometryDefinition>;

pub(super) fn axes(
    snapshot: &DocumentSnapshot,
    request: ListWorkAxesArgs,
) -> Result<Vec<u8>, Error> {
    require_local_occurrence(request.occurrence.as_deref())?;
    let axes = current_rows(
        snapshot,
        DatumCollection::Axes,
        request.include_construction,
        origins::axes(),
        axis_info,
    )?;
    encode(&ListWorkAxesResult { axes: Some(axes) })
}
pub(super) fn points(
    snapshot: &DocumentSnapshot,
    request: ListWorkPointsArgs,
) -> Result<Vec<u8>, Error> {
    require_local_occurrence(request.occurrence.as_deref())?;
    let points = current_rows(
        snapshot,
        DatumCollection::Points,
        request.include_construction,
        vec![origins::point()],
        point_info,
    )?;
    encode(&ListWorkPointsResult {
        points: Some(points),
    })
}
type DatumProjection<T> =
    fn(&DocumentSnapshot, &DatumEntry, i64, &WorkGeometryEvaluation) -> Result<T, Error>;
pub(super) fn current_rows<T>(
    snapshot: &DocumentSnapshot,
    collection: DatumCollection,
    include: bool,
    mut rows: Vec<T>,
    project: DatumProjection<T>,
) -> Result<Vec<T>, Error> {
    let evaluation = evaluate(snapshot)?;
    let prefix = rows.len();
    for (index, feature) in entries(snapshot, collection).enumerate() {
        if feature.is_construction_datum() && !include {
            continue;
        }
        let position = wire_index(index, prefix)?;
        rows.push(project(snapshot, feature, position, &evaluation)?);
    }
    Ok(rows)
}
fn entries(
    snapshot: &DocumentSnapshot,
    collection: DatumCollection,
) -> impl Iterator<Item = &DatumEntry> {
    snapshot
        .work_geometry
        .features()
        .iter()
        .filter(move |feature| collection.contains(*feature.definition()))
}
pub(super) fn evaluate(snapshot: &DocumentSnapshot) -> Result<WorkGeometryEvaluation, Error> {
    snapshot
        .work_geometry
        .evaluate_work_geometry(&snapshot.parameters)
        .map_err(api_error)
}
fn axis_info(
    snapshot: &DocumentSnapshot,
    feature: &DatumEntry,
    index: i64,
    evaluation: &WorkGeometryEvaluation,
) -> Result<WorkAxisInfo, Error> {
    let mut info = axis_identity(snapshot, feature, index)?;
    let geometry = evaluation.axis(feature.id());
    info.healthy = geometry.is_ok();
    info.reason = health::reason(evaluation, feature.id(), geometry.as_ref().err())?;
    info.origin = geometry.as_ref().ok().map(|axis| axis.origin().to_vec());
    info.direction = geometry.ok().map(|axis| axis.direction().to_vec());
    Ok(info)
}
fn axis_identity(
    snapshot: &DocumentSnapshot,
    feature: &DatumEntry,
    index: i64,
) -> Result<WorkAxisInfo, Error> {
    Ok(WorkAxisInfo {
        index,
        ref_: reference::encode_reference(snapshot, feature.id())?,
        name: feature.name().into(),
        kind: constructor_kind(*feature.definition())?,
        construction: feature.is_construction_datum(),
        ..Default::default()
    })
}
fn point_info(
    snapshot: &DocumentSnapshot,
    feature: &DatumEntry,
    index: i64,
    evaluation: &WorkGeometryEvaluation,
) -> Result<WorkPointInfo, Error> {
    let mut info = point_identity(snapshot, feature, index)?;
    let geometry = evaluation.point(feature.id());
    info.healthy = geometry.is_ok();
    info.reason = health::reason(evaluation, feature.id(), geometry.as_ref().err())?;
    info.position = geometry.ok().map(|point| point.to_vec());
    Ok(info)
}
fn point_identity(
    snapshot: &DocumentSnapshot,
    feature: &DatumEntry,
    index: i64,
) -> Result<WorkPointInfo, Error> {
    Ok(WorkPointInfo {
        index,
        ref_: reference::encode_reference(snapshot, feature.id())?,
        name: feature.name().into(),
        kind: constructor_kind(*feature.definition())?,
        visible: feature.is_visible(),
        construction: feature.is_construction_datum(),
        ..Default::default()
    })
}
pub(super) fn constructor_kind(definition: WorkGeometryDefinition) -> Result<String, Error> {
    match definition {
        WorkGeometryDefinition::Axis(_) => Ok(WORK_AXIS_LINE.0.into_owned()),
        WorkGeometryDefinition::AxisThroughPoints(_) => Ok(WORK_AXIS_TWO_POINTS.0.into_owned()),
        WorkGeometryDefinition::Point(_) => Ok(WORK_POINT_POSITION.0.into_owned()),
        WorkGeometryDefinition::PointAtReference(_) => Ok(WORK_POINT_ON_POINT.0.into_owned()),
        _ => Err(Error(format!(
            "datum definition {definition:?}: expected a listed point or axis constructor"
        ))),
    }
}
pub(super) fn creation_reply(
    snapshot: &DocumentSnapshot,
    id: FeatureId,
    collection: DatumCollection,
) -> Result<Vec<u8>, Error> {
    let feature = snapshot.work_geometry.get(id).map_err(api_error)?;
    let position = entries(snapshot, collection)
        .take_while(|feature| feature.id() != id)
        .count();
    let index = wire_index(position, collection.origin_count())?;
    let evaluation = evaluate(snapshot)?;
    match collection {
        DatumCollection::Planes => {
            super::planes::creation_reply(snapshot, feature, index, &evaluation)
        }
        DatumCollection::Axes => axis_reply(axis_info(snapshot, feature, index, &evaluation)?),
        DatumCollection::Points => point_reply(point_info(snapshot, feature, index, &evaluation)?),
    }
}
fn axis_reply(info: WorkAxisInfo) -> Result<Vec<u8>, Error> {
    encode(&CreateWorkAxisResult {
        index: info.index,
        ref_: info.ref_,
        name: info.name,
        healthy: info.healthy,
        reason: info.reason,
    })
}
fn point_reply(info: WorkPointInfo) -> Result<Vec<u8>, Error> {
    encode(&CreateWorkPointResult {
        index: info.index,
        ref_: info.ref_,
        name: info.name,
        healthy: info.healthy,
        reason: info.reason,
    })
}
pub(super) fn wire_index(position: usize, origins: usize) -> Result<i64, Error> {
    let count = position
        .checked_add(origins)
        .and_then(|count| i64::try_from(count).ok());
    count.ok_or_else(|| Error(format!("datum position {position} with {origins} origins: expected a nonnegative i64 collection index")))
}
pub(super) fn require_local_occurrence(occurrence: Option<&[String]>) -> Result<(), Error> {
    if !occurrence.unwrap_or_default().is_empty() {
        return Err(Error(format!(
            "datum occurrence {occurrence:?}: expected the supported local part scope"
        )));
    }
    Ok(())
}
