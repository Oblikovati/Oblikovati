// SPDX-License-Identifier: GPL-2.0-only
use super::{PlanarCurveDefinition, PlanarCurveFailure, construct};
use martensite_tkernel::precision;
use martensite_tkg2d::{adaptor2d::Adaptor2dCurve2d, geom2d_adaptor::Geom2dAdaptorCurve};
use martensite_tkmath::gp::Point2;
mod candidates;
mod domain;

/// The general domain partition distinguishes boundary and native stationary witnesses.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub enum PlanarPickOrigin {
    Boundary,
    Stationary,
}
/// Owned analytic hit witness; no display samples or native handles participate.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PlanarPickPoint {
    pub parameter: f64,
    pub position: [f64; 2],
    pub distance: f64,
    pub position_residual: f64,
    pub stationarity_residual: Option<f64>,
    pub origin: PlanarPickOrigin,
}
/// Version-one native query and independent geometric acceptance limits, in working lengths.
#[derive(Clone, Copy, Debug, PartialEq)]
pub struct PlanarPickOptionsV1 {
    pub native_tolerance: f64,
    pub maximum_position_residual: f64,
    pub maximum_stationarity_residual: f64,
    pub maximum_intervals: usize,
}
impl Default for PlanarPickOptionsV1 {
    fn default() -> Self {
        Self {
            native_tolerance: precision::confusion(),
            maximum_position_residual: precision::confusion(),
            maximum_stationarity_residual: precision::confusion(),
            maximum_intervals: 4096,
        }
    }
}
/// Find the nearest checked native candidate and domain boundary; completion is not root certification.
/// Example: `nearest_planar_curve_candidate(&curve, cursor, PlanarPickOptionsV1::default())?;`.
pub fn nearest_planar_curve_candidate(
    definition: &PlanarCurveDefinition,
    query: [f64; 2],
    options: PlanarPickOptionsV1,
) -> Result<PlanarPickPoint, PlanarCurveFailure> {
    require_pick_options(query, options)?;
    let curve =
        construct::build_curve(definition).map_err(|cause| cause.with_input("pick", definition))?;
    let adaptor = Geom2dAdaptorCurve::with_curve(&curve)?;
    nearest_on_adaptor(&adaptor, query, options).map_err(|cause| {
        cause
            .with_input("pick", definition)
            .with_context(format!("pick query {query:?}"))
    })
}
fn nearest_on_adaptor(
    curve: &Geom2dAdaptorCurve,
    query: [f64; 2],
    options: PlanarPickOptionsV1,
) -> Result<PlanarPickPoint, PlanarCurveFailure> {
    let bounds = domain::continuity_bounds(curve, options.maximum_intervals)?;
    let point = Point2::new(query[0], query[1]);
    let mut witnesses = candidates::boundaries(curve, point, &bounds)?;
    for span in bounds.windows(2) {
        witnesses.extend(candidates::stationary(
            curve,
            point,
            [span[0], span[1]],
            options,
        )?);
    }
    witnesses
        .into_iter()
        .min_by(candidate_order)
        .ok_or_else(|| {
            PlanarCurveFailure::incomplete_operation("planar pick with no finite candidates")
        })
}
fn candidate_order(first: &PlanarPickPoint, second: &PlanarPickPoint) -> std::cmp::Ordering {
    first
        .distance
        .total_cmp(&second.distance)
        .then_with(|| first.parameter.total_cmp(&second.parameter))
        .then_with(|| first.position[0].total_cmp(&second.position[0]))
        .then_with(|| first.position[1].total_cmp(&second.position[1]))
        .then_with(|| first.origin.cmp(&second.origin))
        .then_with(|| first.position_residual.total_cmp(&second.position_residual))
        .then_with(|| {
            optional_residual_order(first.stationarity_residual, second.stationarity_residual)
        })
}
fn optional_residual_order(first: Option<f64>, second: Option<f64>) -> std::cmp::Ordering {
    first
        .is_some()
        .cmp(&second.is_some())
        .then_with(|| first.unwrap_or(0.).total_cmp(&second.unwrap_or(0.)))
}
pub(super) fn require_pick_options(
    query: [f64; 2],
    options: PlanarPickOptionsV1,
) -> Result<(), PlanarCurveFailure> {
    let limits = [
        options.native_tolerance,
        options.maximum_position_residual,
        options.maximum_stationarity_residual,
    ];
    if query.iter().any(|value| !value.is_finite())
        || limits
            .iter()
            .any(|value| !value.is_finite() || *value <= 0.)
        || options.maximum_intervals == 0
        || options.maximum_intervals >= i32::MAX as usize
    {
        return Err(PlanarCurveFailure::input(format!(
            "pick query {query:?}, options {options:?}: expected finite coordinates, positive finite length limits, and interval budget in 1..i32::MAX"
        )));
    }
    Ok(())
}

#[cfg(test)]
mod order_tests {
    use super::*;
    #[test]
    fn identical_roots_use_origin_and_residuals_as_total_tie_breaks() {
        let first = PlanarPickPoint {
            parameter: 0.,
            position: [0.; 2],
            distance: 1.,
            position_residual: 0.,
            stationarity_residual: None,
            origin: PlanarPickOrigin::Boundary,
        };
        let mut second = first;
        second.position_residual = precision::confusion();
        assert!(candidate_order(&first, &second).is_lt());
        second = first;
        second.stationarity_residual = Some(0.);
        assert!(candidate_order(&first, &second).is_lt());
        assert!(optional_residual_order(Some(0.), Some(precision::confusion())).is_lt());
        assert!(candidate_order(&first, &first).is_eq());
    }
}
