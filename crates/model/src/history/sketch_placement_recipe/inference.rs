// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::sketch::{PlanarConstraint, SketchInferencePolicy, SketchPoint2D};

pub(super) struct InferredPlacement {
    pub relations: Vec<SketchConstraintId>,
    pub anchors: Vec<SketchPoint2D>,
}

#[derive(Clone, Copy, Default)]
pub(in crate::history) struct PlacementSolveContext<'a> {
    pub datums: Option<&'a crate::feature::work_geometry::WorkGeometryEvaluation>,
    pub transient: Option<&'a TransientPlacement>,
}

/// A placement-only constraint context; it never enters the document recipe.
#[derive(Clone, Debug)]
pub(in crate::history) struct TransientPlacement {
    pub source: crate::sketch::SketchInferenceFreeze,
    pub anchors: Vec<SketchPoint2D>,
    pub relations: Vec<PlanarConstraint>,
}
impl TransientPlacement {
    pub fn check_preserved(
        &self,
        geometry: &Sketch2D,
    ) -> Result<(), crate::sketch::SketchTransientSolveError> {
        self.source.validate_preserved(geometry)?;
        for anchor in &self.anchors {
            let actual = geometry.point(anchor.id)?.position;
            if actual.map(f64::to_bits) != anchor.position.map(f64::to_bits) {
                return Err(crate::sketch::SketchTransientSolveError::AnchorMoved {
                    point: anchor.id,
                    expected: anchor.position,
                    actual,
                });
            }
        }
        Ok(())
    }
    pub fn solve(
        &self,
        snapshot: &DocumentSnapshot,
        geometry: &Sketch2D,
        sketch: DocumentSketchId,
        options: SolveOptions,
        datums: Option<&crate::feature::work_geometry::WorkGeometryEvaluation>,
    ) -> Result<Sketch2D, crate::sketch::SketchTransientSolveError> {
        let durable_anchors = snapshot.sketch_authoring_anchors(sketch, datums)?;
        let context = crate::sketch::SketchTransientInference {
            source: &self.source,
            durable_anchors: &durable_anchors,
            temporary_anchors: &self.anchors,
            temporary_relations: &self.relations,
        };
        geometry
            .solve_transient_inference(&snapshot.parameters, &context, options, || false)
            .map(|outcome| outcome.into_candidate())
    }
}

pub(super) fn append(
    snapshot: &DocumentSnapshot,
    geometry: &mut Sketch2D,
    sketch: DocumentSketchId,
    fresh: &[SketchEntityId],
    ids: &mut RecipeReservations<'_>,
    scope: Option<&SketchInferenceCandidates>,
) -> Result<InferredPlacement, SketchPlacementCreationError> {
    let settings = snapshot.effective_sketch_settings();
    if !settings.auto_apply_constraints {
        return Ok(InferredPlacement {
            relations: Vec::new(),
            anchors: Vec::new(),
        });
    }
    let policy = SketchInferencePolicy::for_units(snapshot.parameters.units());
    let mut constraints = Vec::new();
    let mut anchors = Vec::new();
    for point in fresh {
        let position = geometry.point(*point)?.position;
        let target = match scope {
            Some(scope) => geometry
                .infer_point_coincidence_scoped(position, fresh, &settings, policy, scope, fresh)?,
            None => geometry.infer_point_coincidence(position, fresh, &settings, policy)?,
        };
        let Some(target) = target else {
            continue;
        };
        if !settings.persist_inferred_constraints {
            let position = geometry.point(target)?.position;
            geometry.set_position(*point, position)?;
            anchors.push(SketchPoint2D {
                id: *point,
                position,
            });
            continue;
        }
        let constraint = ids
            .allocate_sketch_constraint(sketch)
            .map_err(SketchPlacementCreationError::Constraint)?;
        geometry.add_constraint(
            constraint,
            PlanarConstraint::Coincident {
                points: [*point, target],
            },
        )?;
        constraints.push(constraint);
    }
    Ok(InferredPlacement {
        relations: constraints,
        anchors,
    })
}
