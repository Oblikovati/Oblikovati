// SPDX-License-Identifier: GPL-2.0-only
//! Topology-aware associative Work Plane definitions (Program v68, Source v58, #3918).

mod eval;

pub use eval::WorkPlaneExtendedResolver;

use super::{WorkPlaneError, WorkPlaneFaceReference, WorkPlaneReference};
use crate::{
    feature::{
        FeatureId, FeatureReadFootprint, FeatureReadKey,
        work_axis::{WorkAxisLine, WorkAxisReference},
        work_geometry::{DatumAngle, plane_key},
    },
    identity::{ReferenceEnvelope, topology_reference_document},
};

/// Durable B-rep subentity reference (edge kind 2 or vertex kind 3) with source producer ID.
#[derive(Clone, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(try_from = "TopologyRefWire", into = "TopologyRefWire")]
pub struct WorkPlaneTopologyRef {
    key: Vec<u8>,
    source: FeatureId,
    kind: u8,
}

#[derive(Clone, Debug, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
struct TopologyRefWire {
    key: Vec<u8>,
    kind: u8,
}

impl WorkPlaneTopologyRef {
    /// Validate a semantic B-rep vertex reference (kind 3); example: `WorkPlaneTopologyRef::vertex(key)?`.
    pub fn vertex(key: Vec<u8>) -> Result<Self, String> {
        Self::new(key, 3)
    }

    /// Validate a semantic B-rep edge reference (kind 2); example: `WorkPlaneTopologyRef::edge(key)?`.
    pub fn edge(key: Vec<u8>) -> Result<Self, String> {
        Self::new(key, 2)
    }

    fn new(key: Vec<u8>, expected_kind: u8) -> Result<Self, String> {
        topology_reference_document(&key)?;
        let envelope = ReferenceEnvelope::decode(&key).map_err(|cause| cause.to_string())?;
        if envelope.kind != u32::from(expected_kind) {
            return Err(format!(
                "topology key {key:?} kind {}: expected kind {expected_kind}",
                envelope.kind
            ));
        }
        Ok(Self {
            source: FeatureId(envelope.context),
            key,
            kind: expected_kind,
        })
    }

    /// Borrow the raw semantic reference key; example: `reference.key()`.
    pub fn key(&self) -> &[u8] {
        &self.key
    }

    /// Identify the feature that produced this entity; example: `reference.source()`.
    pub fn source(&self) -> FeatureId {
        self.source
    }
}

impl TryFrom<TopologyRefWire> for WorkPlaneTopologyRef {
    type Error = String;
    fn try_from(wire: TopologyRefWire) -> Result<Self, Self::Error> {
        if !matches!(wire.kind, 2 | 3) {
            return Err(format!(
                "work-plane topology ref kind {}: expected edge 2 or vertex 3",
                wire.kind
            ));
        }
        Self::new(wire.key, wire.kind)
    }
}

impl From<WorkPlaneTopologyRef> for TopologyRefWire {
    fn from(reference: WorkPlaneTopologyRef) -> Self {
        Self {
            key: reference.key,
            kind: reference.kind,
        }
    }
}

/// Point reference for extended work-plane constructors (datum point or B-rep vertex).
#[derive(Clone, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum WorkPlaneVertexReference {
    Origin,
    WorkPoint(FeatureId),
    TopologyVertex(WorkPlaneTopologyRef),
}

impl WorkPlaneVertexReference {
    fn read_key(&self) -> Option<FeatureReadKey> {
        match self {
            Self::Origin => None,
            Self::WorkPoint(id) => Some(FeatureReadKey::Feature(*id)),
            Self::TopologyVertex(reference) => Some(FeatureReadKey::Feature(reference.source())),
        }
    }

    fn requires_topology(&self) -> bool {
        matches!(self, Self::TopologyVertex(_))
    }

    /// Resolve to a concrete 3D point; example: `reference.resolve(&resolver)?`.
    pub fn resolve(
        &self,
        resolver: &impl WorkPlaneExtendedResolver,
    ) -> Result<[f64; 3], WorkPlaneError> {
        match self {
            Self::Origin => Ok([0.0; 3]),
            Self::WorkPoint(id) => resolver.point(*id),
            Self::TopologyVertex(reference) => resolver.topology_vertex(reference),
        }
    }
}

/// Line reference for extended work-plane constructors (origin axis, work axis, or straight edge).
#[derive(Clone, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
pub enum WorkPlaneEdgeReference {
    OriginX,
    OriginY,
    OriginZ,
    WorkAxis(FeatureId),
    TopologyEdge(WorkPlaneTopologyRef),
}

impl WorkPlaneEdgeReference {
    fn read_key(&self) -> Option<FeatureReadKey> {
        match self {
            Self::OriginX | Self::OriginY | Self::OriginZ => None,
            Self::WorkAxis(id) => Some(FeatureReadKey::Feature(*id)),
            Self::TopologyEdge(reference) => Some(FeatureReadKey::Feature(reference.source())),
        }
    }

    fn requires_topology(&self) -> bool {
        matches!(self, Self::TopologyEdge(_))
    }

    /// Resolve to an exact unit line; example: `reference.resolve(&resolver)?`.
    pub fn resolve(
        &self,
        resolver: &impl WorkPlaneExtendedResolver,
    ) -> Result<WorkAxisLine, WorkPlaneError> {
        match self {
            Self::OriginX => WorkAxisReference::OriginX.line(resolver),
            Self::OriginY => WorkAxisReference::OriginY.line(resolver),
            Self::OriginZ => WorkAxisReference::OriginZ.line(resolver),
            Self::WorkAxis(id) => resolver.axis(*id),
            Self::TopologyEdge(reference) => resolver.topology_edge(reference),
        }
    }
}

/// Analytic curved face reference for torus midplane and tangent work planes.
#[derive(Clone, Debug, Eq, PartialEq, serde::Serialize, serde::Deserialize)]
pub struct WorkPlaneSurfaceReference {
    pub face: WorkPlaneFaceReference,
}

impl WorkPlaneSurfaceReference {
    fn read_key(&self) -> FeatureReadKey {
        FeatureReadKey::Feature(self.face.source())
    }
}

/// Extended work-plane constructors supporting midplanes, B-rep vertices/edges, torus, and tangency.
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub enum WorkPlaneExtendedDefinition {
    MidplaneTwoPlanes {
        first: WorkPlaneReference,
        second: WorkPlaneReference,
    },
    ThreePoints {
        points: [WorkPlaneVertexReference; 3],
    },
    ParallelThroughPoint {
        base: WorkPlaneReference,
        point: WorkPlaneVertexReference,
    },
    NormalToAxisThroughPoint {
        axis: WorkPlaneEdgeReference,
        point: WorkPlaneVertexReference,
    },
    AngleToPlaneAroundEdge {
        base: WorkPlaneReference,
        edge: WorkPlaneEdgeReference,
        angle: DatumAngle,
    },
    TwoCoplanarEdges {
        first: WorkPlaneEdgeReference,
        second: WorkPlaneEdgeReference,
    },
    MidplaneOfTorus {
        surface: WorkPlaneSurfaceReference,
    },
    TangentThroughEdge {
        surface: WorkPlaneSurfaceReference,
        edge: WorkPlaneEdgeReference,
    },
    TangentThroughPoint {
        surface: WorkPlaneSurfaceReference,
        point: WorkPlaneVertexReference,
    },
    TangentParallelToPlane {
        surface: WorkPlaneSurfaceReference,
        plane: WorkPlaneReference,
    },
}

/// Versioned persisted source recipe for extended work-plane definitions (Source v58 / Program v68).
#[derive(Clone, Debug, PartialEq, serde::Serialize, serde::Deserialize)]
#[serde(deny_unknown_fields)]
pub struct WorkPlaneExtendedRecipe {
    pub definition: WorkPlaneExtendedDefinition,
}

impl WorkPlaneExtendedDefinition {
    /// Capture authored source recipe; example: `definition.recipe()`.
    pub fn recipe(&self) -> WorkPlaneExtendedRecipe {
        WorkPlaneExtendedRecipe {
            definition: self.clone(),
        }
    }

    /// Restore and validate an extended recipe; example: `WorkPlaneExtendedDefinition::from_recipe(recipe)?`.
    pub fn from_recipe(recipe: WorkPlaneExtendedRecipe) -> Result<Self, WorkPlaneError> {
        if let Self::AngleToPlaneAroundEdge { angle, .. } = &recipe.definition {
            angle.validate_source()?;
        }
        Ok(recipe.definition)
    }

    /// Attribute all feature and parameter dependencies; example: `definition.reads()`.
    pub fn reads(&self) -> FeatureReadFootprint {
        let mut keys = Vec::new();
        match self {
            Self::MidplaneTwoPlanes { first, second } => {
                keys.extend(plane_key(first));
                keys.extend(plane_key(second));
            }
            Self::ThreePoints { points } => {
                keys.extend(points.iter().filter_map(WorkPlaneVertexReference::read_key));
            }
            Self::ParallelThroughPoint { base, point } => {
                keys.extend(plane_key(base));
                keys.extend(point.read_key());
            }
            Self::NormalToAxisThroughPoint { axis, point } => {
                keys.extend(axis.read_key());
                keys.extend(point.read_key());
            }
            Self::AngleToPlaneAroundEdge { base, edge, angle } => {
                keys.extend(plane_key(base));
                keys.extend(edge.read_key());
                keys.extend(angle.parameter_id().map(FeatureReadKey::Parameter));
            }
            Self::TwoCoplanarEdges { first, second } => {
                keys.extend(first.read_key());
                keys.extend(second.read_key());
            }
            Self::MidplaneOfTorus { surface } => keys.push(surface.read_key()),
            Self::TangentThroughEdge { surface, edge } => {
                keys.push(surface.read_key());
                keys.extend(edge.read_key());
            }
            Self::TangentThroughPoint { surface, point } => {
                keys.push(surface.read_key());
                keys.extend(point.read_key());
            }
            Self::TangentParallelToPlane { surface, plane } => {
                keys.push(surface.read_key());
                keys.extend(plane_key(plane));
            }
        }
        FeatureReadFootprint::new(keys, false)
    }

    /// Report whether replay must resolve named B-rep entities on a solid body.
    pub fn requires_topology(&self) -> bool {
        let plane_is_face = |plane: &WorkPlaneReference| {
            matches!(
                plane,
                WorkPlaneReference::Face(_) | WorkPlaneReference::GoPrismFace(_)
            )
        };
        match self {
            Self::MidplaneTwoPlanes { first, second } => {
                plane_is_face(first) || plane_is_face(second)
            }
            Self::ThreePoints { points } => points
                .iter()
                .any(WorkPlaneVertexReference::requires_topology),
            Self::ParallelThroughPoint { base, point } => {
                plane_is_face(base) || point.requires_topology()
            }
            Self::NormalToAxisThroughPoint { axis, point } => {
                axis.requires_topology() || point.requires_topology()
            }
            Self::AngleToPlaneAroundEdge { base, edge, .. } => {
                plane_is_face(base) || edge.requires_topology()
            }
            Self::TwoCoplanarEdges { first, second } => {
                first.requires_topology() || second.requires_topology()
            }
            Self::MidplaneOfTorus { .. }
            | Self::TangentThroughEdge { .. }
            | Self::TangentThroughPoint { .. }
            | Self::TangentParallelToPlane { .. } => true,
        }
    }

    /// Compare authored recipe bits exactly; example: `before.same_recipe(&after)`.
    pub fn same_recipe(&self, other: &Self) -> bool {
        match (self, other) {
            (
                Self::AngleToPlaneAroundEdge {
                    base: b1,
                    edge: e1,
                    angle: a1,
                },
                Self::AngleToPlaneAroundEdge {
                    base: b2,
                    edge: e2,
                    angle: a2,
                },
            ) => b1 == b2 && e1 == e2 && a1.same_recipe(*a2),
            _ => self == other,
        }
    }
}
