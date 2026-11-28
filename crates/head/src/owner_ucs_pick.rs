// SPDX-License-Identifier: GPL-2.0-only
//! Ray-picking, triad geometry, and direction math for interactive UCS placement (#3918).
use oblikovati_gui::{GuiUcsStage, GuiUcsTriad};
use oblikovati_model::feature::FeatureId;

pub(super) const DEFAULT_TRIAD_REACH: f64 = 1.5;

pub(super) fn build_triad(
    feature: Option<FeatureId>,
    name: String,
    origin: [f64; 3],
    x_dir: [f64; 3],
    y_dir: [f64; 3],
    reach: f64,
    stage: Option<GuiUcsStage>,
) -> GuiUcsTriad {
    let z_dir = normalize_or(cross(x_dir, y_dir), [0.0, 0.0, 1.0]);
    GuiUcsTriad {
        feature,
        name,
        origin_world: origin,
        x_tip_world: add_scaled(origin, x_dir, reach),
        y_tip_world: add_scaled(origin, y_dir, reach),
        z_tip_world: add_scaled(origin, z_dir, reach),
        origin_pixel: None,
        x_pixel: None,
        y_pixel: None,
        z_pixel: None,
        stage,
    }
}

pub(super) fn compute_triad_reach(vertices: &[(Vec<u8>, [f64; 3])]) -> f64 {
    let Some(&(_, first)) = vertices.first() else {
        return DEFAULT_TRIAD_REACH;
    };
    let (mut lo, mut hi) = (first, first);
    for &(_, pt) in vertices {
        for axis in 0..3 {
            lo[axis] = lo[axis].min(pt[axis]);
            hi[axis] = hi[axis].max(pt[axis]);
        }
    }
    let diag = ((hi[0] - lo[0]).powi(2) + (hi[1] - lo[1]).powi(2) + (hi[2] - lo[2]).powi(2)).sqrt();
    if diag.is_finite() && diag > 1e-4 {
        (diag * 0.28).clamp(0.6, 5.0)
    } else {
        DEFAULT_TRIAD_REACH
    }
}

pub(super) fn nearest_vertex(
    vertices: &[(Vec<u8>, [f64; 3])],
    ray: [[f64; 3]; 2],
    max_distance: f64,
) -> Option<(Vec<u8>, [f64; 3])> {
    vertices
        .iter()
        .filter_map(|(key, point)| {
            let dist = ray_point_distance(*point, ray);
            (dist <= max_distance).then_some((dist, key, *point))
        })
        .min_by(|a, b| a.0.total_cmp(&b.0))
        .map(|(_, key, point)| (key.clone(), point))
}

pub(super) fn resolve_direction_from_ray(
    origin: [f64; 3],
    ray: [[f64; 3]; 2],
    vertices: &[(Vec<u8>, [f64; 3])],
    reach: f64,
    orthogonal_to: Option<[f64; 3]>,
) -> Option<[f64; 3]> {
    let target = nearest_vertex(vertices, ray, reach * 0.35)
        .filter(|(_, pt)| dist3(*pt, origin) > 1e-4)
        .map(|(_, pt)| pt)
        .unwrap_or_else(|| closest_point_on_ray(origin, ray));
    let mut dir = sub3(target, origin);
    if let Some(axis) = orthogonal_to {
        let proj = dot3(dir, axis);
        dir = [
            dir[0] - proj * axis[0],
            dir[1] - proj * axis[1],
            dir[2] - proj * axis[2],
        ];
    }
    normalize_checked(dir)
}

pub(super) fn orthogonal_partner(x: [f64; 3], preferred: [f64; 3]) -> [f64; 3] {
    for candidate in [preferred, [0.0, 1.0, 0.0], [0.0, 0.0, 1.0], [1.0, 0.0, 0.0]] {
        let proj = dot3(candidate, x);
        let ortho = [
            candidate[0] - proj * x[0],
            candidate[1] - proj * x[1],
            candidate[2] - proj * x[2],
        ];
        if let Some(unit) = normalize_checked(ortho) {
            return unit;
        }
    }
    [0.0, 1.0, 0.0]
}

pub(super) fn add_scaled(origin: [f64; 3], dir: [f64; 3], scale: f64) -> [f64; 3] {
    [
        origin[0] + dir[0] * scale,
        origin[1] + dir[1] * scale,
        origin[2] + dir[2] * scale,
    ]
}

pub(super) fn ray_point_distance(point: [f64; 3], ray: [[f64; 3]; 2]) -> f64 {
    dist3(point, closest_point_on_ray(point, ray))
}

pub(super) fn ray_segment_distance(start: [f64; 3], end: [f64; 3], ray: [[f64; 3]; 2]) -> f64 {
    let samples = 8;
    let mut best = f64::INFINITY;
    for i in 0..=samples {
        let frac = f64::from(i) / f64::from(samples);
        let p = [
            start[0] + (end[0] - start[0]) * frac,
            start[1] + (end[1] - start[1]) * frac,
            start[2] + (end[2] - start[2]) * frac,
        ];
        best = best.min(ray_point_distance(p, ray));
    }
    best
}

fn sub3(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [a[0] - b[0], a[1] - b[1], a[2] - b[2]]
}

fn dot3(a: [f64; 3], b: [f64; 3]) -> f64 {
    a[0] * b[0] + a[1] * b[1] + a[2] * b[2]
}

fn cross(a: [f64; 3], b: [f64; 3]) -> [f64; 3] {
    [
        a[1] * b[2] - a[2] * b[1],
        a[2] * b[0] - a[0] * b[2],
        a[0] * b[1] - a[1] * b[0],
    ]
}

fn dist3(a: [f64; 3], b: [f64; 3]) -> f64 {
    let d = sub3(a, b);
    dot3(d, d).sqrt()
}

fn normalize_checked(v: [f64; 3]) -> Option<[f64; 3]> {
    let len = dot3(v, v).sqrt();
    (len.is_finite() && len > 1e-9).then_some([v[0] / len, v[1] / len, v[2] / len])
}

fn normalize_or(v: [f64; 3], fallback: [f64; 3]) -> [f64; 3] {
    normalize_checked(v).unwrap_or(fallback)
}

fn closest_point_on_ray(point: [f64; 3], [origin, dir]: [[f64; 3]; 2]) -> [f64; 3] {
    let Some(u) = normalize_checked(dir) else {
        return origin;
    };
    let t = dot3(sub3(point, origin), u);
    add_scaled(origin, u, t)
}
