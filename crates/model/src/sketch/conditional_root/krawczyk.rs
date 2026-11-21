// SPDX-License-Identifier: GPL-2.0-only
//! Interval Krawczyk inclusion and infinity-norm contraction test.
use super::Enclosure;

pub(super) struct RootRow {
    pub value: [f64; 2],
    pub jacobian: Vec<[f64; 2]>,
}
impl From<&super::super::SketchIntervalEquation> for RootRow {
    fn from(row: &super::super::SketchIntervalEquation) -> Self {
        Self {
            value: row.value,
            jacobian: row.jacobian.clone(),
        }
    }
}

pub(super) struct RootCheck {
    pub enclosure: Vec<[f64; 2]>,
    pub contraction_upper: f64,
}
pub(super) fn evaluate(
    center: &[f64],
    bounds: &[[f64; 2]],
    box_rows: &[RootRow],
    point_rows: &[RootRow],
    matrix: &[Vec<f64>],
) -> Result<RootCheck, String> {
    let n = center.len();
    if matrix.len() != n
        || matrix
            .iter()
            .any(|row| row.len() != n || row.iter().any(|v| !v.is_finite()))
    {
        return Err(format!(
            "preconditioner {matrix:?}: expected a finite {n} by {n} matrix"
        ));
    }
    let coefficient = coefficients(box_rows, matrix)?;
    let contraction_upper = contraction(&coefficient)?;
    if contraction_upper >= 1.0 {
        return Err(format!(
            "Krawczyk contraction upper {contraction_upper:?}: expected less than 1"
        ));
    }
    let enclosure = image(center, bounds, point_rows, matrix, &coefficient)?;
    for (index, (&value, &outer)) in enclosure.iter().zip(bounds).enumerate() {
        if value[0] <= outer[0] || value[1] >= outer[1] {
            return Err(format!(
                "Krawczyk variable {index} image {value:?}, box {outer:?}: expected strict interior inclusion"
            ));
        }
    }
    Ok(RootCheck {
        enclosure,
        contraction_upper,
    })
}
fn point(value: f64) -> Result<Enclosure, String> {
    Enclosure::new([value, value])
}
fn zero() -> Result<Enclosure, String> {
    point(0.0)
}
fn coefficients(rows: &[RootRow], c: &[Vec<f64>]) -> Result<Vec<Vec<Enclosure>>, String> {
    let n = rows.len();
    (0..n)
        .map(|i| {
            (0..n)
                .map(|j| {
                    let sum = (0..n).try_fold(zero()?, |sum, k| {
                        sum.sum(point(c[i][k])?.product(Enclosure::new(rows[k].jacobian[j])?)?)
                    })?;
                    point(if i == j { 1.0 } else { 0.0 })?.difference(sum)
                })
                .collect()
        })
        .collect()
}
fn contraction(coefficient: &[Vec<Enclosure>]) -> Result<f64, String> {
    let mut upper: f64 = 0.0;
    for row in coefficient {
        let sum = row.iter().try_fold(zero()?, |sum, value| {
            let [lower, upper] = value.bounds();
            sum.sum(point(lower.abs().max(upper.abs()))?)
        })?;
        upper = upper.max(sum.bounds()[1]);
    }
    Ok(upper)
}
fn image(
    center: &[f64],
    bounds: &[[f64; 2]],
    rows: &[RootRow],
    c: &[Vec<f64>],
    coefficient: &[Vec<Enclosure>],
) -> Result<Vec<[f64; 2]>, String> {
    let n = center.len();
    (0..n)
        .map(|i| {
            let correction = (0..n).try_fold(zero()?, |sum, k| {
                sum.sum(point(c[i][k])?.product(Enclosure::new(rows[k].value)?)?)
            })?;
            let base = point(center[i])?.difference(correction)?;
            let result = (0..n).try_fold(base, |sum, j| {
                sum.sum(
                    coefficient[i][j]
                        .product(Enclosure::new(bounds[j])?.difference(point(center[j])?)?)?,
                )
            })?;
            Ok(result.bounds())
        })
        .collect()
}
