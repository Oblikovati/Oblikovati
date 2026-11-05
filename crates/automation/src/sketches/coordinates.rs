// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::Error;
pub(super) fn working_pair(vector: &[f64], scale: f64, label: &str) -> Result<[f64; 2], Error> {
    let pair = <[f64; 2]>::try_from(vector).map_err(|_| {
        Error(format!(
            "{label} {vector:?}: expected two finite centimetre components"
        ))
    })?;
    finite_metric_pair(pair, scale, label, "centimetres")
}
pub(super) fn finite_metric_pair(
    pair: [f64; 2],
    scale: f64,
    label: &str,
    units: &str,
) -> Result<[f64; 2], Error> {
    let converted = pair.map(|value| value / scale);
    if pair
        .iter()
        .chain(converted.iter())
        .any(|value| !value.is_finite())
    {
        return Err(Error(format!(
            "{label} {pair:?} scale {scale:?}: expected finite {units} representable in working units"
        )));
    }
    Ok(converted)
}
