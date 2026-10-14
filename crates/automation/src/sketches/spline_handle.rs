// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::Error;

#[allow(dead_code)]
pub(super) fn tangent_direction(input: &Option<Vec<f64>>) -> Result<Option<[f64; 2]>, Error> {
    let Some(values) = input else {
        return Ok(None);
    };
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

#[allow(dead_code)]
pub(super) fn fit_index(value: i64) -> Result<usize, Error> {
    usize::try_from(value).map_err(|_| {
        Error(format!(
            "spline fit point index {value}: expected a nonnegative index"
        ))
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
            Some(vec![]),
            Some(vec![1.]),
            Some(vec![1., 2., 3.]),
            Some(vec![0., 0.]),
            Some(vec![f64::NAN, 0.]),
        ] {
            assert!(tangent_direction(&input).is_err());
        }
        assert_eq!(tangent_direction(&None).unwrap(), None);
    }
}
