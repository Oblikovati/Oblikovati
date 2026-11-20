// SPDX-License-Identifier: GPL-2.0-only
//! Convert singleton derivative and anchor enclosures into affine rows.
use super::{SketchAffineError, SketchRelationId};
use crate::sketch::{SketchIntervalEquation, interval_equations::enclosure::Enclosure};

type Rows = (Vec<Vec<f64>>, Vec<f64>, Vec<(SketchRelationId, usize)>);

pub(super) fn derive(
    domain: &[SketchIntervalEquation],
    point: &[SketchIntervalEquation],
    anchor: &[f64],
) -> Result<Rows, SketchAffineError> {
    if domain.len() != point.len() {
        return Err(SketchAffineError::Input(format!(
            "affine rows: {} domain and {} anchor rows; expected equal lengths",
            domain.len(),
            point.len()
        )));
    }
    let mut coefficients = Vec::with_capacity(domain.len());
    let mut rhs = Vec::with_capacity(domain.len());
    let mut relations = Vec::with_capacity(domain.len());
    for (domain, point) in domain.iter().zip(point) {
        let row = coefficients_for(domain, point, anchor.len())?;
        rhs.push(intercept(&row, point, anchor)?);
        coefficients.push(row);
        relations.push((domain.relation, domain.equation));
    }
    Ok((coefficients, rhs, relations))
}

fn coefficients_for(
    domain: &SketchIntervalEquation,
    point: &SketchIntervalEquation,
    columns: usize,
) -> Result<Vec<f64>, SketchAffineError> {
    if domain.relation != point.relation
        || domain.equation != point.equation
        || domain.jacobian.len() != columns
        || point.jacobian.len() != columns
    {
        return Err(SketchAffineError::Input(format!(
            "affine domain row {:?}/{} and anchor row {:?}/{}: expected matching identities and {columns} columns",
            domain.relation, domain.equation, point.relation, point.equation
        )));
    }
    domain
        .jacobian
        .iter()
        .zip(&point.jacobian)
        .map(|(&interval, &at_point)| {
            let value = singleton(domain, "derivative", interval)?;
            if at_point != [value, value] {
                return Err(SketchAffineError::Indeterminate {
                    relation: domain.relation,
                    equation: domain.equation,
                    field: "anchor derivative",
                    bounds: at_point,
                });
            }
            Ok(value)
        })
        .collect()
}

fn intercept(
    coefficients: &[f64],
    point: &SketchIntervalEquation,
    anchor: &[f64],
) -> Result<f64, SketchAffineError> {
    let value = singleton(point, "anchor value", point.value)?;
    let mut total = interval([0., 0.])?;
    for (&coefficient, &coordinate) in coefficients.iter().zip(anchor) {
        let product = interval([coefficient, coefficient])?
            .product(interval([coordinate, coordinate])?)
            .map_err(SketchAffineError::Arithmetic)?;
        total = total.sum(product).map_err(SketchAffineError::Arithmetic)?;
    }
    let rhs = total
        .difference(interval([value, value])?)
        .map_err(SketchAffineError::Arithmetic)?;
    singleton(point, "right side", rhs.bounds())
}

fn interval(bounds: [f64; 2]) -> Result<Enclosure, SketchAffineError> {
    Enclosure::new(bounds).map_err(SketchAffineError::Arithmetic)
}
fn singleton(
    row: &SketchIntervalEquation,
    field: &'static str,
    bounds: [f64; 2],
) -> Result<f64, SketchAffineError> {
    if bounds[0].is_finite() && bounds[0] == bounds[1] {
        return Ok(bounds[0]);
    }
    Err(SketchAffineError::Indeterminate {
        relation: row.relation,
        equation: row.equation,
        field,
        bounds,
    })
}
