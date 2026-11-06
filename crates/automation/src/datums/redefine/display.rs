// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn prepare(
    current: &WorkPlaneDisplaySettings,
    args: &RedefineWorkPlaneArgs,
) -> Result<WorkPlaneDisplaySettings, Error> {
    let size = parse_size(args.size.as_deref())?;
    current
        .edited(
            args.auto_resize.as_deref().copied(),
            args.grounded.as_deref().copied(),
            size,
        )
        .map_err(api_error)
}
fn parse_size(size: Option<&[Option<Vec<f64>>]>) -> Result<Option<[[f64; 3]; 2]>, Error> {
    let Some(size) = size.filter(|size| !size.is_empty()) else {
        return Ok(None);
    };
    let [first, second] = size else {
        return Err(Error(format!(
            "plane rectangle {size:?}: expected two finite three-component corners in centimetres"
        )));
    };
    Ok(Some([
        input::coordinates(first.as_deref(), "plane rectangle first corner")?,
        input::coordinates(second.as_deref(), "plane rectangle second corner")?,
    ]))
}
