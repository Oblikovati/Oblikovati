// SPDX-License-Identifier: GPL-2.0-only
//! Fit an existing full-frame camera into the document's unobstructed rectangle.

/// A full-frame fit's scale and projection offset for one document area.
#[derive(Clone, Copy, Debug, PartialEq)]
pub(crate) struct FitFraming {
    pub scale: f64,
    pub offset: [i32; 2],
    pub extent: [u32; 2],
}
/// Map a full frame into a document region; for example, `fit_region([640, 480], [160, 80, 640, 480])`.
pub(crate) fn fit_region(extent: [u32; 2], region: [u32; 4]) -> Result<FitFraming, String> {
    validate_region(extent, region)?;
    let [left, top, right, bottom] = region;
    let scale = (f64::from(extent[0]) / f64::from(right - left))
        .max(f64::from(extent[1]) / f64::from(bottom - top));
    let delta = [
        ((i64::from(left) + i64::from(right) - i64::from(extent[0])) / 2) as i32,
        ((i64::from(top) + i64::from(bottom) - i64::from(extent[1])) / 2) as i32,
    ];
    validate_tile_coordinates(extent, region, delta)?;
    Ok(FitFraming {
        scale,
        offset: delta,
        extent,
    })
}
fn validate_region(extent: [u32; 2], region: [u32; 4]) -> Result<(), String> {
    let [left, top, right, bottom] = region;
    if extent[0] == 0
        || extent[1] == 0
        || left >= right
        || top >= bottom
        || right > extent[0]
        || bottom > extent[1]
        || extent.iter().any(|side| *side > i32::MAX as u32)
    {
        return Err(format!(
            "fit region {region:?} for extent {extent:?}: expected a nonempty rectangle inside a positive signed-32-bit extent"
        ));
    }
    Ok(())
}
fn validate_tile_coordinates(
    extent: [u32; 2],
    region: [u32; 4],
    delta: [i32; 2],
) -> Result<(), String> {
    if i64::from(extent[0]) + i64::from(-delta[0]) > i64::from(i32::MAX)
        || i64::from(extent[1]) + i64::from(delta[1]) > i64::from(i32::MAX)
    {
        return Err(format!(
            "fit region {region:?} for extent {extent:?}: expected representable projection tile coordinates"
        ));
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn maps_full_and_offset_regions_and_rejects_invalid_extents() {
        assert_eq!(
            fit_region([640, 480], [0, 0, 640, 480]).unwrap(),
            FitFraming {
                scale: 1.,
                offset: [0, 0],
                extent: [640, 480]
            }
        );
        let frame = fit_region([640, 480], [160, 80, 640, 480]).unwrap();
        assert_eq!(frame.scale, 4. / 3.);
        assert_eq!(frame.offset, [80, 40]);
        for region in [[0, 0, 0, 2], [4, 0, 3, 2], [0, 0, 641, 480]] {
            assert!(fit_region([640, 480], region).is_err());
        }
        assert!(fit_region([0, 480], [0, 0, 1, 1]).is_err());
        assert!(fit_region([i32::MAX as u32, 2], [0, 0, 1, 2]).is_err());
    }
}
