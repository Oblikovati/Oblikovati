// SPDX-License-Identifier: GPL-2.0-only
//! Draw the visible regions of a navigation wheel.
use super::{HOVER_FILL, HUB_SHARE, RING_SHARE, tool_direction};
use imgui::DrawListMut;
use std::f32::consts::TAU;

/// Locate the center of a hub region; example: `hub_direction(0, 4)` points northeast.
pub(super) fn hub_direction(place: usize, count: usize) -> [f32; 2] {
    let angle = TAU * (place as f32 + 0.5) / count as f32;
    [angle.sin(), -angle.cos()]
}

/// Draw ring and hub boundaries; example: draw_partitions(&list, center, side, 4, true).
pub(super) fn draw_partitions(
    list: &DrawListMut,
    center: [f32; 2],
    side: f32,
    count: usize,
    hub: bool,
) {
    let color = [0.8, 0.8, 0.8, 0.8];
    let inner = if hub { side * HUB_SHARE } else { 0. };
    if hub {
        list.add_circle(center, inner, color)
            .num_segments(48)
            .build();
        for index in 0..count {
            let direction = tool_direction(index, count);
            let end = [
                center[0] + direction[0] * inner,
                center[1] + direction[1] * inner,
            ];
            list.add_line(center, end, color).build();
        }
    }
    for index in 0..count {
        let direction = hub_direction(index, count);
        let start = [
            center[0] + direction[0] * inner,
            center[1] + direction[1] * inner,
        ];
        let end = [
            center[0] + direction[0] * side * RING_SHARE,
            center[1] + direction[1] * side * RING_SHARE,
        ];
        list.add_line(start, end, color).build();
    }
}

pub(super) fn draw_mini_segments(
    list: &imgui::DrawListMut,
    center: [f32; 2],
    side: f32,
    count: usize,
    under: Option<usize>,
) {
    let radius = side * RING_SHARE;
    if let Some(place) = under {
        let [across, down] = tool_direction(place, count);
        let at = [
            center[0] + across * radius * 0.6,
            center[1] + down * radius * 0.6,
        ];
        list.add_circle(at, radius * 0.3, HOVER_FILL)
            .num_segments(12)
            .filled(true)
            .build();
    }
    for place in 0..count {
        let half_turn = std::f32::consts::TAU / count as f32 / 2.;
        let [across, down] = tool_direction(place, count);
        let (sin, cos) = half_turn.sin_cos();
        let edge = [across * cos - down * sin, across * sin + down * cos];
        list.add_line(
            center,
            [center[0] + edge[0] * radius, center[1] + edge[1] * radius],
            HOVER_FILL,
        )
        .build();
    }
}
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn hub_labels_use_distinct_quadrants() {
        let directions = (0..4)
            .map(|place| hub_direction(place, 4))
            .collect::<Vec<_>>();
        assert!(directions[0][0] > 0. && directions[0][1] < 0.);
        assert!(directions[1][0] > 0. && directions[1][1] > 0.);
        assert!(directions[2][0] < 0. && directions[2][1] > 0.);
        assert!(directions[3][0] < 0. && directions[3][1] < 0.);
    }
}
