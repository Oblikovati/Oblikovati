// SPDX-License-Identifier: GPL-2.0-only
//! Display-only sampling of exact edges, using Martensite's curve discretizer.
use martensite_tkbrep::{
    brep_adaptor::BRepAdaptorCurve,
    topods::{self, Shape},
};
use martensite_tkg3d::adaptor3d::Adaptor3dCurve;
use martensite_tkgeombase::gcpnts::QuasiUniformDeflection;
use martensite_tkmath::geom_abs;
// Matches the established desktop sketch display point limit.
const MAXIMUM_CURVE_POINTS: u32 = 4096;

pub(super) fn curve_parameters(
    shape: &Shape,
    range: [f64; 2],
    deflection: f64,
) -> Result<Vec<f64>, String> {
    validate_sample_range(range)?;
    if !deflection.is_finite() || deflection <= 0. {
        return Err(format!(
            "display deflection {deflection}: expected positive finite centimetres"
        ));
    }
    let edge = topods::edge(shape).map_err(|cause| cause.to_string())?;
    let curve = BRepAdaptorCurve::with_edge(&edge).map_err(|cause| cause.to_string())?;
    require_capacity(&curve, deflection)?;
    let sampling = QuasiUniformDeflection::with_range(
        &curve as &dyn Adaptor3dCurve,
        deflection,
        range[0],
        range[1],
        geom_abs::Shape::C1,
    )
    .map_err(|cause| format!("drawing curve display sampling: {cause}"))?;
    let count = sampling.nb_points().map_err(|cause| cause.to_string())?;
    if count < 2 || i64::from(count) > i64::from(MAXIMUM_CURVE_POINTS) {
        return Err(format!(
            "drawing curve sample count {count}: expected 2..={MAXIMUM_CURVE_POINTS}"
        ));
    }
    let mut parameters: Vec<f64> = (1..=count)
        .map(|index| sampling.parameter(index).map_err(|cause| cause.to_string()))
        .collect::<Result<_, _>>()?;
    canonicalize_sample_endpoints(&mut parameters, range)?;
    Ok(parameters)
}

fn canonicalize_sample_endpoints(parameters: &mut [f64], range: [f64; 2]) -> Result<(), String> {
    validate_sample_bounds(parameters, range)?;
    reconcile_endpoints(parameters, range)?;
    if parameters
        .windows(2)
        .any(|pair| !pair[0].is_finite() || !pair[1].is_finite() || pair[0] > pair[1])
    {
        return Err(format!(
            "sample parameters {parameters:?}: expected finite nondecreasing values in {range:?}"
        ));
    }
    Ok(())
}
fn validate_sample_bounds(parameters: &[f64], range: [f64; 2]) -> Result<(), String> {
    validate_sample_range(range)?;
    if parameters.len() < 2 {
        return Err(format!(
            "sample range {range:?}, count {}: expected at least two points",
            parameters.len()
        ));
    }
    Ok(())
}
fn validate_sample_range(range: [f64; 2]) -> Result<(), String> {
    if !range[0].is_finite() || !range[1].is_finite() || range[0] > range[1] {
        return Err(format!(
            "sample range {range:?}: expected ordered finite bounds"
        ));
    }
    Ok(())
}
fn reconcile_endpoints(parameters: &mut [f64], range: [f64; 2]) -> Result<(), String> {
    // Martensite #530: the circular sampler accumulates parameter roundoff at the end.
    let scale = range[0].abs().max(range[1].abs()).max(f64::MIN_POSITIVE);
    let bound = (scale * f64::EPSILON).max(f64::from_bits(1)) * parameters.len() as f64;
    let last = parameters.len() - 1;
    for (actual, expected) in [(parameters[0], range[0]), (parameters[last], range[1])] {
        if !actual.is_finite() || (actual - expected).abs() > bound {
            return Err(format!(
                "sample endpoint {actual}: expected {expected} within roundoff {bound}"
            ));
        }
    }
    parameters[0] = range[0];
    parameters[last] = range[1];
    Ok(())
}
fn require_capacity(curve: &BRepAdaptorCurve, deflection: f64) -> Result<(), String> {
    let radius = match curve.get_type().map_err(|cause| cause.to_string())? {
        geom_abs::CurveType::Circle => curve.circle().map_err(|cause| cause.to_string())?.radius(),
        geom_abs::CurveType::Ellipse => curve
            .ellipse()
            .map_err(|cause| cause.to_string())?
            .major_radius(),
        _ => return Ok(()),
    };
    crate::planar_curve::sketch::sampling::require_round_sampling_capacity(
        radius,
        deflection,
        MAXIMUM_CURVE_POINTS,
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use martensite_tkmath::gp::{Axis2, Circle3, Direction3, Point3};
    use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeEdge;

    #[test]
    fn truncated_circle_samples_stay_in_range() {
        let axes = Axis2::new(
            Point3::new(0., 0., 0.),
            Direction3::new(0., 0., 1.).unwrap(),
        )
        .unwrap();
        let edge: Shape = BRepBuilderAPIMakeEdge::from_circ(Circle3::new(axes, 2.).unwrap())
            .unwrap()
            .edge()
            .unwrap()
            .into();
        let range = [std::f64::consts::FRAC_PI_2, std::f64::consts::PI.next_up()];
        let parameters = curve_parameters(&edge, range, 0.1).unwrap();
        assert_eq!(parameters.first(), Some(&range[0]));
        assert_eq!(parameters.last(), Some(&range[1]));
        assert!(
            parameters
                .iter()
                .all(|&parameter| (range[0]..=range[1]).contains(&parameter)),
            "{parameters:?}"
        );
    }

    #[test]
    fn endpoint_normalization_refuses_invalid_sampling() {
        assert!(canonicalize_sample_endpoints(&mut [0., 2., 1.], [0., 1.]).is_err());
        assert!(canonicalize_sample_endpoints(&mut [0., -0.1, 1.], [0., 1.]).is_err());
        assert!(canonicalize_sample_endpoints(&mut [0., 0.5, 1.1], [0., 1.]).is_err());
        assert!(canonicalize_sample_endpoints(&mut [0., f64::NAN, 1.], [0., 1.]).is_err());
        assert!(canonicalize_sample_endpoints(&mut [0., f64::INFINITY, 1.], [0., 1.]).is_err());
        assert!(canonicalize_sample_endpoints(&mut [0., 1.], [1., 0.]).is_err());
    }

    #[test]
    fn endpoint_normalization_preserves_exact_negative_bounds() {
        let range = [-3.0_f64, -1.0_f64];
        let mut parameters = [range[0].next_down(), -2., range[1].next_up()];
        canonicalize_sample_endpoints(&mut parameters, range).unwrap();
        assert_eq!(parameters, [-3., -2., -1.]);
    }

    #[test]
    fn circle_sampling_tracks_requested_deflection() {
        let axes = Axis2::new(
            Point3::new(0., 0., 0.),
            Direction3::new(0., 0., 1.).unwrap(),
        )
        .unwrap();
        let edge: Shape = BRepBuilderAPIMakeEdge::from_circ(Circle3::new(axes, 2.).unwrap())
            .unwrap()
            .edge()
            .unwrap()
            .into();
        let coarse = curve_parameters(&edge, [0., std::f64::consts::TAU], 0.1).unwrap();
        let fine = curve_parameters(&edge, [0., std::f64::consts::TAU], 0.01).unwrap();
        assert!(fine.len() > coarse.len());
        for pair in fine.windows(2) {
            let sag = 2. * (1. - ((pair[1] - pair[0]) / 2.).cos());
            assert!(sag <= 0.01, "circle sag {sag}");
        }
        for invalid in [0., -1., f64::NAN, f64::INFINITY, 1e-100] {
            assert!(curve_parameters(&edge, [0., std::f64::consts::TAU], invalid).is_err());
        }
        for range in [[f64::NAN, 1.], [0., f64::INFINITY], [1., 0.]] {
            assert!(curve_parameters(&edge, range, 0.1).is_err());
        }
    }
}
