// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::{
    document_sketch::SketchPlacement,
    feature::{
        sketch_prism::{SketchPrismExtent, ThroughAllDirection},
        work_plane::{WorkPlaneFaceReference, WorkPlaneOrigin, WorkPlaneReference},
    },
};

pub(super) fn reference_extent(
    snapshot: &DocumentSnapshot,
    input: &Extrude,
) -> Result<Option<SketchPrismExtent>, Error> {
    match input.extent.trim().to_ascii_lowercase().as_str() {
        "" | "distance" if input.from_face.is_empty() && input.to_face.is_empty() => Ok(None),
        "" | "distance" => Err(Error(
            "extrude distance extent: expected no fromFace or toFace".into(),
        )),
        "through-all" => through_all(input).map(Some),
        "distance-from-face" if input.from_face.trim().is_empty() => {
            let from = plane_reference(snapshot, "toFace", &input.to_face)?;
            Ok(Some(SketchPrismExtent::DistanceFromPlane { from }))
        }
        "distance-from-face" => Err(Error(
            "extrude distance-from-face: expected the start reference in toFace alone".into(),
        )),
        "to-face" | "from-to" => parse_plane_extent(snapshot, input).map(Some),
        _ => Err(Error(format!(
            "extrude extent {:?}: expected distance, through-all, to-face, from-to, or distance-from-face",
            input.extent
        ))),
    }
}

fn through_all(input: &Extrude) -> Result<SketchPrismExtent, Error> {
    if !input.operation.trim().eq_ignore_ascii_case("cut") {
        return Err(Error("extrude Through All: expected operation Cut".into()));
    }
    if !input.to_face.is_empty() || !input.from_face.is_empty() || !input.second_distance.is_empty()
    {
        return Err(Error(
            "extrude Through All: expected direction without face bounds or Distance B".into(),
        ));
    }
    let direction = match input.direction.trim().to_ascii_lowercase().as_str() {
        "negative" => ThroughAllDirection::Negative,
        "symmetric" => ThroughAllDirection::Symmetric,
        _ => ThroughAllDirection::Positive,
    };
    Ok(SketchPrismExtent::ThroughAll { direction })
}

fn parse_plane_extent(
    snapshot: &DocumentSnapshot,
    input: &Extrude,
) -> Result<SketchPrismExtent, Error> {
    if !input.second_distance.is_empty() {
        return Err(Error("extrude plane extent: expected no Distance B".into()));
    }
    let to = plane_reference(snapshot, "toFace", &input.to_face)?;
    if input.extent.trim().eq_ignore_ascii_case("from-to") {
        let from = plane_reference(snapshot, "fromFace", &input.from_face)?;
        return Ok(SketchPrismExtent::FromTo { from, to });
    }
    if !input.from_face.is_empty() {
        return Err(Error("extrude to-face: expected no fromFace".into()));
    }
    Ok(SketchPrismExtent::ToFace { to })
}

fn plane_reference(
    snapshot: &DocumentSnapshot,
    field: &str,
    value: &str,
) -> Result<WorkPlaneReference, Error> {
    let value = value.trim();
    let error = |reason| Error(format!("extrude {field} {value:?}: {reason}"));
    let reference = match value {
        "origin/plane/xy" => WorkPlaneReference::Origin(WorkPlaneOrigin::XY),
        "origin/plane/xz" => WorkPlaneReference::Origin(WorkPlaneOrigin::XZ),
        "origin/plane/yz" => WorkPlaneReference::Origin(WorkPlaneOrigin::YZ),
        "" => {
            return Err(error(
                "expected a planar face key, plane index, or origin plane".into(),
            ));
        }
        _ => parse_named_plane(snapshot, value).map_err(|cause| error(cause.to_string()))?,
    };
    Ok(reference)
}

fn parse_named_plane(
    snapshot: &DocumentSnapshot,
    value: &str,
) -> Result<WorkPlaneReference, Error> {
    if let Some(index) = value.strip_prefix("plane/") {
        let index = index
            .parse::<i64>()
            .map_err(|_| Error("expected plane/<index>".into()))?;
        let (placement, _) = crate::datums::sketch_plane_at_index(snapshot, index)?;
        return match placement {
            SketchPlacement::OriginPlane(origin) => Ok(WorkPlaneReference::Origin(origin)),
            SketchPlacement::WorkPlane(id) => Ok(WorkPlaneReference::Feature(id)),
            _ => Err(Error("expected an ordered work plane".into())),
        };
    }
    if value.starts_with("datum/v1/") {
        return crate::datums::resolve_work_plane_reference(snapshot, value)
            .map(WorkPlaneReference::Feature);
    }
    let bytes = crate::reference_bytes::hex_bytes(value)?;
    WorkPlaneFaceReference::new(bytes)
        .map(WorkPlaneReference::Face)
        .map_err(Error)
}
