// SPDX-License-Identifier: GPL-2.0-only
use martensite_tkgeombase::int_ana::IntConicQuad;
use martensite_tkmath::gp::{Direction3, Line3, Plane3, Point3};
use oblikovati_model::sketch::SketchPlane;

/// Intersect an unbounded camera line with an authored sketch frame, without snapping.
/// Example: `sketch_projection(plane, [0.,0.,20.], [0.,0.,-1.], 1e-9)?`.
/// The positive cosine threshold is a UI conditioning limit, not a modelling tolerance.
pub fn sketch_projection(
    plane: SketchPlane,
    origin: [f64; 3],
    direction: [f64; 3],
    min_normal_cosine: f64,
) -> Result<Option<[f64; 2]>, String> {
    validate_projection(origin, direction, min_normal_cosine)?;
    let intersection = intersect_projection(plane, origin, direction, min_normal_cosine)?;
    if intersection
        .is_parallel()
        .map_err(|cause| cause.to_string())?
    {
        return Ok(None);
    }
    let point = intersection.point(1).map_err(|cause| cause.to_string())?;
    plane
        .to_sketch([point.x(), point.y(), point.z()])
        .map(Some)
        .map_err(|cause| cause.to_string())
}
fn validate_projection(origin: [f64; 3], direction: [f64; 3], cosine: f64) -> Result<(), String> {
    if origin.into_iter().chain(direction).any(|v| !v.is_finite())
        || !cosine.is_finite()
        || cosine <= 0.
        || cosine > 1.
    {
        return Err(format!(
            "sketch projection origin {origin:?}, direction {direction:?}, cosine {cosine}: expected finite coordinates and 0 < cosine <= 1"
        ));
    }
    Ok(())
}
fn intersect_projection(
    plane: SketchPlane,
    origin: [f64; 3],
    direction: [f64; 3],
    cosine: f64,
) -> Result<IntConicQuad, String> {
    let normal = projection_direction(plane.normal())?;
    let location = plane.recipe().origin;
    let surface = Plane3::new(Point3::new(location[0], location[1], location[2]), normal)
        .map_err(|cause| cause.to_string())?;
    let line = Line3::new(
        Point3::new(origin[0], origin[1], origin[2]),
        projection_direction(direction)?,
    );
    // Go uses <=; the kernel uses <. The next binary64 value preserves the inclusive UI limit.
    IntConicQuad::line_plane(line, surface, cosine.next_up(), 0., 0.)
        .map_err(|cause| format!("sketch projection line {origin:?}/{direction:?}: {cause}"))
}
fn projection_direction(direction: [f64; 3]) -> Result<Direction3, String> {
    let [x, y, z] = direction;
    let result = Direction3::new(x,y,z)
        .map_err(|cause| format!("sketch projection direction {direction:?}: expected a usable nonzero direction: {cause}"))?;
    if result.coordinates().iter().any(|value| !value.is_finite())
        || *result.coordinates() == [0.; 3]
    {
        return Err(format!(
            "sketch projection direction {direction:?}: expected representable unit direction"
        ));
    }
    Ok(result)
}
