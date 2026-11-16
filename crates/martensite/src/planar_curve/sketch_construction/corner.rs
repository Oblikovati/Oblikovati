// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarCurveFailure, classify_turn};
use martensite_tkmath::predicates::OrientationSign;
use oblikovati_model::sketch::{SketchCornerSeed, SketchCornerSize};

type Point = [f64; 2];
struct CornerFrame {
    origin: Point,
    directions: [Point; 2],
    lengths: [f64; 2],
    turn: OrientationSign,
}
pub(super) fn construct(
    points: [Point; 3],
    size: SketchCornerSize,
    bound: f64,
) -> Result<SketchCornerSeed, PlanarCurveFailure> {
    if !bound.is_finite() || bound < 0.0 {
        return Err(PlanarCurveFailure::input(format!(
            "corner residual bound {bound}: expected a finite nonnegative value"
        )));
    }
    let frame = frame(points)?;
    let (distances, radius) = match size {
        SketchCornerSize::Chamfer(distances) => (distances, None),
        SketchCornerSize::Fillet(radius) => ([fillet_setback(&frame, radius)?; 2], Some(radius)),
    };
    require_setbacks(distances, frame.lengths)?;
    let endpoints =
        std::array::from_fn(|i| add_scaled(frame.origin, frame.directions[i], distances[i]));
    let arc = radius.map(|radius| arc_seed(&frame, endpoints[0], radius));
    let seed = SketchCornerSeed { endpoints, arc };
    verify_seed(&frame, seed, distances, radius, bound)?;
    Ok(seed)
}
fn frame(points: [Point; 3]) -> Result<CornerFrame, PlanarCurveFailure> {
    let turn = classify_turn(points)?;
    if turn == OrientationSign::Zero {
        return Err(PlanarCurveFailure::input(format!(
            "corner points {points:?}: expected noncollinear rays"
        )));
    }
    let (first, first_length) = ray(points[0], points[1])?;
    let (second, second_length) = ray(points[0], points[2])?;
    Ok(CornerFrame {
        origin: points[0],
        directions: [first, second],
        lengths: [first_length, second_length],
        turn,
    })
}
fn ray(origin: Point, end: Point) -> Result<(Point, f64), PlanarCurveFailure> {
    let vector = [end[0] - origin[0], end[1] - origin[1]];
    let length = vector[0].hypot(vector[1]);
    if !length.is_finite() || length == 0.0 {
        return Err(PlanarCurveFailure::input(format!(
            "corner ray {origin:?} to {end:?}: expected finite nonzero length"
        )));
    }
    Ok(([vector[0] / length, vector[1] / length], length))
}
fn fillet_setback(frame: &CornerFrame, radius: f64) -> Result<f64, PlanarCurveFailure> {
    if !radius.is_finite() || radius <= 0.0 {
        return Err(PlanarCurveFailure::input(format!(
            "corner radius {radius}: expected finite positive length"
        )));
    }
    let [a, b] = frame.directions;
    let sine = (a[0] * b[1] - a[1] * b[0]).abs();
    let cosine = a[0] * b[0] + a[1] * b[1];
    let value = radius * (1.0 + cosine) / sine;
    if !value.is_finite() || value <= 0.0 {
        return Err(PlanarCurveFailure::result(format!(
            "corner radius {radius}, directions {:?}: expected a representable positive setback",
            frame.directions
        )));
    }
    Ok(value)
}
fn require_setbacks(distances: [f64; 2], lengths: [f64; 2]) -> Result<(), PlanarCurveFailure> {
    for (distance, length) in distances.into_iter().zip(lengths) {
        if !distance.is_finite() || distance <= 0.0 || distance >= length {
            return Err(PlanarCurveFailure::input(format!(
                "corner setback {distance} on segment {length}: expected a finite positive distance less than its segment length"
            )));
        }
    }
    Ok(())
}
fn add_scaled(point: Point, direction: Point, distance: f64) -> Point {
    [
        point[0] + distance * direction[0],
        point[1] + distance * direction[1],
    ]
}
fn arc_seed(frame: &CornerFrame, endpoint: Point, radius: f64) -> (Point, bool) {
    let direction = frame.directions[0];
    let side = if frame.turn == OrientationSign::Positive {
        1.0
    } else {
        -1.0
    };
    let normal = [-side * direction[1], side * direction[0]];
    (
        add_scaled(endpoint, normal, radius),
        frame.turn == OrientationSign::Negative,
    )
}
fn verify_seed(
    frame: &CornerFrame,
    seed: SketchCornerSeed,
    distances: [f64; 2],
    radius: Option<f64>,
    bound: f64,
) -> Result<(), PlanarCurveFailure> {
    for (i, distance) in distances.into_iter().enumerate() {
        let delta = [
            seed.endpoints[i][0] - frame.origin[0],
            seed.endpoints[i][1] - frame.origin[1],
        ];
        let direction = frame.directions[i];
        let residual = (delta[0] * direction[0] + delta[1] * direction[1] - distance)
            .abs()
            .max((delta[0] * direction[1] - delta[1] * direction[0]).abs());
        require_residual(residual, bound)?;
        if let (Some((center, _)), Some(radius)) = (seed.arc, radius) {
            let radial = [
                seed.endpoints[i][0] - center[0],
                seed.endpoints[i][1] - center[1],
            ];
            let residual = (radial[0].hypot(radial[1]) - radius)
                .abs()
                .max((radial[0] * direction[0] + radial[1] * direction[1]).abs());
            require_residual(residual, bound)?;
        }
    }
    Ok(())
}
fn require_residual(residual: f64, bound: f64) -> Result<(), PlanarCurveFailure> {
    if !residual.is_finite() || residual > bound {
        return Err(PlanarCurveFailure::result(format!(
            "corner construction residual {residual}: expected finite value at most {bound}"
        )));
    }
    Ok(())
}
