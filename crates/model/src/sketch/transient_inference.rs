// SPDX-License-Identifier: GPL-2.0-only
//! Solve inferred placement guidance without storing temporary relations or freezing durable DOF.
use super::{
    PlanarConstraint, Sketch2D, SketchConstraintId, SketchEntityId, SketchError, SketchPoint2D,
    grounded, variables::SketchVariable,
};
use crate::parameter::ParameterCollection;
mod source;
use oblikovati_solve::{
    diagnostics::ConstraintStructure,
    nonlinear::{SolveOptions, SolveTermination},
};
use std::{
    collections::{BTreeMap, BTreeSet},
    error::Error,
    fmt,
};

/// Immutable pre-placement solver values, keyed by owned entity identity.
/// Example: `let source = SketchInferenceFreeze::capture(&sketch)?;` before adding geometry.
#[derive(Clone, Debug)]
pub struct SketchInferenceFreeze {
    points: Vec<SketchPoint2D>,
    scalars: Vec<(SketchVariable, f64)>,
    entities: Vec<SketchEntityId>,
    constraints: Vec<super::SketchConstraint>,
    topology: source::FrozenTopology,
}
impl SketchInferenceFreeze {
    /// Capture the complete source variable set without altering the sketch.
    /// Example: `let frozen = SketchInferenceFreeze::capture(&source)?;`.
    pub fn capture(sketch: &Sketch2D) -> Result<Self, SketchError> {
        Ok(Self {
            points: sketch.points().to_vec(),
            scalars: capture_scalars(sketch)?,
            entities: sketch.entity_ids().to_vec(),
            constraints: sketch.constraints().to_vec(),
            topology: source::FrozenTopology::capture(sketch),
        })
    }
    fn validate_points(&self, sketch: &Sketch2D) -> Result<(), SketchTransientSolveError> {
        self.validate_structure(sketch)?;
        for point in &self.points {
            let actual = sketch.point(point.id)?.position;
            if bits(actual) != bits(point.position) {
                return Err(SketchTransientSolveError::SourcePointChanged {
                    point: point.id,
                    expected: point.position,
                    actual,
                });
            }
        }
        Ok(())
    }
    fn validate_structure(&self, sketch: &Sketch2D) -> Result<(), SketchTransientSolveError> {
        self.topology.validate(sketch)?;
        if !sketch.entity_ids().starts_with(&self.entities) {
            return Err(SketchTransientSolveError::SourceStructureChanged(
                "source entity order or identity changed".into(),
            ));
        }
        let valid = sketch.constraints().len() >= self.constraints.len()
            && self
                .constraints
                .iter()
                .zip(sketch.constraints())
                .all(|(first, second)| {
                    first.id == second.id
                        && super::recipe_compare::same_relation(&first.relation, &second.relation)
                });
        if !valid {
            return Err(SketchTransientSolveError::SourceStructureChanged(
                "source constraints changed".into(),
            ));
        }
        Ok(())
    }
}
fn capture_scalars(sketch: &Sketch2D) -> Result<Vec<(SketchVariable, f64)>, SketchError> {
    let variables = sketch
        .circles()
        .iter()
        .map(|circle| SketchVariable::Radius(circle.id))
        .chain(
            sketch
                .ellipses()
                .iter()
                .flat_map(|ellipse| super::ellipse::variables::ellipse_variables(ellipse.id())),
        )
        .chain(
            sketch
                .offset_splines()
                .iter()
                .map(|offset| SketchVariable::OffsetSpline(offset.id())),
        );
    let values = sketch.variable_values();
    variables
        .map(|variable| {
            let column = sketch.variable_column(variable)?;
            let value = values.get(column).copied().ok_or_else(|| SketchError::Definition(format!(
            "inference source scalar {variable:?} column {column}: expected an owned solver value"
        )))?;
            Ok((variable, value))
        })
        .collect()
}

/// Temporary inference inputs. Durable anchors are the document's real projected sources.
/// Example: `SketchTransientInference { source: &freeze, durable_anchors: &[], temporary_anchors: &[], temporary_relations: &[] }`.
pub struct SketchTransientInference<'a> {
    pub source: &'a SketchInferenceFreeze,
    pub durable_anchors: &'a [SketchPoint2D],
    pub temporary_anchors: &'a [SketchPoint2D],
    pub temporary_relations: &'a [PlanarConstraint],
}
/// Successful geometry and structure of the durable graph, without temporary identities.
#[derive(Debug)]
pub struct SketchTransientSolveOutcome {
    candidate: Sketch2D,
    durable_structure: ConstraintStructure,
}
impl SketchTransientSolveOutcome {
    /// Read the candidate without modifying it; example: `outcome.candidate().points()`.
    pub fn candidate(&self) -> &Sketch2D {
        &self.candidate
    }
    /// Read freedom without temporary guidance; example: `outcome.durable_structure().degrees_of_freedom`.
    pub fn durable_structure(&self) -> &ConstraintStructure {
        &self.durable_structure
    }
    /// Take the solved candidate; example: `let geometry = outcome.into_candidate();`.
    pub fn into_candidate(self) -> Sketch2D {
        self.candidate
    }
}
/// A transient inference solve declined before publication.
#[derive(Clone, Debug, PartialEq)]
pub enum SketchTransientSolveError {
    Sketch(SketchError),
    SourceStructureChanged(String),
    SourcePointChanged {
        point: SketchEntityId,
        expected: [f64; 2],
        actual: [f64; 2],
    },
    ConflictingAnchor {
        point: SketchEntityId,
        first: [f64; 2],
        second: [f64; 2],
    },
    TemporaryDimension {
        relation: PlanarConstraint,
    },
    TemporaryIdentityExhausted,
    NotConverged(SolveTermination),
}
impl From<SketchError> for SketchTransientSolveError {
    fn from(cause: SketchError) -> Self {
        Self::Sketch(cause)
    }
}
impl fmt::Display for SketchTransientSolveError {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Sketch(cause) => write!(out, "transient inference solve: {cause}"),
            Self::SourceStructureChanged(detail) => write!(
                out,
                "inference {detail}: expected source identities and constraints preserved"
            ),
            Self::SourcePointChanged {
                point,
                expected,
                actual,
            } => write!(
                out,
                "inference source point {point:?} is {actual:?}: expected unchanged {expected:?}"
            ),
            Self::ConflictingAnchor {
                point,
                first,
                second,
            } => write!(
                out,
                "conflicting inference anchors for {point:?}: {first:?} and {second:?}; expected identical positions"
            ),
            Self::TemporaryDimension { relation } => write!(
                out,
                "temporary inference relation {relation:?}: expected a geometric relation without a parameter"
            ),
            Self::TemporaryIdentityExhausted => write!(
                out,
                "temporary inference relation IDs exhausted: expected an unused nonzero identity"
            ),
            Self::NotConverged(status) => write!(
                out,
                "transient inference solve {status:?}: expected convergence"
            ),
        }
    }
}
impl Error for SketchTransientSolveError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Sketch(cause) => Some(cause),
            _ => None,
        }
    }
}
impl Sketch2D {
    /// Solve temporary inference on a private copy, then report only the durable graph.
    /// Example: `sketch.solve_transient_inference(&parameters, &context, options, || false)?`.
    pub fn solve_transient_inference(
        &self,
        parameters: &ParameterCollection<u64>,
        context: &SketchTransientInference<'_>,
        options: SolveOptions,
        cancelled: impl FnMut() -> bool,
    ) -> Result<SketchTransientSolveOutcome, SketchTransientSolveError> {
        context.source.validate_points(self)?;
        let anchors = merged_anchors(context)?;
        let (mut augmented, temporary_ids) =
            add_temporary_relations(self, context.temporary_relations)?;
        let prepared = grounded::prepare_with_fixed_scalars(
            &augmented,
            parameters,
            &anchors,
            &context.source.scalars,
        )?;
        let numerical = augmented.solve_prepared_grounded(&prepared, options, cancelled)?;
        if numerical.termination != SolveTermination::Converged {
            return Err(SketchTransientSolveError::NotConverged(
                numerical.termination,
            ));
        }
        augmented = augmented.candidate_values(&prepared.restore(&numerical.values)?)?;
        for id in temporary_ids {
            augmented.remove_authored_relation(id, false)?;
        }
        context.source.validate_points(&augmented)?;
        grounded::add_fixed_scalar_columns(
            &augmented,
            &context.source.scalars,
            &mut BTreeSet::new(),
        )?;
        let durable_structure = augmented.analyze_with_grounded_points(
            parameters,
            context.durable_anchors,
            options.rank_tolerance,
        )?;
        Ok(SketchTransientSolveOutcome {
            candidate: augmented,
            durable_structure,
        })
    }
}
fn merged_anchors(
    context: &SketchTransientInference<'_>,
) -> Result<Vec<SketchPoint2D>, SketchTransientSolveError> {
    let mut unique = BTreeMap::new();
    for point in context
        .source
        .points
        .iter()
        .chain(context.durable_anchors)
        .chain(context.temporary_anchors)
    {
        if let Some(previous) = unique.insert(point.id, point.position)
            && bits(previous) != bits(point.position)
        {
            return Err(SketchTransientSolveError::ConflictingAnchor {
                point: point.id,
                first: previous,
                second: point.position,
            });
        }
    }
    Ok(unique
        .into_iter()
        .map(|(id, position)| SketchPoint2D { id, position })
        .collect())
}
fn bits(position: [f64; 2]) -> [u64; 2] {
    position.map(f64::to_bits)
}
fn add_temporary_relations(
    sketch: &Sketch2D,
    relations: &[PlanarConstraint],
) -> Result<(Sketch2D, Vec<SketchConstraintId>), SketchTransientSolveError> {
    let mut candidate = sketch.clone();
    let mut used: BTreeSet<_> = sketch
        .constraints()
        .iter()
        .map(|constraint| constraint.id.0)
        .collect();
    let mut next = u64::MAX;
    let mut ids = Vec::with_capacity(relations.len());
    for relation in relations {
        if super::dependencies::dimension_parameter(relation).is_some() {
            return Err(SketchTransientSolveError::TemporaryDimension {
                relation: relation.clone(),
            });
        }
        while used.contains(&next) {
            next = next
                .checked_sub(1)
                .filter(|value| *value != 0)
                .ok_or(SketchTransientSolveError::TemporaryIdentityExhausted)?;
        }
        let id = SketchConstraintId(next);
        used.insert(next);
        candidate.add_constraint(id, relation.clone())?;
        ids.push(id);
    }
    Ok((candidate, ids))
}
