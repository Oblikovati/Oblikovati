// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn encode_created(
    result: oblikovati_session::SketchPolylineProjectionInput,
) -> Result<Vec<u8>, Error> {
    let report = result.map_err(api_error)?;
    let lines: Vec<u64> = report.lines.into_iter().map(|id| id.0).collect();
    encode(&AddSketchEntityResult {
        entity_id: lines[0],
        kind: "polyline".into(),
        entity_i_ds: Some(lines),
        point_i_ds: Some(report.points.into_iter().map(|id| id.0).collect()),
        ..Default::default()
    })
}
