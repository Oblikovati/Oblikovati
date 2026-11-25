// SPDX-License-Identifier: GPL-2.0-only
//! Check a rectangle's four expected edges in the viewport capture.
use super::ApplicationWorld;
use oblikovati_viewport::ViewportCapture;

const EDGE_SEARCH: i32 = 2;
const BACKGROUND_OFFSET: i32 = 6;
const MIN_CHANNEL_DIFFERENCE: u8 = 24;

pub(super) fn current_rectangle_visible(world: &mut ApplicationWorld) -> bool {
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let area = world
        .app()
        .window
        .native
        .gui
        .as_ref()
        .expect("expected a native GUI")
        .document_area(capture.extent)
        .expect("expected the current document area");
    rectangle_visible_in_view(
        &capture,
        super::documents::RECTANGLE,
        area,
        &super::diagnostics::window_occlusions(),
    )
    .expect("expected an unobstructed rectangle probe")
}

pub(super) fn rectangle_edges_visible(capture: &ViewportCapture, corners: [[i32; 2]; 2]) -> bool {
    let [[left, top], [right, bottom]] = corners;
    if left >= right || top >= bottom {
        return false;
    }
    [
        ([left, top], [right, top], [0, 1]),
        ([left, bottom], [right, bottom], [0, 1]),
        ([left, top], [left, bottom], [1, 0]),
        ([right, top], [right, bottom], [1, 0]),
    ]
    .into_iter()
    .all(|(start, end, normal)| edge_visible(capture, start, end, normal))
}

/// Check every rectangle edge only when its probe area is clear of UI.
/// Example: `rectangle_visible_in_view(&capture, corners, area, &windows)`.
pub(crate) fn rectangle_visible_in_view(
    capture: &ViewportCapture,
    corners: [[i32; 2]; 2],
    area: [u32; 4],
    occlusions: &[[f32; 4]],
) -> Result<bool, String> {
    let probe = rectangle_probe_bounds(corners)
        .ok_or_else(|| format!("rectangle corners {corners:?}: expected ordered corners"))?;
    if !probe_inside_area(probe, area, capture.extent) {
        return Err(format!(
            "rectangle probe {probe:?}: expected inside document area {area:?} of {:?}",
            capture.extent
        ));
    }
    if let Some(window) = occlusions
        .iter()
        .find(|window| probe_intersects_window(probe, **window))
    {
        return Err(format!(
            "rectangle probe {probe:?}: expected no overlap with window {window:?}"
        ));
    }
    Ok(rectangle_edges_visible(capture, corners))
}

fn rectangle_probe_bounds([[left, top], [right, bottom]]: [[i32; 2]; 2]) -> Option<[i64; 4]> {
    if left >= right || top >= bottom {
        return None;
    }
    Some([
        i64::from(left) - i64::from(EDGE_SEARCH),
        i64::from(top) - i64::from(EDGE_SEARCH),
        i64::from(right) + i64::from(BACKGROUND_OFFSET) + 1,
        i64::from(bottom) + i64::from(BACKGROUND_OFFSET) + 1,
    ])
}

fn probe_inside_area(probe: [i64; 4], area: [u32; 4], extent: [u32; 2]) -> bool {
    area[0] < area[2]
        && area[1] < area[3]
        && area[2] <= extent[0]
        && area[3] <= extent[1]
        && probe[2] <= i64::from(i32::MAX)
        && probe[3] <= i64::from(i32::MAX)
        && probe[0] >= i64::from(area[0])
        && probe[1] >= i64::from(area[1])
        && probe[2] <= i64::from(area[2])
        && probe[3] <= i64::from(area[3])
}

fn probe_intersects_window(probe: [i64; 4], window: [f32; 4]) -> bool {
    if window.iter().any(|value| !value.is_finite())
        || window[0] >= window[2]
        || window[1] >= window[3]
    {
        return true;
    }
    (probe[0] as f64) < f64::from(window[2])
        && (probe[2] as f64) > f64::from(window[0])
        && (probe[1] as f64) < f64::from(window[3])
        && (probe[3] as f64) > f64::from(window[1])
}

/// Check contrast along one projected edge, for example `edge_visible(capture, a, b, [0, -1])`.
pub(super) fn edge_visible(
    capture: &ViewportCapture,
    start: [i32; 2],
    end: [i32; 2],
    normal: [i32; 2],
) -> bool {
    (1..16).all(|fraction| {
        let point = std::array::from_fn(|axis| {
            let first = i64::from(start[axis]);
            let last = i64::from(end[axis]);
            (first + (last - first) * fraction / 16) as i32
        });
        sample_contrast(capture, point, normal)
    })
}

fn sample_contrast(capture: &ViewportCapture, point: [i32; 2], normal: [i32; 2]) -> bool {
    let background = offset_point(point, normal, BACKGROUND_OFFSET);
    let Some(background) = pixel(capture, background) else {
        return false;
    };
    (-EDGE_SEARCH..=EDGE_SEARCH).any(|offset| {
        pixel(capture, offset_point(point, normal, offset)).is_some_and(|sample| {
            (0..3).any(|channel| {
                sample[channel].abs_diff(background[channel]) >= MIN_CHANNEL_DIFFERENCE
            })
        })
    })
}

fn offset_point(point: [i32; 2], normal: [i32; 2], offset: i32) -> [i32; 2] {
    [point[0] + normal[0] * offset, point[1] + normal[1] * offset]
}

fn pixel(capture: &ViewportCapture, [x, y]: [i32; 2]) -> Option<&[u8]> {
    let [width, height] = capture.extent;
    let (x, y) = (u32::try_from(x).ok()?, u32::try_from(y).ok()?);
    if x >= width || y >= height {
        return None;
    }
    let index = ((y as usize * width as usize) + x as usize) * 4;
    capture.rgba.get(index..index + 4)
}

#[cfg(test)]
#[path = "rectangle_visibility_tests.rs"]
mod tests;
