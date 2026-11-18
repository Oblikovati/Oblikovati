// SPDX-License-Identifier: GPL-2.0-only
//! Classify the supported straight-hole contract before editing a document (#4093).
use super::*;
use oblikovati_model::feature::{
    hole::{HoleExtent, HolePosition, HoleRecipe},
    work_geometry::DatumLength,
};

pub(super) fn require_supported(input: &Hole) -> Result<(), Error> {
    let mut unsupported = input.clone();
    for (name, value, accepted) in [
        ("type", &input.type_, "drilled"),
        ("tap", &input.tap, "none"),
        ("drillPoint", &input.drill_point, "flat"),
    ] {
        if !value.is_empty() && !value.eq_ignore_ascii_case(accepted) {
            return Err(Error(format!(
                "hole {name} {value:?}: expected {accepted:?} or its empty default"
            )));
        }
    }
    unsupported.face_ref.clear();
    unsupported.type_.clear();
    unsupported.tap.clear();
    unsupported.drill_point.clear();
    unsupported.diameter.clear();
    unsupported.depth.clear();
    unsupported.termination.clear();
    unsupported.center = None;
    unsupported.center_expr = None;
    if unsupported != Hole::default() {
        return Err(Error(format!(
            "hole options {unsupported:?}: expected a straight, flat-ended, face-placed bore"
        )));
    }
    Ok(())
}
pub(super) fn recipe(
    snapshot: &DocumentSnapshot,
    body: Vec<u8>,
    face: Vec<u8>,
    input: &Hole,
) -> Result<HoleRecipe, Error> {
    Ok(HoleRecipe {
        version: 1,
        body,
        face,
        position: position(snapshot, input)?,
        diameter: crate::extrusion::positive_length(snapshot, &input.diameter, "hole diameter")?,
        extent: extent(snapshot, input)?,
    })
}
fn extent(snapshot: &DocumentSnapshot, input: &Hole) -> Result<HoleExtent, Error> {
    let termination = input.termination.trim();
    if termination.is_empty() && input.depth.trim().is_empty() {
        return Ok(HoleExtent::ThroughAll);
    }
    if termination.eq_ignore_ascii_case("through-all") && input.depth.trim().is_empty() {
        return Ok(HoleExtent::ThroughAll);
    }
    if (termination.is_empty() || termination.eq_ignore_ascii_case("distance"))
        && !input.depth.trim().is_empty()
    {
        return crate::extrusion::positive_length(snapshot, &input.depth, "hole depth")
            .map(HoleExtent::Distance);
    }
    Err(Error(format!(
        "hole termination {termination:?}, depth {:?}: expected through-all without depth or distance with a positive depth",
        input.depth
    )))
}
fn position(snapshot: &DocumentSnapshot, input: &Hole) -> Result<HolePosition, Error> {
    if let Some(expressions) = &input.center_expr {
        let values: &[String; 3] = expressions.as_slice().try_into().map_err(|_| {
            Error(format!(
                "hole centerExpr {expressions:?}: expected exactly three length expressions"
            ))
        })?;
        let [x, y, z] = values;
        return Ok(HolePosition::ModelPoint([
            crate::extrusion::finite_length(snapshot, x, "hole center x")?,
            crate::extrusion::finite_length(snapshot, y, "hole center y")?,
            crate::extrusion::finite_length(snapshot, z, "hole center z")?,
        ]));
    }
    let Some(center) = &input.center else {
        return Ok(HolePosition::FaceCentroid);
    };
    let point: [f64; 3] = center.as_slice().try_into().map_err(|_| {
        Error(format!(
            "hole center {center:?}: expected three finite model coordinates"
        ))
    })?;
    if point.iter().any(|value| !value.is_finite()) {
        return Err(Error(format!(
            "hole center {point:?}: expected three finite model coordinates"
        )));
    }
    Ok(HolePosition::ModelPoint(point.map(DatumLength::Constant)))
}
