// SPDX-License-Identifier: GPL-2.0-only
//! Place projected dimension text in logical screen coordinates without changing its model anchor.
use super::GuiDimensionLabel;
use crate::GuiFrameRequest;
use std::collections::HashMap;

const CELL: f32 = 64.;
const GAP: f32 = 6.;
const WITNESS_CLEARANCE: f32 = 7.;
const MAX_CANDIDATES: usize = 256;
const MAX_INDEX_CELLS: i64 = 64;

type Cell = (i32, i32);

#[derive(Clone, Copy, Debug, PartialEq)]
pub(crate) struct LabelPosition {
    pub center: [f32; 2],
    pub size: [f32; 2],
    pub anchor: [f32; 2],
}

impl LabelPosition {
    pub(crate) fn contains(&self, point: [f32; 2]) -> bool {
        (0..2).all(|axis| (point[axis] - self.center[axis]).abs() <= self.size[axis] / 2. + 2.)
    }
}

#[derive(Clone, Copy, Debug)]
struct Box2 {
    min: [f32; 2],
    max: [f32; 2],
}

impl Box2 {
    fn around(center: [f32; 2], size: [f32; 2]) -> Self {
        Self {
            min: std::array::from_fn(|axis| center[axis] - size[axis] / 2. - GAP / 2.),
            max: std::array::from_fn(|axis| center[axis] + size[axis] / 2. + GAP / 2.),
        }
    }
    fn intersects(self, other: Self) -> bool {
        (0..2).all(|axis| self.min[axis] < other.max[axis] && other.min[axis] < self.max[axis])
    }
    fn cells(self) -> impl Iterator<Item = Cell> {
        let [left, right, top, bottom] = self.cell_bounds();
        (top..=bottom).flat_map(move |row| (left..=right).map(move |column| (column, row)))
    }
    fn fits_grid(self) -> bool {
        let [left, right, top, bottom] = self.cell_bounds();
        let width = i64::from(right) - i64::from(left) + 1;
        let height = i64::from(bottom) - i64::from(top) + 1;
        width
            .checked_mul(height)
            .is_some_and(|count| count <= MAX_INDEX_CELLS)
    }
    fn cell_bounds(self) -> [i32; 4] {
        [
            (self.min[0] / CELL).floor() as i32,
            (self.max[0] / CELL).floor() as i32,
            (self.min[1] / CELL).floor() as i32,
            (self.max[1] / CELL).floor() as i32,
        ]
    }
}

#[derive(Default)]
struct Occupancy {
    boxes: Vec<Box2>,
    cells: HashMap<Cell, Vec<usize>>,
    searches: HashMap<[u32; 4], usize>,
    broad: Vec<usize>,
}
impl Occupancy {
    fn collides(&self, candidate: Box2) -> bool {
        if !candidate.fits_grid() {
            return self.boxes.iter().any(|item| item.intersects(candidate));
        }
        if self
            .broad
            .iter()
            .any(|&index| self.boxes[index].intersects(candidate))
        {
            return true;
        }
        candidate.cells().any(|cell| {
            self.cells.get(&cell).is_some_and(|indices| {
                indices
                    .iter()
                    .any(|&index| self.boxes[index].intersects(candidate))
            })
        })
    }
    fn insert(&mut self, candidate: Box2) {
        let index = self.boxes.len();
        self.boxes.push(candidate);
        if !candidate.fits_grid() {
            self.broad.push(index);
            return;
        }
        for cell in candidate.cells() {
            self.cells.entry(cell).or_default().push(index);
        }
    }
}

pub(crate) fn dimension_label_layout(
    ui: &imgui::Ui,
    request: &GuiFrameRequest,
    viewport: crate::application::document_creation::workspaces::WorkspaceArea,
) -> Vec<Option<LabelPosition>> {
    let _numeric = crate::ui_scale::numeric_font(ui);
    let scale = request.layout_scale();
    let origin = [viewport.left, viewport.top];
    let area = [
        (viewport.right - viewport.left).max(0.),
        (viewport.bottom - viewport.top).max(0.),
    ];
    let mut occupancy = Occupancy::default();
    let labels = &request.commands.dimension_labels;
    let mut positions = vec![None; labels.len()];
    // A transient preview must not move already authored dimensions.
    for editable in [true, false] {
        for (index, label) in labels
            .iter()
            .enumerate()
            .filter(|(_, label)| label.edit.is_some() == editable)
        {
            positions[index] = layout_label(
                ui,
                label,
                request.extent,
                scale,
                origin,
                area,
                &mut occupancy,
            );
        }
    }
    positions
}
fn layout_label(
    ui: &imgui::Ui,
    label: &GuiDimensionLabel,
    extent: [u32; 2],
    scale: [f32; 2],
    origin: [f32; 2],
    area: [f32; 2],
    occupancy: &mut Occupancy,
) -> Option<LabelPosition> {
    if !label
        .pixel
        .iter()
        .zip(extent)
        .all(|(pixel, extent)| *pixel >= 0 && i64::from(*pixel) < i64::from(extent))
    {
        return None;
    }
    let anchor = std::array::from_fn(|axis| label.pixel[axis] as f32 / scale[axis]);
    let size = ui.calc_text_size(label.caption());
    if !label_reaches_viewport(label, anchor, size, scale, origin, area) {
        return None;
    }
    let collision_size = std::array::from_fn(|axis| size[axis].min(area[axis]));
    let local_anchor = std::array::from_fn(|axis| anchor[axis] - origin[axis]);
    let preferred = label.line.map_or(local_anchor, |[from, to]| {
        let dx = (to[0] as f64 - from[0] as f64) as f32 / scale[0];
        let dy = (to[1] as f64 - from[1] as f64) as f32 / scale[1];
        // Prefer >= 6px clearance right of a vertical measure and above a horizontal measure.
        if dy.abs() > dx.abs() {
            [
                local_anchor[0] + size[0] / 2. + WITNESS_CLEARANCE,
                local_anchor[1],
            ]
        } else {
            [
                local_anchor[0],
                local_anchor[1] - size[1] / 2. - WITNESS_CLEARANCE,
            ]
        }
    });
    let local_center = choose_center(preferred, collision_size, area, occupancy);
    occupancy.insert(Box2::around(local_center, collision_size));
    let center = std::array::from_fn(|axis| local_center[axis] + origin[axis]);
    Some(LabelPosition {
        center,
        size,
        anchor,
    })
}
fn label_reaches_viewport(
    label: &GuiDimensionLabel,
    anchor: [f32; 2],
    size: [f32; 2],
    scale: [f32; 2],
    origin: [f32; 2],
    area: [f32; 2],
) -> bool {
    if area.iter().any(|dimension| *dimension <= 0.) {
        return false;
    }
    let local = |point: [f32; 2]| std::array::from_fn(|axis| point[axis] - origin[axis]);
    let local_anchor = local(anchor);
    if (0..2).all(|axis| {
        local_anchor[axis] + size[axis] / 2. > 0.
            && local_anchor[axis] - size[axis] / 2. < area[axis]
    }) {
        return true;
    }
    label.line.is_some_and(|[a, b]| {
        let point = |p: [i32; 2]| local(std::array::from_fn(|axis| p[axis] as f32 / scale[axis]));
        segment_intersects_viewport(point(a), point(b), area)
    })
}
fn segment_intersects_viewport(start: [f32; 2], end: [f32; 2], area: [f32; 2]) -> bool {
    let mut entry = 0_f32;
    let mut exit = 1_f32;
    for axis in 0..2 {
        let direction = end[axis] - start[axis];
        if direction == 0. {
            if !(0. ..=area[axis]).contains(&start[axis]) {
                return false;
            }
            continue;
        }
        let a = -start[axis] / direction;
        let b = (area[axis] - start[axis]) / direction;
        entry = entry.max(a.min(b));
        exit = exit.min(a.max(b));
        if entry > exit {
            return false;
        }
    }
    true
}
fn choose_center(
    anchor: [f32; 2],
    size: [f32; 2],
    area: [f32; 2],
    occupancy: &mut Occupancy,
) -> [f32; 2] {
    let first = clamp_center(anchor, size, area);
    if !occupancy.collides(Box2::around(first, size)) {
        return first;
    }
    let step = [size[0] + GAP, size[1] + GAP];
    let key = [
        first[0].to_bits(),
        first[1].to_bits(),
        size[0].to_bits(),
        size[1].to_bits(),
    ];
    // Occupancy only grows during this frame, so a rejected candidate cannot become free.
    let start = occupancy.searches.get(&key).copied().unwrap_or(0);
    for index in start..MAX_CANDIDATES {
        let (x, y) = spiral_offset(index);
        let center = clamp_center(
            [first[0] + x as f32 * step[0], first[1] + y as f32 * step[1]],
            size,
            area,
        );
        if !occupancy.collides(Box2::around(center, size)) {
            occupancy.searches.insert(key, index + 1);
            return center;
        }
    }
    occupancy.searches.insert(key, MAX_CANDIDATES);
    first
}
fn clamp_center(point: [f32; 2], size: [f32; 2], area: [f32; 2]) -> [f32; 2] {
    std::array::from_fn(|axis| {
        let half = (size[axis] / 2.).min(area[axis] / 2.);
        point[axis].clamp(half, area[axis] - half)
    })
}
fn spiral_offset(index: usize) -> (i32, i32) {
    let mut remaining = index;
    let mut ring = 1_i32;
    loop {
        let side = 2 * ring;
        let count = (side * 4) as usize;
        if remaining < count {
            let position = remaining as i32;
            return match position / side {
                0 => (-ring + position, -ring),
                1 => (ring, -ring + position - side),
                2 => (ring - (position - 2 * side), ring),
                _ => (-ring, ring - (position - 3 * side)),
            };
        }
        remaining -= count;
        ring += 1;
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn hidden_anchors_do_not_create_false_edge_labels() {
        let mut label = GuiDimensionLabel {
            pixel: [90, 150],
            line: None,
            witnesses: None,
            unit: None,
            text: "10".into(),
            role: crate::GuiDimensionLabelRole::Driving,
            edit: None,
        };
        let visible = |label: &GuiDimensionLabel| {
            label_reaches_viewport(
                label,
                [90., 150.],
                [10., 10.],
                [1., 1.],
                [100., 100.],
                [100., 100.],
            )
        };
        assert!(!visible(&label));
        label.line = Some([[80, 150], [99, 150]]);
        assert!(!visible(&label));
        label.line = Some([[80, 150], [110, 150]]);
        assert!(visible(&label));
    }

    #[test]
    fn clipped_dimension_lines_are_visible_but_hidden_lines_are_not() {
        let area = [100., 100.];
        assert!(segment_intersects_viewport([-10., 50.], [10., 50.], area));
        assert!(!segment_intersects_viewport([-10., 50.], [-1., 50.], area));
        assert!(!segment_intersects_viewport(
            [-10., -10.],
            [110., -10.],
            area
        ));
        assert!(segment_intersects_viewport([50., -10.], [50., 110.], area));
    }

    #[test]
    fn separated_labels_keep_their_user_positions() {
        let mut occupied = Occupancy::default();
        let size = [36., 14.];
        let first = [150., 100.];
        occupied.insert(Box2::around(first, size));
        let second = [320., 180.];
        assert_eq!(
            choose_center(second, size, [640., 480.], &mut occupied),
            second
        );
    }

    #[test]
    fn overlapping_labels_find_distinct_visible_positions() {
        let mut occupied = Occupancy::default();
        let size = [50., 14.];
        let first = [320., 240.];
        occupied.insert(Box2::around(first, size));
        let next = choose_center(first, size, [640., 480.], &mut occupied);
        assert_ne!(first, next);
        assert!(!occupied.collides(Box2::around(next, size)));
    }

    #[test]
    fn edge_collision_uses_a_visible_candidate() {
        let mut occupied = Occupancy::default();
        let size = [80., 18.];
        let area = [320., 200.];
        let first = clamp_center([0., 0.], size, area);
        occupied.insert(Box2::around(first, size));
        let next = choose_center([0., 0.], size, area, &mut occupied);
        let bounds = Box2::around(next, size);
        assert!(next[0] >= size[0] / 2. && next[1] >= size[1] / 2.);
        assert!(next[0] <= area[0] - size[0] / 2. && next[1] <= area[1] - size[1] / 2.);
        assert!(!occupied.collides(bounds));
    }

    #[test]
    fn search_is_deterministic_when_the_viewport_is_full() {
        let mut occupied = Occupancy::default();
        let area = [24., 16.];
        let size = area;
        let anchor = [12., 8.];
        occupied.insert(Box2::around(anchor, size));
        assert_eq!(choose_center(anchor, size, area, &mut occupied), anchor);
    }

    #[test]
    fn spiral_visits_distinct_candidates() {
        let offsets: std::collections::BTreeSet<_> =
            (0..MAX_CANDIDATES).map(spiral_offset).collect();
        assert_eq!(offsets.len(), MAX_CANDIDATES);
        assert!(!offsets.contains(&(0, 0)));
    }

    #[test]
    fn repeated_overlaps_reuse_the_search_without_reusing_occupied_space() {
        let mut occupied = Occupancy::default();
        let size = [30., 12.];
        for _ in 0..80 {
            let center = choose_center([320., 240.], size, [640., 480.], &mut occupied);
            let bounds = Box2::around(center, size);
            assert!(!occupied.collides(bounds));
            occupied.insert(bounds);
        }
    }

    #[test]
    fn oversized_labels_are_checked_without_indexing_a_huge_grid() {
        let mut occupied = Occupancy::default();
        let huge = Box2::around([500_000., 500_000.], [1_000_000., 1_000_000.]);
        occupied.insert(huge);
        assert!(occupied.cells.is_empty());
        assert!(occupied.collides(Box2::around([10., 10.], [20., 20.])));
        assert!(!occupied.collides(Box2::around([2_000_000., 2_000_000.], [20., 20.])));
    }
}
