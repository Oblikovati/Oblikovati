// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{Error, types::*};
use oblikovati_model::sketch::{DistanceMeasure, PlanarConstraint};
type DimensionShape = (DimensionConstraintKind, &'static str, bool);
pub(super) fn require(relation: &PlanarConstraint) -> Result<DimensionShape, Error> {
    use PlanarConstraint::*;
    let shape = match relation {
        Distance { measure, .. } => (DIM_CONSTRAINT_DISTANCE, orientation(*measure), false),
        Radius { .. } => (DIM_CONSTRAINT_RADIUS, "", false),
        Diameter { .. } => (DIM_CONSTRAINT_DIAMETER, "", false),
        ArcLength { .. } => (DIM_CONSTRAINT_ARC_LENGTH, "", false),
        Offset {
            linear_diameter, ..
        } => (DIM_CONSTRAINT_OFFSET, "", *linear_diameter),
        Angle { .. } => (DIM_CONSTRAINT_ANGLE, "", false),
        ThreePointAngle { .. } => (DIM_CONSTRAINT_THREE_POINT_ANGLE, "", false),
        TangentDistance { options, .. } => {
            (DIM_CONSTRAINT_TANGENT_DISTANCE, "", options.linear_diameter)
        }
        _ => return unsupported(relation),
    };
    Ok(shape)
}
fn orientation(measure: DistanceMeasure) -> &'static str {
    match measure {
        DistanceMeasure::Aligned => "",
        DistanceMeasure::Horizontal => "horizontal",
        DistanceMeasure::Vertical => "vertical",
    }
}

fn unsupported(relation: &PlanarConstraint) -> Result<DimensionShape, Error> {
    Err(Error(format!(
        "dimension representation {relation:?}: expected an authored dimensional relation"
    )))
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn geometric_relations_cannot_be_reported_as_unknown_dimensions() {
        let relation = PlanarConstraint::LineHorizontal {
            line: oblikovati_model::sketch::SketchEntityId(7),
        };
        let cause = require(&relation).unwrap_err().0;
        assert!(cause.contains("LineHorizontal") && cause.contains("dimensional relation"));
    }
}
