// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) fn resolve(
    snapshot: &DocumentSnapshot,
    request: &AddSketchEntityArgs,
) -> Result<Vec<[f64; 2]>, Error> {
    let expressions = request.point_exprs.as_deref().unwrap_or_default();
    if !expressions.is_empty() {
        return expressions
            .iter()
            .enumerate()
            .map(|(index, pair)| {
                expression_pair(snapshot, pair.as_deref().unwrap_or_default(), index)
            })
            .collect();
    }
    literal_points(snapshot, request.points.as_deref().unwrap_or_default())
}

fn expression_pair(
    snapshot: &DocumentSnapshot,
    pair: &[String],
    index: usize,
) -> Result<[f64; 2], Error> {
    if pair.len() != 2 {
        return Err(Error(format!(
            "sketch.addEntity pointExprs {index} {pair:?}: expected [x-expression, y-expression]"
        )));
    }
    Ok([
        length_expression(snapshot, &pair[0])?,
        length_expression(snapshot, &pair[1])?,
    ])
}

fn literal_points(
    snapshot: &DocumentSnapshot,
    points: &[Option<Vec<f64>>],
) -> Result<Vec<[f64; 2]>, Error> {
    let scale = snapshot.parameters.units().working_scale();
    points
        .iter()
        .enumerate()
        .map(|(index, pair)| {
            crate::sketches::coordinates::working_pair(
                pair.as_deref().unwrap_or_default(),
                scale,
                &format!("sketch.addEntity point {index}"),
            )
        })
        .collect()
}
