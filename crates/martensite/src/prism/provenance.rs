// SPDX-License-Identifier: GPL-2.0-only
//! Retain kernel edge and vertex history with analytic representative points.
use super::*;
use martensite_tkbo::profile_extrusion::{
    HistoryRecord, HistoryRelation, HistoryRole, HistorySource, OperationResult, ResultEntity,
};
use martensite_tkbrep::{
    brep_adaptor::BRepAdaptorCurve,
    topods::{self, Shape},
};
use martensite_tkg3d::adaptor3d::Adaptor3dCurve;
use martensite_tkmath::bnd::BoundingBox3;
use martensite_tktopalgo::brep_bnd_lib::BRepBndLib;
use std::collections::BTreeSet;

/// The authored identity from which an edge or vertex was produced.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Ord, PartialOrd)]
pub enum PrismHistorySource {
    Curve(u64),
    Vertex(u64),
}
/// The construction role of a named edge or vertex.
#[derive(Clone, Copy, Debug, Eq, PartialEq, Ord, PartialOrd)]
pub enum PrismSubentityRole {
    StartEdge,
    EndEdge,
    LateralEdge,
    StartVertex,
    EndVertex,
}
/// Whether the kernel generated or modified this entity.
#[derive(Clone, Copy, Debug, Eq, PartialEq)]
pub enum PrismHistoryRelation {
    Generated,
    Modified,
}
/// A result's parent and representative point, independent of native handle order.
#[derive(Clone, Debug, PartialEq)]
pub struct PrismSubentityProvenance {
    pub role: PrismSubentityRole,
    pub source: PrismHistorySource,
    pub relation: PrismHistoryRelation,
    pub point: [f64; 3],
}
/// Complete lower-dimensional result lineage, in semantic order.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct PrismSubentityHistory {
    pub edges: Vec<PrismSubentityProvenance>,
    pub vertices: Vec<PrismSubentityProvenance>,
}
pub(super) fn capture(
    result: &OperationResult,
) -> Result<PrismSubentityHistory, EvaluationFailure> {
    let edges = result
        .edges()
        .iter()
        .enumerate()
        .map(|(index, edge)| record(result, ResultEntity::Edge(index), &edge.clone().into()))
        .collect::<Result<Vec<_>, _>>()?;
    let vertices = result
        .vertices()
        .iter()
        .enumerate()
        .map(|(index, vertex)| record(result, ResultEntity::Vertex(index), &vertex.clone().into()))
        .collect::<Result<Vec<_>, _>>()?;
    Ok(PrismSubentityHistory {
        edges: ordered(edges)?,
        vertices: ordered(vertices)?,
    })
}
pub(super) fn source_shapes(
    result: &OperationResult,
    history: &PrismSubentityHistory,
) -> Result<Vec<Shape>, EvaluationFailure> {
    history
        .edges
        .iter()
        .chain(&history.vertices)
        .map(|wanted| {
            let matches: Vec<_> = result
                .history()
                .records
                .iter()
                .filter(|record| {
                    semantic_role(record, record.result)
                        .is_ok_and(|(source, role)| source == wanted.source && role == wanted.role)
                })
                .collect();
            let [record] = matches.as_slice() else {
                return Err(Phase::History.refused(
                    Kind::IncompleteHistory,
                    Entity::Body,
                    format!(
                        "extrusion {:?} {:?} has {} matching source shapes: expected one",
                        wanted.role,
                        wanted.source,
                        matches.len()
                    ),
                ));
            };
            match record.result {
                ResultEntity::Edge(index) => result.edges().get(index).cloned().map(Into::into),
                ResultEntity::Vertex(index) => {
                    result.vertices().get(index).cloned().map(Into::into)
                }
                ResultEntity::Face(_) => None,
            }
            .ok_or_else(|| {
                Phase::History.refused(
                    Kind::IncompleteHistory,
                    Entity::Body,
                    format!(
                        "extrusion {:?}: expected an indexed edge or vertex",
                        record.result
                    ),
                )
            })
        })
        .collect()
}
fn record(
    result: &OperationResult,
    entity: ResultEntity,
    shape: &Shape,
) -> Result<PrismSubentityProvenance, EvaluationFailure> {
    let records: Vec<_> = result
        .history()
        .records
        .iter()
        .filter(|record| record.result == entity)
        .collect();
    let [record] = records.as_slice() else {
        return Err(Phase::History.refused(
            Kind::IncompleteHistory,
            Entity::Body,
            format!(
                "extrusion {entity:?} has {} parents: expected one",
                records.len()
            ),
        ));
    };
    let (source, role) = semantic_role(record, entity)?;
    let relation = match record.relation {
        HistoryRelation::Generated => PrismHistoryRelation::Generated,
        HistoryRelation::Modified => PrismHistoryRelation::Modified,
    };
    Ok(PrismSubentityProvenance {
        role,
        source,
        relation,
        point: match entity {
            ResultEntity::Edge(_) => edge_midpoint(shape)?,
            _ => representative(shape, &format!("{entity:?}"))?,
        },
    })
}
fn semantic_role(
    record: &HistoryRecord,
    entity: ResultEntity,
) -> Result<(PrismHistorySource, PrismSubentityRole), EvaluationFailure> {
    let source = match record.source {
        HistorySource::Curve(id) => PrismHistorySource::Curve(id.0),
        HistorySource::Vertex(id) => PrismHistorySource::Vertex(id.0),
        _ => {
            return Err(Phase::History.refused(
                Kind::IncompleteHistory,
                Entity::Body,
                format!(
                    "extrusion {entity:?} source {:?}: expected a curve or vertex",
                    record.source
                ),
            ));
        }
    };
    let role = match (entity, record.role) {
        (ResultEntity::Edge(_), HistoryRole::StartEdge) => PrismSubentityRole::StartEdge,
        (ResultEntity::Edge(_), HistoryRole::EndEdge) => PrismSubentityRole::EndEdge,
        (ResultEntity::Edge(_), HistoryRole::LateralEdge) => PrismSubentityRole::LateralEdge,
        (ResultEntity::Vertex(_), HistoryRole::StartVertex) => PrismSubentityRole::StartVertex,
        (ResultEntity::Vertex(_), HistoryRole::EndVertex) => PrismSubentityRole::EndVertex,
        _ => {
            return Err(Phase::History.refused(
                Kind::IncompleteHistory,
                Entity::Body,
                format!(
                    "extrusion {entity:?} role {:?}: expected a matching edge or vertex role",
                    record.role
                ),
            ));
        }
    };
    if matches!(
        role,
        PrismSubentityRole::StartEdge | PrismSubentityRole::EndEdge
    ) && !matches!(source, PrismHistorySource::Curve(_))
    {
        return Err(Phase::History.refused(
            Kind::IncompleteHistory,
            Entity::Body,
            format!(
                "extrusion {entity:?} role {role:?}, source {source:?}: expected a curve parent"
            ),
        ));
    }
    Ok((source, role))
}
fn edge_midpoint(shape: &Shape) -> Result<[f64; 3], EvaluationFailure> {
    let measure = || -> Result<[f64; 3], martensite_tkernel::standard::Failure> {
        let edge = topods::edge(shape)?;
        let curve = BRepAdaptorCurve::with_edge(&edge)?;
        let first = curve.first_parameter()?;
        let last = curve.last_parameter()?;
        let point = curve.value(first / 2. + last / 2.)?;
        Ok([point.x(), point.y(), point.z()])
    };
    let point = measure().map_err(|cause| {
        Phase::Measurement.native(Entity::Body, "extrusion edge midpoint".into(), cause)
    })?;
    if point.iter().any(|value| !value.is_finite()) {
        return Err(Phase::Measurement.refused(
            Kind::InvalidResult,
            Entity::Body,
            format!("extrusion edge midpoint {point:?}: expected finite coordinates"),
        ));
    }
    Ok(point)
}
pub(super) fn representative(shape: &Shape, entity: &str) -> Result<[f64; 3], EvaluationFailure> {
    let mut bounds = BoundingBox3::default();
    BRepBndLib::add_optimal(shape, &mut bounds, false, false).map_err(|cause| {
        Phase::Measurement.native(
            Entity::Body,
            format!("extrusion {entity:?} analytic bounds"),
            cause,
        )
    })?;
    let center = bounds.center().ok_or_else(|| {
        Phase::Measurement.refused(
            Kind::InvalidResult,
            Entity::Body,
            format!("extrusion {entity:?} bounds: expected a finite center"),
        )
    })?;
    let point = [center.x(), center.y(), center.z()];
    if point.iter().any(|value| !value.is_finite()) {
        return Err(Phase::Measurement.refused(
            Kind::InvalidResult,
            Entity::Body,
            format!("extrusion {entity:?} center {point:?}: expected finite coordinates"),
        ));
    }
    Ok(point)
}
fn ordered(
    mut records: Vec<PrismSubentityProvenance>,
) -> Result<Vec<PrismSubentityProvenance>, EvaluationFailure> {
    records.sort_by_key(|record| (record.role, record.source));
    let identities: BTreeSet<_> = records
        .iter()
        .map(|record| (record.role, record.source))
        .collect();
    if identities.len() != records.len() {
        return Err(Phase::History.refused(
            Kind::IncompleteHistory,
            Entity::Body,
            format!("extrusion subentity history {records:?}: expected unique source-role pairs"),
        ));
    }
    Ok(records)
}

#[cfg(test)]
#[path = "../../tests/unit/prism_provenance.rs"]
mod tests;
