// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::quantity::{Dimension, Quantity};
pub(super) fn constructor(definition: WorkGeometryDefinition) -> Result<String, Error> {
    match definition {
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
    if let WorkGeometryDefinition::ParallelThroughPoint(_) = definition {
        info.slots = Some(super::parallel::slots());
    }
    if let WorkGeometryDefinition::Offset { distance, .. } = definition {
        info.slots = Some(vec![WorkPlaneRefSlot {
            index: 0,
            label: "Base plane".into(),
            kind: "plane".into(),
        }]);
        if let Ok(value) = distance.resolve(&snapshot.parameters) {
            info.scalars = Some(vec![offset_scalar(snapshot, value)?]);
        }
    }
    Ok(())
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
