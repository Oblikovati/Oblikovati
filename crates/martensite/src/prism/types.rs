// SPDX-License-Identifier: GPL-2.0-only
/// Ordered straight segments: edge key i connects vertex i to the next vertex, wrapping at the end.
#[derive(Clone, Debug, PartialEq)]
pub struct PrismProbeLoop {
    pub edge_keys: Vec<u64>,
    pub vertices: Vec<[f64; 2]>,
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
    pub faces: Vec<PrismFaceProbe>,
}
