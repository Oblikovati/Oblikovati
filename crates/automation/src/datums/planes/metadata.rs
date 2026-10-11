// SPDX-License-Identifier: GPL-2.0-only
use super::axis_point::AxisPointPlaneConstructor;
use super::*;
use oblikovati_model::quantity::{Dimension, Quantity};
pub(super) fn constructor(definition: WorkGeometryDefinition) -> Result<String, Error> {
    match definition {
        WorkGeometryDefinition::ThroughTwoAxes(_) => Ok(WORK_PLANE_TWO_LINES.0.into_owned()),
        WorkGeometryDefinition::NormalToAxis(_) => Ok(WORK_PLANE_NORMAL_TO_CURVE.0.into_owned()),
        WorkGeometryDefinition::LineThroughPoint(_) => Ok(WORK_PLANE_LINE_AND_POINT.0.into_owned()),
        WorkGeometryDefinition::ThroughThreePoints(_) => Ok(WORK_PLANE_THREE_POINTS.0.into_owned()),
        WorkGeometryDefinition::Fixed(_) => Ok(WORK_PLANE_FIXED.0.into_owned()),
        WorkGeometryDefinition::ParallelThroughPoint(_) => {
            Ok(WORK_PLANE_PLANE_AND_POINT.0.into_owned())
        }
        WorkGeometryDefinition::Offset { .. } => Ok(WORK_PLANE_OFFSET.0.into_owned()),
        _ => Err(Error(format!(
            "plane definition {definition:?}: expected a supported plane definition"
        ))),
    }
}
pub(super) fn append_inputs(
    info: &mut WorkPlaneInfo,
    snapshot: &DocumentSnapshot,
    definition: WorkGeometryDefinition,
) -> Result<(), Error> {
    info.slots = reference_slots(definition);
    if let WorkGeometryDefinition::Offset { distance, .. } = definition
        && let Ok(value) = distance.resolve(&snapshot.parameters)
    {
        info.scalars = Some(vec![offset_scalar(snapshot, value)?]);
    }
    Ok(())
}
fn reference_slots(definition: WorkGeometryDefinition) -> Option<Vec<WorkPlaneRefSlot>> {
    match definition {
        WorkGeometryDefinition::ThroughTwoAxes(_) => Some(super::two_axes::slots()),
        WorkGeometryDefinition::NormalToAxis(_) => {
            Some(AxisPointPlaneConstructor::NormalToAxis.slots())
        }
        WorkGeometryDefinition::LineThroughPoint(_) => {
            Some(AxisPointPlaneConstructor::ContainsLine.slots())
        }
        WorkGeometryDefinition::ThroughThreePoints(_) => Some(super::three_points::slots()),
        WorkGeometryDefinition::ParallelThroughPoint(_) => Some(super::parallel::slots()),
        WorkGeometryDefinition::Offset { .. } => {
            Some(super::reference_only::slots(&[("Base plane", "plane")]))
        }
        _ => None,
    }
}

fn offset_scalar(snapshot: &DocumentSnapshot, value: f64) -> Result<WorkPlaneScalar, Error> {
    let units = snapshot.parameters.units();
    let displayed = units.to_preferred(Quantity::new(value, Dimension::Length));
    if !displayed.is_finite() {
        return Err(Error(format!(
            "offset {value} displayed as {displayed}: expected a finite scalar in preferred units"
        )));
    }
    Ok(WorkPlaneScalar {
        label: "Offset".into(),
        unit: units.preferred_name(Dimension::Length).into(),
        value: displayed,
        ..Default::default()
    })
}
