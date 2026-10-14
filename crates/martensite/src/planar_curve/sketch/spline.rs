// SPDX-License-Identifier: GPL-2.0-only
use super::{ResolvedSketchCurve, SketchCurveFailure, shape};
use crate::{
    PlanarSplineDefinition,
    planar_curve::{PlanarCurveDefinition, PlanarGeometry, construct},
};
use oblikovati_api::types::{SPLINE_FIT_CHORD, SPLINE_FIT_SMOOTH, SPLINE_FIT_SWEET};
use oblikovati_model::sketch::{
    OwnedSketchGeometry, SketchEntityId, SketchOffsetSpline2D, SketchReadIndex, SketchSpline2D,
    SketchSplineHandle2D, SketchSplineKind,
};
use rustycad_tkgeomalgo::sketch_spline::{
    SketchFitSpacing, SketchHandleOptionsV1, SketchSplineHandle, SketchSplineIntent,
    SketchSplineOptionsV1, SketchSplineReport, build_handled_spline, build_unhandled_spline,
};
use rustycad_tkmath::gp::Point2;
/// Measured checks and policy for one owned parent construction.
/// Example: inspect `curve.spline_construction.as_ref()?.fit_residual` after resolution.
#[derive(Clone, Debug, PartialEq)]
pub struct SketchSplineConstructionReport {
    /// The authored parent, also when the resolved entity is a derived offset.
    pub source: SketchEntityId,
    /// Native interpolation parameters in authored point order.
    pub stations: Vec<f64>,
    /// Measured maximum fit error; absent for a control spline.
    pub fit_residual: Option<f64>,
    /// Measured seam position and derivative errors; absent for an open spline.
    pub seam_residuals: Option<[f64; 3]>,
    /// Minimum accepted collocation pivot in the constructor's own units.
    pub minimum_pivot: f64,
    /// Maximum accepted fit error, separate from display deflection.
    pub maximum_fit_residual: f64,
    /// Maximum accepted closed-seam position and derivative errors.
    pub maximum_seam_residuals: [f64; 3],
}
pub(super) fn resolve(
    index: &SketchReadIndex<'_>,
    spline: &SketchSpline2D,
    limit: f64,
) -> Result<ResolvedSketchCurve, SketchCurveFailure> {
    let entity = spline.id();
    let (definition, report) = native_definition(index, spline)?;
    let mut result = resolved_report(entity, definition, report)?;
    if let Some(ends) = authored_ends(spline) {
        let positions = [shape::point(index, ends[0])?, shape::point(index, ends[1])?];
        result.endpoint_residuals = Some(shape::endpoints::measure(
            &result,
            [0., 1.],
            positions,
            limit,
        )?);
    }
    Ok(result)
}
fn native_definition(
    index: &SketchReadIndex<'_>,
    spline: &SketchSpline2D,
) -> Result<
    (
        rustycad_tkgeomalgo::sketch_spline::SketchSplineDefinition,
        SketchSplineReport,
    ),
    SketchCurveFailure,
> {
    let points = native_points(index, spline)?;
    let handles = native_handles(index, spline)?;
    if handles.is_empty() {
        return natural_definition(spline, &points);
    }
    let entity = spline.id();
    let report = build_handled_spline(
        &points,
        fit_spacing(spline)?,
        spline.closed(),
        &handles,
        SketchHandleOptionsV1::default(),
    )
    .map_err(|cause| SketchCurveFailure::curve(entity, cause.into()))?;
    Ok((report.definition, report.natural))
}
fn natural_definition(
    spline: &SketchSpline2D,
    points: &[Point2],
) -> Result<
    (
        rustycad_tkgeomalgo::sketch_spline::SketchSplineDefinition,
        SketchSplineReport,
    ),
    SketchCurveFailure,
> {
    let entity = spline.id();
    let report = build_unhandled_spline(
        points,
        intent(spline)?,
        spline.closed(),
        SketchSplineOptionsV1::default(),
    )
    .map_err(|cause| SketchCurveFailure::curve(entity, cause.into()))?;
    Ok((report.definition.clone(), report))
}
fn native_handles(
    index: &SketchReadIndex<'_>,
    spline: &SketchSpline2D,
) -> Result<Vec<SketchSplineHandle>, SketchCurveFailure> {
    index
        .spline_handles(spline.id())
        .map_err(|cause| SketchCurveFailure::model(spline.id(), cause))?
        .into_iter()
        .map(|handle| native_handle(index, handle))
        .collect()
}
fn native_handle(
    index: &SketchReadIndex<'_>,
    handle: SketchSplineHandle2D,
) -> Result<SketchSplineHandle, SketchCurveFailure> {
    Ok(SketchSplineHandle {
        fit_index: handle.fit_index(),
        end: shape::point(index, handle.end()).map(|[x, y]| Point2::new(x, y))?,
    })
}
fn native_points(
    index: &SketchReadIndex<'_>,
    spline: &SketchSpline2D,
) -> Result<Vec<Point2>, SketchCurveFailure> {
    spline
        .points()
        .iter()
        .map(|&id| shape::point(index, id).map(|[x, y]| Point2::new(x, y)))
        .collect()
}
fn authored_ends(spline: &SketchSpline2D) -> Option<[SketchEntityId; 2]> {
    if spline.closed() && spline.kind() == SketchSplineKind::Control {
        return None;
    }
    let first = *spline.points().first()?;
    let last = if spline.closed() {
        first
    } else {
        *spline.points().last()?
    };
    Some([first, last])
}
fn resolved_report(
    entity: SketchEntityId,
    definition: rustycad_tkgeomalgo::sketch_spline::SketchSplineDefinition,
    report: SketchSplineReport,
) -> Result<ResolvedSketchCurve, SketchCurveFailure> {
    let definition = owned_definition(&definition)
        .map_err(|cause| SketchCurveFailure::curve(entity, cause.into()))?;
    Ok(ResolvedSketchCurve {
        entity,
        definition: PlanarCurveDefinition {
            geometry: PlanarGeometry::Spline(definition),
            bounds: None,
            left_offset: 0.,
        },
        endpoint_residuals: None,
        spline_construction: Some(construction_report(entity, report)),
    })
}
fn intent(spline: &SketchSpline2D) -> Result<SketchSplineIntent, SketchCurveFailure> {
    let spacing = match spline.kind() {
        SketchSplineKind::Control => return Ok(SketchSplineIntent::Control),
        SketchSplineKind::Fit {
            method: SPLINE_FIT_SMOOTH,
        } => SketchFitSpacing::Centripetal,
        SketchSplineKind::Fit {
            method: SPLINE_FIT_CHORD,
        } => SketchFitSpacing::ChordLength,
        SketchSplineKind::Fit {
            method: SPLINE_FIT_SWEET,
        } => SketchFitSpacing::Uniform,
        unexpected => {
            return Err(SketchCurveFailure::input(
                spline.id(),
                format!("spline kind {unexpected:?}: expected control or a released fit method"),
            ));
        }
    };
    Ok(SketchSplineIntent::Fit(spacing))
}
fn fit_spacing(spline: &SketchSpline2D) -> Result<SketchFitSpacing, SketchCurveFailure> {
    match intent(spline)? {
        SketchSplineIntent::Fit(spacing) => Ok(spacing),
        SketchSplineIntent::Control => Err(SketchCurveFailure::input(
            spline.id(),
            "active spline handles: expected a fit spline".to_owned(),
        )),
    }
}
fn owned_definition(
    definition: &rustycad_tkgeomalgo::sketch_spline::SketchSplineDefinition,
) -> Result<PlanarSplineDefinition, rustycad_tkernel::standard::Failure> {
    let curve = definition.curve()?;
    Ok(PlanarSplineDefinition {
        poles: definition
            .poles()
            .iter()
            .map(|point| [point.x(), point.y()])
            .collect(),
        weights: None,
        knots: curve.knots().as_slice().to_vec(),
        multiplicities: curve.multiplicities().as_slice().to_vec(),
        degree: definition.degree(),
        // The released recipe represents closure with wrapped poles, not a periodic flag.
        periodic: false,
    })
}
fn construction_report(
    source: SketchEntityId,
    report: SketchSplineReport,
) -> SketchSplineConstructionReport {
    SketchSplineConstructionReport {
        source,
        stations: report.stations,
        fit_residual: report.fit_residual,
        seam_residuals: report.seam_residuals,
        minimum_pivot: report.options.minimum_pivot,
        maximum_fit_residual: report.options.maximum_fit_residual,
        maximum_seam_residuals: report.options.maximum_seam_residuals,
    }
}
pub(super) fn offset(
    index: &SketchReadIndex<'_>,
    offset: SketchOffsetSpline2D,
    limit: f64,
) -> Result<ResolvedSketchCurve, SketchCurveFailure> {
    let parent = offset_parent(index, offset)?;
    let mut result = resolve(index, parent, limit)?;
    result.entity = offset.id();
    result.endpoint_residuals = None;
    result.definition.left_offset = offset.distance();
    construct::build_curve(&result.definition)
        .map_err(|cause| SketchCurveFailure::curve(result.entity, cause))?;
    Ok(result)
}

fn offset_parent<'a>(
    index: &SketchReadIndex<'a>,
    offset: SketchOffsetSpline2D,
) -> Result<&'a SketchSpline2D, SketchCurveFailure> {
    let parent = index
        .entity(offset.parent())
        .map_err(|cause| SketchCurveFailure::model(offset.id(), cause))?;
    let OwnedSketchGeometry::Spline(parent) = parent else {
        return Err(SketchCurveFailure::input(
            offset.id(),
            format!(
                "offset parent {:?}: expected an owned spline",
                offset.parent()
            ),
        ));
    };
    Ok(parent)
}
#[cfg(test)]
#[path = "../../../tests/unit/sketch_spline.rs"]
mod tests;
