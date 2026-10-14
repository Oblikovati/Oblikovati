// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    Error,
    types::{SPLINE_FIT_CHORD, SPLINE_FIT_SMOOTH, SPLINE_FIT_SWEET},
    wire::{SetSplineHandleArgs, SplineHandleInfo},
};
use oblikovati_model::{
    history::{SketchSplineHandleEdit, SketchSplineHandleReport},
    sketch::{Sketch2D, SketchEntityId, SketchSplineKind},
};
use rustycad_tkg2d::geom2d::Geom2dCurve;
use rustycad_tkgeomalgo::sketch_spline::{
    SketchFitSpacing, SketchSplineIntent, SketchSplineOptionsV1, build_unhandled_spline,
};
use rustycad_tkmath::gp::Point2;

pub(super) fn tangent_direction(input: &Option<Vec<f64>>) -> Result<Option<[f64; 2]>, Error> {
    let Some(values) = input else {
        return Ok(None);
    };
    if values.is_empty() {
        return Ok(None);
    }
    let [x, y]: [f64; 2] = values
        .as_slice()
        .try_into()
        .map_err(|_| Error(format!("spline handle tangent {values:?}: expected [x, y]")))?;
    if !x.is_finite() || !y.is_finite() || (x == 0. && y == 0.) {
        return Err(Error(format!(
            "spline handle tangent {values:?}: expected a finite nonzero direction"
        )));
    }
    Ok(Some([x, y]))
}

pub(super) fn fit_index(value: i64) -> Result<usize, Error> {
    usize::try_from(value).map_err(|_| {
        Error(format!(
            "spline fit point index {value}: expected a nonnegative index"
        ))
    })
}

pub(super) fn apply<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    request: SetSplineHandleArgs,
) -> Result<Vec<u8>, Error> {
    let (document, sketch) = active_sketch(files, request.sketch_index)?;
    let request = handle_request(request)?;
    commit(files, document, sketch, request)
}

fn handle_request(request: SetSplineHandleArgs) -> Result<HandleRequest, Error> {
    let parent = SketchEntityId(request.spline);
    let fit_index = fit_index(request.fit_point_index)?;
    let tangent = request
        .active
        .then(|| tangent_direction(&request.tangent))
        .transpose()?
        .flatten();
    Ok(HandleRequest {
        parent,
        fit_index,
        active: request.active,
        tangent,
        weight: request.weight,
    })
}

fn commit<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    document: u64,
    sketch: DocumentSketchId,
    request: HandleRequest,
) -> Result<Vec<u8>, Error> {
    files
        .commands(document)
        .map_err(api_error)?
        .run_projected(
            "Set spline handle",
            |snapshot, ids| edit(snapshot, sketch, request, ids),
            |snapshot, report| reply(snapshot, sketch, report),
        )
        .map_err(api_error)
}

#[derive(Clone, Copy)]
struct HandleRequest {
    parent: SketchEntityId,
    fit_index: usize,
    active: bool,
    tangent: Option<[f64; 2]>,
    weight: f64,
}

fn edit(
    snapshot: &mut DocumentSnapshot,
    sketch: DocumentSketchId,
    request: HandleRequest,
    ids: &mut oblikovati_model::history::RecipeReservations<'_>,
) -> Result<SketchSplineHandleReport, Error> {
    let geometry = snapshot.sketches.get(sketch).map_err(api_error)?.geometry();
    geometry.spline(request.parent).map_err(api_error)?;
    let endpoint = requested_endpoint(geometry, request)?;
    save_edit(snapshot, sketch, request, endpoint, ids)
}

fn requested_endpoint(
    geometry: &Sketch2D,
    request: HandleRequest,
) -> Result<Option<[f64; 2]>, Error> {
    request
        .active
        .then(|| endpoint(geometry, request))
        .transpose()
}

fn save_edit(
    snapshot: &mut DocumentSnapshot,
    sketch: DocumentSketchId,
    request: HandleRequest,
    endpoint: Option<[f64; 2]>,
    ids: &mut oblikovati_model::history::RecipeReservations<'_>,
) -> Result<SketchSplineHandleReport, Error> {
    snapshot
        .edit_spline_handle(
            sketch,
            SketchSplineHandleEdit {
                parent: request.parent,
                fit_index: request.fit_index,
                endpoint,
            },
            ids,
        )
        .map_err(api_error)
}

fn endpoint(geometry: &Sketch2D, request: HandleRequest) -> Result<[f64; 2], Error> {
    let spline = fit_spline(geometry, request.parent, request.fit_index)?;
    let spacing = fit_spacing(spline.kind())?;
    let points = spline_points(geometry, spline.points())?;
    let anchor = points[request.fit_index];
    let span = neighbor_span(&points, spline.closed(), request.fit_index)?;
    let current = current_endpoint(geometry, request.parent, request.fit_index)?;
    let direction = direction(
        request.tangent,
        current,
        &points,
        spline.closed(),
        request.fit_index,
        spacing,
    )?;
    let scale = handle_weight(request.weight, current, anchor, span)?;
    Ok(endpoint_position(anchor, direction, span, scale))
}

fn endpoint_position(anchor: [f64; 2], direction: [f64; 2], span: f64, weight: f64) -> [f64; 2] {
    [
        anchor[0] + direction[0] * span * weight,
        anchor[1] + direction[1] * span * weight,
    ]
}

fn fit_spline(
    geometry: &Sketch2D,
    parent: SketchEntityId,
    fit_index: usize,
) -> Result<&oblikovati_model::sketch::SketchSpline2D, Error> {
    let spline = geometry.spline(parent).map_err(api_error)?;
    if !matches!(spline.kind(), SketchSplineKind::Fit { .. }) || fit_index >= spline.points().len()
    {
        return Err(Error(format!(
            "spline handle parent {parent:?} fit point {fit_index}: expected an in-range fit spline point"
        )));
    }
    Ok(spline)
}

fn spline_points(geometry: &Sketch2D, ids: &[SketchEntityId]) -> Result<Vec<[f64; 2]>, Error> {
    ids.iter()
        .map(|id| Ok(geometry.point(*id).map_err(api_error)?.position))
        .collect()
}

fn current_endpoint(
    geometry: &Sketch2D,
    parent: SketchEntityId,
    fit_index: usize,
) -> Result<Option<[f64; 2]>, Error> {
    let Ok(handle) = geometry.spline_handle(parent, fit_index) else {
        return Ok(None);
    };
    Ok(Some(
        geometry.point(handle.end()).map_err(api_error)?.position,
    ))
}

fn direction(
    requested: Option<[f64; 2]>,
    current: Option<[f64; 2]>,
    points: &[[f64; 2]],
    closed: bool,
    fit_index: usize,
    spacing: SketchFitSpacing,
) -> Result<[f64; 2], Error> {
    if let Some(direction) = requested {
        return unit(direction, "requested spline handle tangent");
    }
    if let Some(endpoint) = current {
        return unit(
            delta(endpoint, points[fit_index]),
            "current spline handle tangent",
        );
    }
    natural_direction(points, closed, fit_index, spacing)
}

fn natural_direction(
    points: &[[f64; 2]],
    closed: bool,
    fit_index: usize,
    spacing: SketchFitSpacing,
) -> Result<[f64; 2], Error> {
    let derivative = natural_derivative(points, closed, fit_index, spacing)?;
    unit(
        [derivative.x(), derivative.y()],
        "natural spline handle tangent",
    )
}

fn natural_derivative(
    points: &[[f64; 2]],
    closed: bool,
    fit_index: usize,
    spacing: SketchFitSpacing,
) -> Result<rustycad_tkmath::gp::Vector2, Error> {
    let point_values = kernel_points(points);
    let report = build_unhandled_spline(
        &point_values,
        SketchSplineIntent::Fit(spacing),
        closed,
        SketchSplineOptionsV1::default(),
    )
    .map_err(api_error)?;
    let evaluation = report
        .definition
        .curve()
        .map_err(api_error)?
        .eval_d1(report.stations[fit_index])
        .map_err(api_error)?;
    Ok(evaluation.d1)
}

fn kernel_points(points: &[[f64; 2]]) -> Vec<Point2> {
    points
        .iter()
        .map(|point| Point2::new(point[0], point[1]))
        .collect()
}

fn fit_spacing(kind: SketchSplineKind) -> Result<SketchFitSpacing, Error> {
    let SketchSplineKind::Fit { method } = kind else {
        return Err(Error(format!("spline kind {kind:?}: expected fit intent")));
    };
    if method == SPLINE_FIT_SMOOTH {
        return Ok(SketchFitSpacing::Centripetal);
    }
    if method == SPLINE_FIT_CHORD {
        return Ok(SketchFitSpacing::ChordLength);
    }
    if method == SPLINE_FIT_SWEET {
        return Ok(SketchFitSpacing::Uniform);
    }
    Err(Error(format!(
        "spline fit method {method:?}: expected smooth, chord, or sweet"
    )))
}

fn neighbor_span(points: &[[f64; 2]], closed: bool, index: usize) -> Result<f64, Error> {
    let count = points.len();
    let previous = if closed {
        (index + count - 1) % count
    } else {
        index.saturating_sub(1)
    };
    let next = if closed {
        (index + 1) % count
    } else {
        (index + 1).min(count - 1)
    };
    let span = length(delta(points[next], points[previous]))? / 3.0;
    if !span.is_finite() {
        return Err(Error(format!(
            "spline handle span {span}: expected a finite length"
        )));
    }
    Ok(span)
}

fn handle_weight(
    requested: f64,
    current: Option<[f64; 2]>,
    anchor: [f64; 2],
    span: f64,
) -> Result<f64, Error> {
    if requested != 0.0 {
        return Ok(requested);
    }
    let Some(endpoint) = current else {
        return Ok(1.0);
    };
    if span == 0.0 {
        return Ok(1.0);
    }
    Ok(length(delta(endpoint, anchor))? / span)
}

fn delta(end: [f64; 2], start: [f64; 2]) -> [f64; 2] {
    [end[0] - start[0], end[1] - start[1]]
}

fn unit(vector: [f64; 2], name: &str) -> Result<[f64; 2], Error> {
    let magnitude = length(vector)?;
    if magnitude == 0.0 {
        return Err(Error(format!(
            "{name} {vector:?}: expected a nonzero direction"
        )));
    }
    Ok([vector[0] / magnitude, vector[1] / magnitude])
}

fn length(vector: [f64; 2]) -> Result<f64, Error> {
    let value = (vector[0] * vector[0] + vector[1] * vector[1]).sqrt();
    if !value.is_finite() {
        return Err(Error(format!(
            "spline handle vector {vector:?}: expected a representable finite length"
        )));
    }
    Ok(value)
}

fn reply(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
    report: SketchSplineHandleReport,
) -> Result<Vec<u8>, Error> {
    encode(&handle_info(snapshot, sketch, report.handle)?)
}

fn handle_info(
    snapshot: &DocumentSnapshot,
    sketch: DocumentSketchId,
    handle: Option<oblikovati_model::sketch::SketchSplineHandle2D>,
) -> Result<SplineHandleInfo, Error> {
    let Some(handle) = handle else {
        return Ok(SplineHandleInfo::default());
    };
    let geometry = snapshot.sketches.get(sketch).map_err(api_error)?.geometry();
    let spline = fit_spline(geometry, handle.parent(), handle.fit_index())?;
    let points = spline_points(geometry, spline.points())?;
    let anchor = points[handle.fit_index()];
    let endpoint = geometry.point(handle.end()).map_err(api_error)?.position;
    let span = neighbor_span(&points, spline.closed(), handle.fit_index())?;
    let vector = delta(endpoint, anchor);
    let magnitude = length(vector)?;
    Ok(SplineHandleInfo {
        handle_id: handle.end().0,
        tangent: (magnitude != 0.0).then(|| vec![vector[0] / magnitude, vector[1] / magnitude]),
        weight: if span == 0.0 { 1.0 } else { magnitude / span },
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn tangent_requires_exactly_two_finite_nonzero_components() {
        assert_eq!(
            tangent_direction(&Some(vec![3., 4.])).unwrap(),
            Some([3., 4.])
        );
        for input in [
            Some(vec![1.]),
            Some(vec![1., 2., 3.]),
            Some(vec![0., 0.]),
            Some(vec![f64::NAN, 0.]),
        ] {
            assert!(tangent_direction(&input).is_err());
        }
        assert_eq!(tangent_direction(&Some(vec![])).unwrap(), None);
        assert_eq!(tangent_direction(&None).unwrap(), None);
    }
}
