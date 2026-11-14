// SPDX-License-Identifier: GPL-2.0-only
//! Fit visible analytic body bounds while retaining the selected orientation pivot.
use super::{Vector, dot, frame};
use oblikovati_viewport::ViewportCamera;

/// The standard Fit margin is ten percent of the resulting view height.
const FIT_MARGIN_FACTOR: f64 = 1.1;

pub(crate) fn fit_about_selection(
    camera: &ViewportCamera,
    bounds: &[[[f64; 3]; 2]],
    extent: [u32; 2],
    area: [u32; 4],
) -> Result<ViewportCamera, String> {
    let region = region_fractions(extent, area)?;
    let points = corners(bounds)?;
    if points.is_empty() {
        return Err("selection fit bounds []: expected visible bodies".into());
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
    distance.max(nearest + camera.near * 2.)
}
/// Keep every point from the previous fitted frustum visible after centering on the selection.
pub(crate) fn cover_previous_fit(
    candidate: &ViewportCamera,
    fitted: &ViewportCamera,
    extent: [u32; 2],
    area: [u32; 4],
) -> Result<ViewportCamera, String> {
    let region = region_fractions(extent, area)?;
    let axes = frame(candidate);
    let (eye, distance, depth_shift) = covering_eye(candidate, fitted, axes.2);
    let scale = covering_scale(fitted, eye, distance, depth_shift, [axes.0, axes.1], region)?;
    Ok(ViewportCamera {
        eye,
        scale: candidate.scale.max(scale),
        far: candidate.far.max(fitted.far + depth_shift),
        ..*candidate
    })
}
fn covering_eye(
    candidate: &ViewportCamera,
    fitted: &ViewportCamera,
    toward: Vector,
) -> (Vector, f64, f64) {
    let old_delta = relative(fitted.eye, fitted.target);
    let target_delta = relative(candidate.target, fitted.target);
    let candidate_delta = relative(candidate.eye, candidate.target);
    let distance = dot(candidate_delta, candidate_delta)
        .sqrt()
        .max(dot(old_delta, old_delta).sqrt() - dot(target_delta, toward));
    let eye = std::array::from_fn(|axis| candidate.target[axis] + toward[axis] * distance);
    (eye, distance, dot(relative(eye, fitted.eye), toward))
}
fn covering_scale(
    fitted: &ViewportCamera,
    eye: Vector,
    distance: f64,
    depth_shift: f64,
    axes: [Vector; 2],
    region: [f64; 2],
) -> Result<f64, String> {
    let old = relative(fitted.eye, fitted.target);
    let old_distance = dot(old, old).sqrt();
    let minimum_depth = fitted.near + depth_shift;
    if old_distance <= 0. || minimum_depth <= 0. {
        return Err(format!(
            "selection fit distance {old_distance}, depth {minimum_depth}: expected positive camera distances"
        ));
    }
    let eye_shift = relative(eye, fitted.eye);
    let shift =
        (dot(eye_shift, axes[0]).abs() / region[0]).max(dot(eye_shift, axes[1]).abs() / region[1]);
    let scale = projected_cover_scale(fitted, distance, old_distance, minimum_depth, shift);
    if !scale.is_finite() {
        return Err(format!(
            "selection fit scale {scale}: expected a finite camera scale"
        ));
    }
    Ok(scale)
}
fn projected_cover_scale(
    fitted: &ViewportCamera,
    distance: f64,
    old_distance: f64,
    minimum_depth: f64,
    shift: f64,
) -> f64 {
    if fitted.perspective {
        return fitted.scale * distance / old_distance + 2. * shift * distance / minimum_depth;
    }
    fitted.scale + 2. * shift
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
    fn conservative_fit_keeps_prior_visible_frustum_after_pivot_shift() {
        for perspective in [false, true] {
            let old = ViewportCamera {
                perspective,
                near: 2.,
                far: 40.,
                ..Default::default()
            };
            let candidate = ViewportCamera {
                eye: [3., 4., 24.],
                target: [3., 4., 4.],
                perspective,
                ..old
            };
            let fitted =
                cover_previous_fit(&candidate, &old, [800, 600], [0, 0, 800, 600]).unwrap();
            let distance = fitted.eye[2] - fitted.target[2];
            for depth in [old.near, old.far] {
                for x in [-1., 1.] {
                    for y in [-1., 1.] {
                        let old_factor = if perspective { depth / 20. } else { 1. };
                        let point = [
                            x * old.scale * (800. / 600.) * old_factor / 2.,
                            y * old.scale * old_factor / 2.,
                            20. - depth,
                        ];
                        let new_depth = fitted.eye[2] - point[2];
                        let new_factor = if perspective {
                            distance / new_depth
                        } else {
                            1.
                        };
                        assert!(
                            ((point[0] - fitted.target[0]) * new_factor).abs()
                                <= fitted.scale * (800. / 600.) / 2. + 1e-9
                        );
                        assert!(
                            ((point[1] - fitted.target[1]) * new_factor).abs()
                                <= fitted.scale / 2. + 1e-9
                        );
                    }
                }
            }
        }
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
