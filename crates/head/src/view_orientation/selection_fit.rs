// SPDX-License-Identifier: GPL-2.0-only
//! Fit displayed scene bounds while retaining the analytic selection pivot.
use super::{Vector, dot, frame};
use oblikovati_viewport::ViewportCamera;

/// The standard Fit margin is ten percent of the resulting view height.
const FIT_MARGIN_FACTOR: f64 = 1.1;
/// Keep the eye clear of the visible box independently of the camera near plane.
const FIT_EYE_CLEARANCE: f64 = 0.1;

pub(crate) fn fit_about_selection(
    camera: &ViewportCamera,
    bounds: &[[[f64; 3]; 2]],
    extent: [u32; 2],
    area: [u32; 4],
) -> Result<ViewportCamera, String> {
    let region = region_fractions(extent, area)?;
    let points = corners(bounds)?;
    if points.is_empty() {
        return Err("selection fit bounds []: expected visible content".into());
    }
    fit_camera(camera, &points, region)
}
fn fit_camera(
    camera: &ViewportCamera,
    points: &[Vector],
    region: [f64; 2],
) -> Result<ViewportCamera, String> {
    let (right, up, toward) = frame(camera);
    let distance = fit_distance(camera, points, toward);
    let scale = required_scale(points, camera, [right, up, toward], distance, region)?;
    let farthest = points
        .iter()
        .map(|point| distance - dot(relative(*point, camera.target), toward))
        .fold(0., f64::max);
    Ok(ViewportCamera {
        eye: std::array::from_fn(|axis| camera.target[axis] + toward[axis] * distance),
        scale: scale * FIT_MARGIN_FACTOR,
        far: camera.far.max(farthest * 2.),
        ..*camera
    })
}

fn region_fractions(extent: [u32; 2], area: [u32; 4]) -> Result<[f64; 2], String> {
    let [left, top, right, bottom] = area;
    if extent[0] == 0
        || extent[1] == 0
        || left >= right
        || top >= bottom
        || right > extent[0]
        || bottom > extent[1]
    {
        return Err(format!(
            "selection fit area {area:?}, extent {extent:?}: expected a nonempty document rectangle"
        ));
    }
    Ok([
        f64::from(right - left) / f64::from(extent[1]),
        f64::from(bottom - top) / f64::from(extent[1]),
    ])
}
fn fit_distance(camera: &ViewportCamera, points: &[Vector], toward: Vector) -> f64 {
    let delta = relative(camera.eye, camera.target);
    let distance = dot(delta, delta).sqrt();
    let nearest = points
        .iter()
        .map(|point| dot(relative(*point, camera.target), toward))
        .fold(f64::NEG_INFINITY, f64::max);
    let clearance = (scene_diagonal(points) * FIT_EYE_CLEARANCE).max(camera.near * 2.);
    distance.max(nearest + clearance)
}
fn scene_diagonal(points: &[Vector]) -> f64 {
    let span: Vector = std::array::from_fn(|axis| {
        let low = points
            .iter()
            .map(|point| point[axis])
            .fold(f64::INFINITY, f64::min);
        let high = points
            .iter()
            .map(|point| point[axis])
            .fold(f64::NEG_INFINITY, f64::max);
        high - low
    });
    dot(span, span).sqrt()
}
fn relative(point: Vector, origin: Vector) -> Vector {
    std::array::from_fn(|axis| point[axis] - origin[axis])
}
fn corners(bounds: &[[[f64; 3]; 2]]) -> Result<Vec<Vector>, String> {
    let mut points = Vec::with_capacity(bounds.len() * 8);
    for [low, high] in bounds {
        validate_box(*low, *high)?;
        points.extend((0..8).map(|bits| {
            std::array::from_fn(|axis| {
                if bits & (1 << axis) == 0 {
                    low[axis]
                } else {
                    high[axis]
                }
            })
        }));
    }
    Ok(points)
}
fn validate_box(low: Vector, high: Vector) -> Result<(), String> {
    if (0..3)
        .any(|axis| !low[axis].is_finite() || !high[axis].is_finite() || low[axis] > high[axis])
    {
        return Err(format!(
            "selection fit bounds {low:?}..{high:?}: expected ordered finite bounds"
        ));
    }
    Ok(())
}
fn required_scale(
    points: &[Vector],
    camera: &ViewportCamera,
    axes: [Vector; 3],
    distance: f64,
    region: [f64; 2],
) -> Result<f64, String> {
    let mut radius: f64 = 0.;
    for point in points {
        radius = radius.max(point_radius(*point, camera, axes, distance, region)?);
    }
    let scale = radius * 2.;
    if !scale.is_finite() || scale <= 0. {
        return Err(format!(
            "selection fit scale {scale}: expected a positive finite scale"
        ));
    }
    Ok(scale)
}
fn point_radius(
    point: Vector,
    camera: &ViewportCamera,
    axes: [Vector; 3],
    distance: f64,
    region: [f64; 2],
) -> Result<f64, String> {
    let delta = relative(point, camera.target);
    let depth = distance - dot(delta, axes[2]);
    if !depth.is_finite() || depth <= 0. {
        return Err(format!(
            "selection fit depth {depth}: expected a positive finite depth"
        ));
    }
    let factor = if camera.perspective {
        distance / depth
    } else {
        1.
    };
    Ok(((dot(delta, axes[0]) * factor).abs() / region[0])
        .max((dot(delta, axes[1]) * factor).abs() / region[1]))
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn fits_shifted_bounds_around_pivot_in_both_projections() {
        let bounds = [[[0., 0., 0.], [4., 3., 2.]]];
        for perspective in [false, true] {
            let camera = ViewportCamera {
                eye: [2., 1.5, 22.],
                target: [2., 1.5, 2.],
                perspective,
                ..Default::default()
            };
            let fitted =
                fit_about_selection(&camera, &bounds, [1000, 1000], [200, 100, 1000, 900]).unwrap();
            assert_eq!(fitted.target, camera.target);
            assert!(fitted.scale >= 5.5, "fit scale {}", fitted.scale);
            let required = required_scale(
                &corners(&bounds).unwrap(),
                &fitted,
                [frame(&fitted).0, frame(&fitted).1, frame(&fitted).2],
                20.,
                [0.8, 0.8],
            )
            .unwrap();
            assert!(fitted.scale >= required);
        }
    }
    #[test]
    fn perspective_fit_is_stable_across_near_planes_and_scene_scales() {
        for scene_scale in [1e-4, 1., 1e4] {
            let bounds = [
                [[0., 0., 0.], [4., 3., 2.]].map(|point| point.map(|value| value * scene_scale))
            ];
            let mut observed = Vec::new();
            for near in [1e-9, 1e-6, 1e-3] {
                let camera = ViewportCamera {
                    eye: [2. * scene_scale, 1.5 * scene_scale, 22. * scene_scale],
                    target: [2. * scene_scale, 1.5 * scene_scale, 2. * scene_scale],
                    near: near * scene_scale,
                    far: 100. * scene_scale,
                    scale: 20. * scene_scale,
                    perspective: true,
                    ..Default::default()
                };
                let fitted =
                    fit_about_selection(&camera, &bounds, [1000, 1000], [0, 0, 1000, 1000])
                        .unwrap();
                observed.push(fitted.scale / scene_scale);
                assert!(
                    fitted.scale / scene_scale < 10.,
                    "unexpected zoom {}",
                    fitted.scale / scene_scale
                );
            }
            assert!(
                (observed[0] - observed[2]).abs() < 1e-9,
                "near plane changed fit: {observed:?}"
            );
        }
    }
    #[test]
    fn camera_inside_scene_uses_geometry_clearance_not_the_near_plane() {
        let bounds = [[[0., 0., 0.], [4., 3., 30.]]];
        let mut values = Vec::new();
        for near in [1e-9, 1e-6, 1e-3] {
            let camera = ViewportCamera {
                eye: [2., 1.5, 22.],
                target: [2., 1.5, 2.],
                near,
                perspective: true,
                ..Default::default()
            };
            let fitted =
                fit_about_selection(&camera, &bounds, [1000, 1000], [0, 0, 1000, 1000]).unwrap();
            values.push(fitted.scale);
            assert!(fitted.scale < 100., "unusable fit scale {}", fitted.scale);
            assert!(fitted.eye[2] > 30., "eye must be outside bounds");
        }
        assert!(
            (values[0] - values[2]).abs() < 1e-9,
            "near plane changed fit: {values:?}"
        );
    }
    #[test]
    fn refuses_empty_invalid_and_unrepresentable_bounds() {
        let camera = ViewportCamera::default();
        assert!(fit_about_selection(&camera, &[], [1, 1], [0, 0, 1, 1]).is_err());
        assert!(
            fit_about_selection(
                &camera,
                &[[[1., 0., 0.], [0., 1., 1.]]],
                [1, 1],
                [0, 0, 1, 1]
            )
            .is_err()
        );
        assert!(fit_about_selection(&camera, &[[[0.; 3], [1.; 3]]], [0, 1], [0, 0, 1, 1]).is_err());
    }
}
