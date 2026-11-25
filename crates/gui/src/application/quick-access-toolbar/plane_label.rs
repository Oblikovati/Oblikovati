// SPDX-License-Identifier: GPL-2.0-only
//! Place and pick a readable label in the projected plane's surface.
use super::inside;
use crate::application::document_creation::workspaces::WorkspaceArea;

#[derive(Clone, Copy)]
pub(super) struct PlaneLabel {
    map: QuadProjection,
    center: [f32; 2],
    scale: f32,
    pub(super) opacity: f32,
    footprint: [[f32; 2]; 4],
}

impl PlaneLabel {
    pub(super) fn place(
        mut corners: [[f32; 2]; 4],
        [width, height]: [f32; 2],
        facing: f32,
        area: WorkspaceArea,
    ) -> Option<Self> {
        let opacity = (facing.abs() / 0.15).clamp(0., 1.);
        if opacity < 0.03 {
            return None;
        }
        orient(&mut corners);
        let map = QuadProjection::from_corners(corners)?;
        let lengths = [
            length(corners[0], corners[1]),
            length(corners[0], corners[3]),
        ];
        let scale = (1. / lengths[0].max(lengths[1])).min(0.7 / width.max(height));
        let half = [width * scale / 2., height * scale / 2.];
        let inset = lengths.map(|span| (10. / span).min(0.15));
        let margins = std::array::from_fn(|axis| half[axis] + inset[axis]);
        Self::at_visible_corner(map, margins, half, scale, opacity, area)
    }

    fn at_visible_corner(
        map: QuadProjection,
        [u, v]: [f32; 2],
        [half_u, half_v]: [f32; 2],
        scale: f32,
        opacity: f32,
        area: WorkspaceArea,
    ) -> Option<Self> {
        [[1. - u, 1. - v], [u, 1. - v], [1. - u, v], [u, v]]
            .into_iter()
            .find_map(|center| {
                let footprint = [
                    [-half_u, -half_v],
                    [half_u, -half_v],
                    [half_u, half_v],
                    [-half_u, half_v],
                ]
                .map(|[x, y]| map.point([center[0] + x, center[1] + y]));
                footprint
                    .iter()
                    .all(|&[x, y]| {
                        (area.left..=area.right).contains(&x)
                            && (area.top..=area.bottom).contains(&y)
                    })
                    .then_some(Self {
                        map,
                        center,
                        scale,
                        opacity,
                        footprint,
                    })
            })
    }

    pub(super) fn point(&self, [x, y]: [f32; 2]) -> [f32; 2] {
        self.map.point([
            self.center[0] + x * self.scale,
            self.center[1] + y * self.scale,
        ])
    }

    pub(super) fn contains(&self, pointer: [f32; 2]) -> bool {
        inside(&self.footprint, pointer)
    }
}

fn orient(corners: &mut [[f32; 2]; 4]) {
    let [x, y] = delta(corners[0], corners[1]);
    if x < -0.01 || (x.abs() <= 0.01 && y > 0.) {
        corners.swap(0, 1);
        corners.swap(2, 3);
    }
    if cross(delta(corners[0], corners[1]), delta(corners[0], corners[3])) < 0. {
        corners.swap(0, 3);
        corners.swap(1, 2);
    }
}

#[derive(Clone, Copy)]
struct QuadProjection {
    offset: [f32; 2],
    horizontal: [f32; 2],
    vertical: [f32; 2],
    perspective: [f32; 2],
}

impl QuadProjection {
    fn from_corners([a, b, c, d]: [[f32; 2]; 4]) -> Option<Self> {
        let bc = delta(c, b);
        let dc = delta(c, d);
        let bend = std::array::from_fn(|i| a[i] - b[i] + c[i] - d[i]);
        let det = cross(bc, dc);
        if det.abs() < 0.01 {
            return None;
        }
        let [g, h] = [cross(bend, dc) / det, cross(bc, bend) / det];
        // A projection that crosses the eye cannot have all four corners on the same surface.
        if [1. + g, 1. + h, 1. + g + h].into_iter().any(|w| w < 0.01) {
            return None;
        }
        Some(Self {
            offset: a,
            horizontal: std::array::from_fn(|i| b[i] * (1. + g) - a[i]),
            vertical: std::array::from_fn(|i| d[i] * (1. + h) - a[i]),
            perspective: [g, h],
        })
    }

    fn point(self, [u, v]: [f32; 2]) -> [f32; 2] {
        let w = 1. + self.perspective[0] * u + self.perspective[1] * v;
        std::array::from_fn(|i| {
            (self.offset[i] + self.horizontal[i] * u + self.vertical[i] * v) / w
        })
    }
}

fn delta(from: [f32; 2], to: [f32; 2]) -> [f32; 2] {
    [to[0] - from[0], to[1] - from[1]]
}

fn cross([x, y]: [f32; 2], [u, v]: [f32; 2]) -> f32 {
    x * v - y * u
}

fn length(from: [f32; 2], to: [f32; 2]) -> f32 {
    let [x, y] = delta(from, to);
    x.hypot(y)
}

#[cfg(test)]
mod tests {
    use super::*;
    const VIEW: WorkspaceArea = WorkspaceArea {
        left: 0.,
        top: 0.,
        right: 400.,
        bottom: 400.,
    };

    #[test]
    fn maps_the_unit_square_to_the_perspective_quad() {
        let corners = [[80., 90.], [280., 100.], [260., 220.], [100., 190.]];
        let map = QuadProjection::from_corners(corners).unwrap();
        for (unit, corner) in [[0., 0.], [1., 0.], [1., 1.], [0., 1.]]
            .into_iter()
            .zip(corners)
        {
            let actual = map.point(unit);
            assert!((actual[0] - corner[0]).abs() < 0.001);
            assert!((actual[1] - corner[1]).abs() < 0.001);
        }
    }

    #[test]
    fn chooses_a_visible_corner_and_uses_the_label_footprint_for_picking() {
        let quad = [[60., 80.], [455., 115.], [470., 280.], [75., 245.]];
        let label = PlaneLabel::place(quad, [65., 13.], 0.8, VIEW).unwrap();
        let center = label.point([0., 0.]);
        assert!(center[0] < 120.);
        assert!(label.contains(center));
        assert!(!label.contains([0., 0.]));
    }

    #[test]
    fn grazing_labels_fade_and_back_faces_keep_readable_winding() {
        let reverse = [[250., 160.], [80., 100.], [90., 240.], [260., 300.]];
        let half = PlaneLabel::place(reverse, [65., 13.], -0.075, VIEW).unwrap();
        assert_eq!(half.opacity, 0.5);
        let center = half.point([0., 0.]);
        let right = delta(center, half.point([10., 0.]));
        let down = delta(center, half.point([0., 10.]));
        assert!(right[0] > 0. && cross(right, down) > 0.);
        assert!(PlaneLabel::place(reverse, [65., 13.], 0., VIEW).is_none());
    }
}
