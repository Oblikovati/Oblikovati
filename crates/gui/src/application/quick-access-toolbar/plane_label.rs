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
        let at_center = |center: [f32; 2]| {
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
                    (area.left..=area.right).contains(&x) && (area.top..=area.bottom).contains(&y)
                })
                .then_some(Self {
                    map,
                    center,
                    scale,
                    opacity,
                    footprint,
                })
        };
        [[1. - u, 1. - v], [u, 1. - v], [1. - u, v], [u, v]]
            .into_iter()
            .find_map(at_center)
            .or_else(|| {
                map.visible_center([u, v], [half_u, half_v], area)
                    .and_then(at_center)
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

    fn visible_center(
        self,
        [u, v]: [f32; 2],
        half: [f32; 2],
        area: WorkspaceArea,
    ) -> Option<[f32; 2]> {
        const VIEW_MARGIN: f32 = 8.;
        let limits = [
            CenterBound([1., 0.], -u),
            CenterBound([-1., 0.], 1. - u),
            CenterBound([0., 1.], -v),
            CenterBound([0., -1.], 1. - v),
            self.screen_limit(0, area.left + VIEW_MARGIN, 1., half),
            self.screen_limit(0, area.right - VIEW_MARGIN, -1., half),
            self.screen_limit(1, area.top + VIEW_MARGIN, 1., half),
            self.screen_limit(1, area.bottom - VIEW_MARGIN, -1., half),
        ];
        // A preferred corner of the feasible polygon lies at two boundary lines.
        limits
            .iter()
            .enumerate()
            .flat_map(|(index, first)| {
                limits[index + 1..]
                    .iter()
                    .filter_map(|second| first.crossing(*second))
            })
            .filter(|&center| limits.iter().all(|limit| limit.includes(center)))
            .max_by(|a, b| (a[0] + a[1]).total_cmp(&(b[0] + b[1])))
    }

    fn screen_limit(self, axis: usize, edge: f32, sign: f32, half: [f32; 2]) -> CenterBound {
        // The homogeneous denominator stays positive on the plane. Reserve enough
        // space on each side for the most extreme corner of the label footprint.
        let u = sign * (self.horizontal[axis] - edge * self.perspective[0]);
        let v = sign * (self.vertical[axis] - edge * self.perspective[1]);
        let offset = sign * (self.offset[axis] - edge) - u.abs() * half[0] - v.abs() * half[1];
        CenterBound([u, v], offset)
    }
}

#[derive(Clone, Copy)]
struct CenterBound([f32; 2], f32);

impl CenterBound {
    fn crossing(self, other: Self) -> Option<[f32; 2]> {
        let denominator = cross(self.0, other.0);
        if denominator.abs() < 1e-6 {
            return None;
        }
        let [a, b] = self.0;
        let [c, d] = other.0;
        Some([
            (b * other.1 - d * self.1) / denominator,
            (c * self.1 - a * other.1) / denominator,
        ])
    }

    fn includes(self, center: [f32; 2]) -> bool {
        center[0] * self.0[0] + center[1] * self.0[1] + self.1 >= -0.001
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
    fn a_cropped_perspective_plane_keeps_the_label_inside_the_visible_region() {
        let quad = [[-260., -190.], [670., -72.], [650., 600.], [-180., 700.]];
        let label = PlaneLabel::place(quad, [65., 13.], 0.8, VIEW).unwrap();
        let center = label.point([0., 0.]);
        assert!(center[0] > 300. && center[1] > 300., "{center:?}");
        assert!(label.contains(center));
        assert!(
            label
                .footprint
                .into_iter()
                .flatten()
                .all(|p| (7.9..=392.1).contains(&p))
        );
    }

    #[test]
    fn a_plane_entirely_outside_the_viewport_places_no_label() {
        let quad = [[420., 0.], [560., 30.], [600., 240.], [450., 200.]];
        assert!(PlaneLabel::place(quad, [65., 13.], 0.8, VIEW).is_none());
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
