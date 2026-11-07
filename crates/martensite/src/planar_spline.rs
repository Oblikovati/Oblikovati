// SPDX-License-Identifier: GPL-2.0-only
use martensite_tkernel::{ncollection::Array1, standard::Handle};
use martensite_tkg2d::geom2d::{Geom2dBSplineCurve, Geom2dCurve, Geom2dOffsetCurve};
use martensite_tkmath::gp::Point2;

pub(crate) mod validation;
/// Immutable inputs for analytic evaluation; no kernel handle crosses this boundary.
#[derive(Clone, Debug, PartialEq)]
pub struct PlanarSplineDefinition {
    pub poles: Vec<[f64; 2]>,
    pub weights: Option<Vec<f64>>,
    pub knots: Vec<f64>,
    pub multiplicities: Vec<i32>,
    pub degree: i32,
    pub periodic: bool,
}
/// Analytic position and derivative with respect to the parent's parameter.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PlanarSplineSample {
    pub position: [f64; 2],
    pub tangent: [f64; 2],
}
/// Evaluate a parent or its signed left offset without sampled geometry.
/// Example: `let sample = evaluate_planar_spline(&definition, 0.5, 0.2)?;`.
pub fn evaluate_planar_spline(
    definition: &PlanarSplineDefinition,
    parameter: f64,
    left_offset: f64,
) -> Result<PlanarSplineSample, String> {
    validation::require_definition(definition)?;
    validation::require_scalars(parameter, left_offset)?;
    let parent = build_parent(definition).map_err(|cause| {
        format!("spline {definition:?}: expected a valid analytic B-spline: {cause}")
    })?;
    validation::require_parameter(
        parent
            .get()
            .ok_or("spline parent handle: expected a constructed analytic curve")?,
        parameter,
    )?;
    evaluate_owned_curve(&parent, parameter, left_offset)
}
pub(crate) fn build_parent(
    definition: &PlanarSplineDefinition,
) -> Result<Handle<dyn Geom2dCurve>, martensite_tkernel::standard::Failure> {
    let poles: Vec<Point2> = definition
        .poles
        .iter()
        .map(|point| Point2::new(point[0], point[1]))
        .collect();
    let poles = Array1::from_shared(&poles, 1);
    let knots = Array1::from_shared(&definition.knots, 1);
    let mults = Array1::from_shared(&definition.multiplicities, 1);
    let curve = match &definition.weights {
        None => build_unweighted(definition, &poles, &knots, &mults),
        Some(weights) => build_weighted(definition, &poles, &knots, &mults, weights),
    };
    curve.map(|curve| Handle::new(curve).cast())
}
fn evaluate_owned_curve(
    parent: &Handle<dyn Geom2dCurve>,
    parameter: f64,
    left_offset: f64,
) -> Result<PlanarSplineSample, String> {
    let basis = sample_curve(
        parent
            .get()
            .ok_or("spline parent handle: expected a constructed analytic curve")?,
        parameter,
    )?;
    if left_offset == 0.0 {
        return Ok(basis);
    }
    validation::require_regular_sample(basis, parameter, left_offset)?;
    let offset = build_left_offset(parent, left_offset).map_err(|cause| format!("spline offset {left_offset} at {parameter}: expected an offset-compatible parent: {cause}"))?;
    let sample = sample_curve(&offset, parameter)?;
    validation::require_regular_sample(sample, parameter, left_offset)?;
    Ok(sample)
}
pub(crate) fn build_left_offset(
    parent: &Handle<dyn Geom2dCurve>,
    left_offset: f64,
) -> Result<Geom2dOffsetCurve, martensite_tkernel::standard::Failure> {
    // #3789 / ADR-0016: OCCT's right normal is the inverse of the authored left side.
    Geom2dOffsetCurve::new(parent, -left_offset, false)
}
fn sample_curve(curve: &dyn Geom2dCurve, parameter: f64) -> Result<PlanarSplineSample, String> {
    let result = curve.eval_d1(parameter).map_err(|cause| {
        format!("spline parameter {parameter}: expected an analytic position and tangent: {cause}")
    })?;
    let sample = PlanarSplineSample {
        position: [result.point.x(), result.point.y()],
        tangent: [result.d1.x(), result.d1.y()],
    };
    validation::require_finite_sample(sample, parameter)?;
    Ok(sample)
}

fn build_weighted(
    definition: &PlanarSplineDefinition,
    poles: &Array1<'_, Point2>,
    knots: &Array1<'_, f64>,
    mults: &Array1<'_, i32>,
    weights: &[f64],
) -> Result<Geom2dBSplineCurve, martensite_tkernel::standard::Failure> {
    Geom2dBSplineCurve::with_weights(
        poles,
        &Array1::from_shared(weights, 1),
        knots,
        mults,
        definition.degree,
        definition.periodic,
    )
}

fn build_unweighted(
    definition: &PlanarSplineDefinition,
    poles: &Array1<'_, Point2>,
    knots: &Array1<'_, f64>,
    mults: &Array1<'_, i32>,
) -> Result<Geom2dBSplineCurve, martensite_tkernel::standard::Failure> {
    Geom2dBSplineCurve::new(poles, knots, mults, definition.degree, definition.periodic)
}
