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
    (1..=count)
        .map(|index| sampling.parameter(index).map_err(|cause| cause.to_string()))
        .collect()
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
    }
}
