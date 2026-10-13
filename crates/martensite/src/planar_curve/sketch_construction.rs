// SPDX-License-Identifier: GPL-2.0-only
use super::{
    CircumcircleOptions, PlanarCircumcircle, PlanarCurveFailure,
    circumcircle::{classify_points, measure_residuals, require_options},
    construct_planar_circumcircle,
};
use oblikovati_model::sketch::{SketchCircleConstruction, SolvedCircularSeed};
use std::error::Error;
/// Stateless geometry service injected at the desktop and CLI composition roots.
#[derive(Clone, Copy, Debug, Default)]
pub struct MartensiteSketchConstruction;
impl SketchCircleConstruction for MartensiteSketchConstruction {
    fn circumcircle(
        &self,
        points: [[f64; 2]; 3],
        options: CircumcircleOptions,
    ) -> Result<PlanarCircumcircle, Box<dyn Error>> {
        construct_planar_circumcircle(points, options)
            .map_err(|cause| Box::new(cause) as Box<dyn Error>)
    }
    fn verify(
        &self,
        points: [[f64; 2]; 3],
        solved: SolvedCircularSeed,
        options: CircumcircleOptions,
    ) -> Result<(), Box<dyn Error>> {
        verify_solved(points, solved, options).map_err(|cause| Box::new(cause) as Box<dyn Error>)
    }
}
fn verify_solved(
    points: [[f64; 2]; 3],
    solved: SolvedCircularSeed,
    options: CircumcircleOptions,
) -> Result<(), PlanarCurveFailure> {
    require_options(options)?;
    require_solved_circle(solved)?;
    measure_residuals(points, solved.center, solved.radius, options)?;
    if let Some((endpoints, ccw)) = solved.arc {
        measure_residuals(
            [endpoints[0], points[1], endpoints[1]],
            solved.center,
            solved.radius,
            options,
        )?;
        verify_arc(points, endpoints, ccw, options)?;
    }
    Ok(())
}
fn verify_arc(
    points: [[f64; 2]; 3],
    endpoints: [[f64; 2]; 2],
    ccw: bool,
    options: CircumcircleOptions,
) -> Result<(), PlanarCurveFailure> {
    verify_endpoint_displacement(points, endpoints, options)?;
    let final_ccw = classify_points([endpoints[0], points[1], endpoints[1]])?;
    if final_ccw != ccw {
        return Err(PlanarCurveFailure::result(format!(
            "solved arc winding {ccw}, points {points:?}, endpoints {endpoints:?}: expected winding {final_ccw} through the supplied middle point"
        )));
    }
    Ok(())
}

fn require_solved_circle(solved: SolvedCircularSeed) -> Result<(), PlanarCurveFailure> {
    if !solved.radius.is_finite()
        || solved.radius <= 0.0
        || solved.center.iter().any(|v| !v.is_finite())
    {
        return Err(PlanarCurveFailure::result(format!(
            "solved circular seed {solved:?}: expected a finite center and positive finite radius"
        )));
    }
    Ok(())
}

fn verify_endpoint_displacement(
    points: [[f64; 2]; 3],
    endpoints: [[f64; 2]; 2],
    options: CircumcircleOptions,
) -> Result<(), PlanarCurveFailure> {
    for (actual, expected) in endpoints.into_iter().zip([points[0], points[2]]) {
        let shift = (actual[0] - expected[0]).hypot(actual[1] - expected[1]);
        if !shift.is_finite() || shift > options.maximum_point_residual {
            return Err(PlanarCurveFailure::result(format!(
                "solved arc endpoint {actual:?}, expected {expected:?}: expected finite displacement <= {}",
                options.maximum_point_residual
            )));
        }
    }
    Ok(())
}
