// SPDX-License-Identifier: GPL-2.0-only
/// An exact source curve in a planar prism boundary.
#[derive(Clone, Debug, PartialEq)]
pub enum PrismProbeLoop {
    /// Ordered segments: edge i joins vertex i to the next vertex, wrapping at the end.
    Polygon {
        edge_keys: Vec<u64>,
        vertices: Vec<[f64; 2]>,
    },
    /// One closed analytic circle. Its edge remains a single provenance source.
    Circle {
        edge_key: u64,
        center: [f64; 2],
        radius: f64,
    },
}

impl PrismProbeLoop {
    /// Build a polygonal loop; example: `PrismProbeLoop::polygon(vec![1, 2, 3], vertices)`.
    pub fn polygon(edge_keys: Vec<u64>, vertices: Vec<[f64; 2]>) -> Self {
        Self::Polygon {
            edge_keys,
            vertices,
        }
    }

    /// Build an analytic circular loop; example: `PrismProbeLoop::circle(1, [0., 0.], 2.)`.
    pub fn circle(edge_key: u64, center: [f64; 2], radius: f64) -> Self {
        Self::Circle {
            edge_key,
            center,
            radius,
        }
    }

    /// Return source identities in traversal order; example: `loop_.edge_keys()`.
    pub fn edge_keys(&self) -> Vec<u64> {
        match self {
            Self::Polygon { edge_keys, .. } => edge_keys.clone(),
            Self::Circle { edge_key, .. } => vec![*edge_key],
        }
    }
}
/// A local planar profile with an outer loop and oriented inner loops in working units.
/// Displacement components follow local X, Y and normal; the default entry point places it on XY.
#[derive(Clone, Debug, PartialEq)]
pub struct PrismProbeDefinition {
    pub outer: PrismProbeLoop,
    pub holes: Vec<PrismProbeLoop>,
    pub displacement: [f64; 3],
}
/// Semantic construction role, independent of pointer values and face enumeration.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum PrismFaceRole {
    Start,
    End,
    ProfileEdge(u64),
}
#[derive(Clone, Debug, PartialEq)]
pub struct PrismFaceProbe {
    pub role: PrismFaceRole,
    pub area: f64,
    pub centroid: [f64; 3],
    pub loops: usize,
}
/// Diagnostic acceptance report; full feature validity, tolerance budgets, and persistent keys remain open.
#[derive(Clone, Debug, PartialEq)]
pub struct PrismProbeReport {
    pub properties: crate::ReleaseProbeProperties,
    pub topology: PrismTopologySummary,
    pub faces: Vec<PrismFaceProbe>,
}

/// Unique entities in the retained B-rep, independent of mesh resolution and orientation uses.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PrismTopologySummary {
    pub solid: bool,
    pub faces: usize,
    pub edges: usize,
    pub vertices: usize,
    pub shells: usize,
    pub wires: usize,
}
