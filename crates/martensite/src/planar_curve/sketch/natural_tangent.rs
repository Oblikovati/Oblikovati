// SPDX-License-Identifier: GPL-2.0-only
use martensite_tkg2d::geom2d::Geom2dCurve;
use martensite_tkgeomalgo::sketch_spline::{
    SketchFitSpacing, SketchSplineIntent, SketchSplineOptionsV1, build_unhandled_spline,
};
use martensite_tkmath::gp::Point2;
use oblikovati_api::types::{SPLINE_FIT_CHORD, SPLINE_FIT_SMOOTH, SPLINE_FIT_SWEET};
use oblikovati_model::sketch::SketchSplineKind;

/// Return the native unhandled fit derivative at an authored point.
/// Example: `natural_fit_tangent(&points, false, 0, kind)?` before adding a handle.
pub fn natural_fit_tangent(
    points: &[[f64; 2]],
    closed: bool,
    fit_index: usize,
    kind: SketchSplineKind,
) -> Result<[f64; 2], String> {
    if fit_index >= points.len() {
        return Err(format!(
            "fit point index {fit_index}: expected an index below {}",
            points.len()
        ));
    }
    let spacing = fit_spacing(kind)
        .ok_or_else(|| format!("spline kind {kind:?}: expected a supported fit spline"))?;
    let native: Vec<_> = points
        .iter()
        .map(|point| Point2::new(point[0], point[1]))
        .collect();
    let report = build_unhandled_spline(
        &native,
        SketchSplineIntent::Fit(spacing),
        closed,
        SketchSplineOptionsV1::default(),
    )
    .map_err(|cause| cause.to_string())?;
    let derivative = report
        .definition
        .curve()
        .map_err(|cause| cause.to_string())?
        .eval_d1(report.stations[fit_index])
        .map_err(|cause| cause.to_string())?
        .d1;
    Ok([derivative.x(), derivative.y()])
}

pub(super) fn fit_spacing(kind: SketchSplineKind) -> Option<SketchFitSpacing> {
    match kind {
        SketchSplineKind::Fit {
            method: SPLINE_FIT_SMOOTH,
        } => Some(SketchFitSpacing::Centripetal),
        SketchSplineKind::Fit {
            method: SPLINE_FIT_CHORD,
        } => Some(SketchFitSpacing::ChordLength),
        SketchSplineKind::Fit {
            method: SPLINE_FIT_SWEET,
        } => Some(SketchFitSpacing::Uniform),
        _ => None,
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn native_fit_tangent_follows_a_straight_line_and_refuses_invalid_inputs() {
        let points = [[0., 0.], [1., 0.], [2., 0.], [3., 0.]];
        let kind = SketchSplineKind::Fit {
            method: SPLINE_FIT_SMOOTH,
        };
        let tangent = natural_fit_tangent(&points, false, 1, kind).unwrap();
        assert!(tangent[0] > 0.);
        assert!(tangent[1].abs() < 1e-12);
        assert!(natural_fit_tangent(&points, false, 4, kind).is_err());
        assert!(natural_fit_tangent(&points, false, 1, SketchSplineKind::Control).is_err());
    }
}
