// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn append(
    info: &mut WorkPlaneInfo,
    snapshot: &DocumentSnapshot,
    feature: &PlaneEntry,
    frame: Option<SketchPlane>,
) -> Result<(), Error> {
    let corners = feature
        .plane_display_settings()
        .current_corners(frame, snapshot.parameters.units())
        .map_err(api_error)?;
    info.size = corners.map(wire_corners);
    Ok(())
}
pub(in crate::datums) fn wire_corners(corners: [[f64; 3]; 2]) -> Vec<Option<Vec<f64>>> {
    corners
        .into_iter()
        .map(|corner| Some(corner.to_vec()))
        .collect()
}
