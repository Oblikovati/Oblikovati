// SPDX-License-Identifier: GPL-2.0-only
//! Reject a Boolean or shell result whose entity tolerances exceed the source and configured cap.
use super::{PrismAchievedTolerance, PrismResolution, Shape, solid_boolean::entities};
use martensite_tkbrep::{brep::BRepTool, topods};
use martensite_tkg3d::top_abs::ShapeEnum;

pub(super) fn check(
    source: &Shape,
    result: &Shape,
    resolution: Option<PrismResolution>,
    operation: &str,
) -> Result<PrismAchievedTolerance, String> {
    let resolution = resolution
        .ok_or_else(|| format!("{operation} input: expected a certified source resolution"))?
        .validate()?;
    let before = measure(source, operation)?;
    let after = measure(result, operation)?;
    for (kind, value, source_limit) in [
        ("vertex", after.vertex, before.vertex),
        ("edge", after.edge, before.edge),
        ("face", after.face, before.face),
    ] {
        let limit = source_limit.min(resolution.maximum_tolerance);
        if !value.is_finite() || value < 0. || value > limit {
            return Err(format!(
                "{operation} {kind} tolerance {value}: expected finite nonnegative tolerance at most {limit}"
            ));
        }
    }
    Ok(after)
}
pub(super) fn check_boolean<'a>(
    target: &Shape,
    tools: impl Iterator<Item = (&'a Shape, PrismResolution)>,
    result: &Shape,
    target_resolution: Option<PrismResolution>,
    operation: &str,
) -> Result<PrismAchievedTolerance, String> {
    let target_resolution = target_resolution
        .ok_or_else(|| format!("{operation}: expected a certified target resolution"))?
        .validate()?;
    require_budget(
        measure(target, operation)?,
        target_resolution.maximum_tolerance,
        operation,
    )?;
    let mut cap = target_resolution.maximum_tolerance;
    for (tool, resolution) in tools {
        let maximum = resolution.validate()?.maximum_tolerance;
        require_budget(measure(tool, operation)?, maximum, operation)?;
        cap = cap.min(maximum);
    }
    let achieved = measure(result, operation)?;
    require_budget(achieved, cap, operation)?;
    Ok(achieved)
}

fn require_budget(
    measured: PrismAchievedTolerance,
    cap: f64,
    operation: &str,
) -> Result<(), String> {
    for (kind, value) in [
        ("vertex", measured.vertex),
        ("edge", measured.edge),
        ("face", measured.face),
    ] {
        if value > cap {
            return Err(format!(
                "{operation} {kind} tolerance {value}: expected at most {cap}"
            ));
        }
    }
    Ok(())
}
fn measure(shape: &Shape, operation: &str) -> Result<PrismAchievedTolerance, String> {
    Ok(PrismAchievedTolerance {
        vertex: maximum(shape, ShapeEnum::Vertex, operation)?,
        edge: maximum(shape, ShapeEnum::Edge, operation)?,
        face: maximum(shape, ShapeEnum::Face, operation)?,
    })
}
fn maximum(shape: &Shape, kind: ShapeEnum, operation: &str) -> Result<f64, String> {
    let mut maximum: f64 = 0.;
    for entity in entities(shape, kind)? {
        let tolerance = match kind {
            ShapeEnum::Vertex => BRepTool::vertex_tolerance(
                &topods::vertex(&entity).map_err(|error| error.to_string())?,
            ),
            ShapeEnum::Edge => {
                BRepTool::edge_tolerance(&topods::edge(&entity).map_err(|error| error.to_string())?)
            }
            ShapeEnum::Face => {
                BRepTool::face_tolerance(&topods::face(&entity).map_err(|error| error.to_string())?)
            }
            _ => {
                return Err(format!(
                    "{operation} tolerance kind {kind:?}: expected vertex, edge or face"
                ));
            }
        }
        .map_err(|error| error.to_string())?;
        if !tolerance.is_finite() || tolerance < 0. {
            return Err(format!(
                "{operation} {kind:?} tolerance {tolerance}: expected finite nonnegative value"
            ));
        }
        maximum = maximum.max(tolerance);
    }
    Ok(maximum)
}

#[cfg(test)]
mod tests {
    use super::*;
    use martensite_tkprim::brep_prim_api::BRepPrimAPIMakeBox;
    use martensite_tktopalgo::brep_builder_api::BRepBuilderAPIMakeShape;

    #[test]
    fn tolerance_gate_rejects_a_cap_below_the_measured_geometry() {
        let shape = BRepPrimAPIMakeBox::from_sizes(1., 1., 1.)
            .unwrap()
            .shape()
            .unwrap()
            .clone();
        let bounds = PrismResolution {
            version: 1,
            weld: 1e-12,
            sew: 1e-11,
            angular: 1e-9,
            display_chordal: 1e-3,
            display_angular: 0.1,
            maximum_tolerance: 1e-10,
        };
        let error = check(&shape, &shape, Some(bounds), "hole").unwrap_err();
        assert!(
            error.contains("tolerance"),
            "expected a measured tolerance failure: {error}"
        );
        assert!(check(&shape, &shape, None, "hole").is_err());
    }

    #[test]
    fn tolerance_gate_refuses_widening_a_single_result_vertex() {
        let source = BRepPrimAPIMakeBox::from_sizes(1., 1., 1.)
            .unwrap()
            .shape()
            .unwrap()
            .clone();
        let mut copy = crate::checked_shape::copy_shape(&source, "tolerance test").unwrap();
        let result = copy.shape().unwrap().clone();
        let vertex = topods::vertex(&entities(&result, ShapeEnum::Vertex).unwrap()[0]).unwrap();
        martensite_tkbrep::brep::BRepBuilder::default()
            .update_vertex_tolerance(&vertex, 5e-7)
            .unwrap();
        let bounds = PrismResolution {
            version: 1,
            weld: 1e-6,
            sew: 1e-7,
            angular: 1e-9,
            display_chordal: 1e-3,
            display_angular: 0.1,
            maximum_tolerance: 1e-5,
        };
        let error = check(&source, &result, Some(bounds), "hole").unwrap_err();
        assert!(
            error.contains("vertex tolerance"),
            "expected a vertex widening failure: {error}"
        );
    }
}
