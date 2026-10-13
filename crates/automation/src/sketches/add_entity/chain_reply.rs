// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn encode_created(
    result: oblikovati_session::SketchPolylineProjectionInput,
    kind: &'static str,
) -> Result<Vec<u8>, Error> {
    let report = result.map_err(api_error)?;
    let lines: Vec<u64> = report.lines.into_iter().map(|id| id.0).collect();
    encode(&AddSketchEntityResult {
        entity_id: lines[0],
        kind: kind.into(),
        entity_i_ds: Some(lines),
        point_i_ds: Some(report.points.into_iter().map(|id| id.0).collect()),
        ..Default::default()
    })
}
