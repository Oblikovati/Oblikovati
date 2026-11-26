// SPDX-License-Identifier: GPL-2.0-only
//! Project the authored border grid into the sheet canvas.
use imgui::{DrawListMut, Ui};
use oblikovati_api::types::{BorderLabelMode, NO_BORDER_LABEL};
use oblikovati_model::drawing::{DrawingSheetRecipe, border_zone_label};

const DIVISION_COLOR: [f32; 4] = [0.1, 0.1, 0.12, 0.23];
const LABEL_COLOR: [f32; 4] = super::LINE_COLOR;

pub(super) fn draw(
    ui: &Ui,
    list: &DrawListMut<'_>,
    sheet: &DrawingSheetRecipe,
    outer: [[f32; 2]; 2],
    inner: [[f32; 2]; 2],
) {
    for (axis, (count, mode)) in [
        (sheet.border_h_zones, sheet.border_h_label),
        (sheet.border_v_zones, sheet.border_v_label),
    ]
    .into_iter()
    .enumerate()
    {
        let Ok(count) = u64::try_from(count) else {
            continue;
        };
        draw_axis(
            ui,
            list,
            ZoneAxis {
                axis,
                count,
                mode,
                outer,
                inner,
            },
        );
    }
}

struct ZoneAxis {
    axis: usize,
    count: u64,
    mode: BorderLabelMode,
    outer: [[f32; 2]; 2],
    inner: [[f32; 2]; 2],
}

fn draw_axis(ui: &Ui, list: &DrawListMut<'_>, zones: ZoneAxis) {
    let [minimum, maximum] = zones.inner;
    let extent = maximum[zones.axis] - minimum[zones.axis];
    if extent < 1. || zones.count == 0 || (0..2).any(|axis| minimum[axis] >= maximum[axis]) {
        return;
    }
    // Sample from the authored divisions when more cells exist than screen positions.
    let stride = division_stride(zones.count, extent);
    let divisions = std::iter::successors(Some(stride), |index| index.checked_add(stride));
    for index in divisions.take_while(|index| *index < zones.count) {
        let coordinate = zone_coordinate(minimum[zones.axis], extent, index as f64, zones.count);
        let mut endpoints = [minimum, maximum];
        for endpoint in &mut endpoints {
            endpoint[zones.axis] = coordinate;
        }
        list.add_line(endpoints[0], endpoints[1], DIVISION_COLOR)
            .build();
    }
    if zones.mode != NO_BORDER_LABEL && stride == 1 {
        draw_labels(ui, list, &zones, extent);
    }
}

fn division_stride(count: u64, extent: f32) -> u64 {
    count.div_ceil((extent as u64).max(1)).max(1)
}

fn zone_coordinate(start: f32, length: f32, index: f64, count: u64) -> f32 {
    start + (f64::from(length) * index / count as f64) as f32
}

fn draw_labels(ui: &Ui, list: &DrawListMut<'_>, zones: &ZoneAxis, extent: f32) {
    let cross = 1 - zones.axis;
    let minimum = zones.inner[0][zones.axis];
    let cell = extent / zones.count as f32;
    if cell < ui.current_font_size() * 0.5 {
        return;
    }
    for index in 0..zones.count {
        let label = border_zone_label(index, zones.mode);
        let size = ui.calc_text_size(&label);
        if size[zones.axis] + 2. > cell {
            continue;
        }
        let along = zone_coordinate(minimum, extent, index as f64 + 0.5, zones.count);
        label_margins(list, zones, cross, along, size, &label);
    }
}

fn label_margins(
    list: &DrawListMut<'_>,
    zones: &ZoneAxis,
    cross: usize,
    along: f32,
    size: [f32; 2],
    label: &str,
) {
    for side in 0..2 {
        let outside = zones.outer[side][cross];
        let inside = zones.inner[side][cross];
        if (outside - inside).abs() < size[cross] + 2. {
            continue;
        }
        let mut position = [0.; 2];
        position[zones.axis] = along - size[zones.axis] * 0.5;
        position[cross] = (outside + inside - size[cross]) * 0.5;
        list.add_text(position, LABEL_COLOR, label);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn divisions_follow_sheet_coordinates_and_remain_bounded_on_screen() {
        assert_eq!(division_stride(4, 120.), 1);
        assert_eq!(zone_coordinate(30., 120., 1., 4), 60.);
        assert_eq!(zone_coordinate(30., 120., 2.5, 4), 105.);
        assert_eq!(division_stride(4000, 59.), 68);
        assert!(u64::MAX / division_stride(u64::MAX, 48.) <= 48);
    }
}
