// SPDX-License-Identifier: GPL-2.0-only
//! Place a preview wholly beside its tab and inside the visible viewport.

const GAP: f32 = 8.;

#[derive(Clone, Copy, Debug, PartialEq)]
pub(super) struct PreviewPlacement {
    pub position: [f32; 2],
    pub size: [f32; 2],
}

/// Prefer the area above the tab; use the space below when above cannot fit.
/// Example: `preview_placement([800., 600.], [[50., 300.], [150., 325.]], [200., 120.], 40.)`.
pub(super) fn preview_placement(
    viewport: [f32; 2],
    tab: [[f32; 2]; 2],
    desired: [f32; 2],
    minimum_height: f32,
) -> Option<PreviewPlacement> {
    if viewport[0] <= 0. || viewport[1] <= 0. || desired[0] > viewport[0] {
        return None;
    }
    let (y, height) = vertical_placement(viewport[1], tab, desired[1], minimum_height)?;
    Some(PreviewPlacement {
        position: [tab[0][0].clamp(0., viewport[0] - desired[0]), y],
        size: [desired[0], height],
    })
}

fn vertical_placement(
    viewport: f32,
    tab: [[f32; 2]; 2],
    desired: f32,
    minimum: f32,
) -> Option<(f32, f32)> {
    let above = (tab[0][1] - GAP).clamp(0., viewport);
    let below = (viewport - tab[1][1] - GAP).clamp(0., viewport);
    let use_above = above >= desired || (below < desired && above >= below);
    let available = if use_above { above } else { below };
    // Do not hide the tab behind a preview whose labels cannot fit beside it.
    if available < minimum {
        return None;
    }
    let height = desired.min(available);
    let y = if use_above {
        above - height
    } else {
        tab[1][1] + GAP
    };
    Some((y, height))
}

#[cfg(test)]
#[path = "../../../tests/application/document-tabs/preview_placement.rs"]
mod tests;
