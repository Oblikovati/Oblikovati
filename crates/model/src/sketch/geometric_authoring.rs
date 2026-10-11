// SPDX-License-Identifier: GPL-2.0-only
use super::{
    CircleContact, LineContactSide, PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId,
    SketchError, dependencies::dimension_parameter,
};
/// Typed geometric intent; seed-dependent relations capture current owner coordinates.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchGeometricConstraintCreation {
    Relation(PlanarConstraint),
    Fix {
        point: SketchEntityId,
    },
    LineTangent {
        line: SketchEntityId,
        curve: SketchEntityId,
    },
    CircularTangent {
        curves: [SketchEntityId; 2],
    },
}
impl SketchGeometricConstraintCreation {
    pub(crate) fn validate(&self, geometry: &Sketch2D) -> Result<(), SketchError> {
        if let Self::Relation(relation) = self {
            require_direct_relation(relation)?;
            return geometry.validate_constraint_relation(relation);
        }
        geometry.validate_constraint_relation(&self.seed_validation_relation(geometry)?)
    }
    fn seed_validation_relation(
        &self,
        geometry: &Sketch2D,
    ) -> Result<PlanarConstraint, SketchError> {
        // These temporary records validate ownership only; append selects and retains the real seed branch.
        Ok(match self {
            Self::Relation(relation) => relation.clone(),
            Self::Fix { point } => PlanarConstraint::Fix {
                point: *point,
                position: geometry.point(*point)?.position,
            },
            Self::LineTangent { line, curve } => PlanarConstraint::LineTangent {
                line: *line,
                curve: *curve,
                side: LineContactSide::Left,
            },
            Self::CircularTangent { curves } => PlanarConstraint::CircularTangent {
                curves: *curves,
                contact: CircleContact::External,
            },
        })
    }
    pub(crate) fn requires_current_seed(&self) -> bool {
        !matches!(self, Self::Relation(_))
    }
    pub(crate) fn append(
        &self,
        geometry: &mut Sketch2D,
        id: SketchConstraintId,
    ) -> Result<(), SketchError> {
        match self {
            Self::Relation(relation) => geometry.add_constraint(id, relation.clone()),
            Self::Fix { point } => {
                let position = geometry.point(*point)?.position;
                geometry.add_constraint(
                    id,
                    PlanarConstraint::Fix {
                        point: *point,
                        position,
                    },
                )
            }
            Self::LineTangent { line, curve } => geometry.add_line_tangent(id, *line, *curve),
            Self::CircularTangent { curves } => geometry.add_circular_tangent(id, *curves),
        }
    }
}
fn require_direct_relation(relation: &PlanarConstraint) -> Result<(), SketchError> {
    if dimension_parameter(relation).is_some()
        || matches!(
            relation,
            PlanarConstraint::Fix { .. }
                | PlanarConstraint::LineTangent { .. }
                | PlanarConstraint::CircularTangent { .. }
        )
    {
        return Err(SketchError::Definition(format!(
            "geometric creation {relation:?}: expected a seed-independent geometric relation; fix and tangent require typed current-seed inputs"
        )));
    }
    Ok(())
}
