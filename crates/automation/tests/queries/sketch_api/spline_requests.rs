// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::AddSketchEntityArgs;

pub(super) fn fit_request(method: &str, closed: bool) -> AddSketchEntityArgs {
    let points = [[0., 0.], [0.2, 1.], [2., 1.5], [3., -0.5], [1., -2.]];
    AddSketchEntityArgs {
        kind: "spline".into(),
        points: Some(points.iter().map(|point| Some(point.to_vec())).collect()),
        fit_method: method.into(),
        closed,
        ..Default::default()
    }
}

pub(super) fn offset_request(parent: u64, distance: &str) -> AddSketchEntityArgs {
    AddSketchEntityArgs {
        kind: "offsetSpline".into(),
        entity_refs: Some(vec![parent]),
        radius: distance.into(),
        ..Default::default()
    }
}
