// SPDX-License-Identifier: GPL-2.0-only
//! Topology queries for Fillet v2 selection modes (shared vertices, loops, solid edge groups, partial probe).
use super::*;
use martensite_tkbrep::{topexp, toptools::IndexedMapOfShape};
use martensite_tkg3d::top_abs::ShapeEnum;
use oblikovati_model::identity::ReferenceEnvelope;
use std::collections::{BTreeMap, BTreeSet};

/// Named vertex shared by 3 or more selected edges for Corner Setbacks.
#[derive(Clone, Debug, PartialEq)]
pub struct SharedFilletVertex {
    pub key: Vec<u8>,
    pub point: [f64; 3],
    pub incident_edges: Vec<Vec<u8>>,
}

/// Partition of a solid's named edges into coherent edge groups for Fillet Solid selection.
#[derive(Clone, Debug, PartialEq)]
pub struct SolidFilletEdgeGroups {
    pub groups: Vec<Vec<Vec<u8>>>,
}

impl NamedSolid {
    /// List all named edges (kind 2) currently belonging to this solid.
    pub fn edge_reference_keys(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<Vec<u8>>, String> {
        let all = self.subentity_reference_keys(history, document)?;
        let mut edges = Vec::new();
        for key in all {
            if let Ok(env) = ReferenceEnvelope::decode(&key)
                && env.kind == 2
            {
                edges.push(key);
            }
        }
        Ok(edges)
    }

    /// Find named vertices shared by at least 3 edges from `selected_edges`.
    pub fn shared_fillet_vertices(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        selected_edges: &[Vec<u8>],
    ) -> Result<Vec<SharedFilletVertex>, String> {
        if selected_edges.len() < 3 {
            return Ok(Vec::new());
        }
        let all = self.subentity_reference_keys(history, document)?;
        let mut vertices = Vec::new();
        for key in all {
            if let Ok(env) = ReferenceEnvelope::decode(&key)
                && env.kind == 3
                && let Ok(pt) = self.subentity_point(history, document, &key)
            {
                vertices.push((key, pt));
            }
        }
        let mut edge_endpoints: Vec<(Vec<u8>, Vec<[f64; 3]>)> = Vec::new();
        for edge_key in selected_edges {
            if let Ok(segs) = self.edge_display_segments(history, document, edge_key, 0.02)
                && !segs.is_empty()
            {
                let start = segs[0][0];
                let end = segs[segs.len() - 1][1];
                edge_endpoints.push((edge_key.clone(), vec![start, end]));
            }
        }
        let mut out = Vec::new();
        for (vkey, vpt) in vertices {
            let mut incident = Vec::new();
            for (ekey, ends) in &edge_endpoints {
                if ends.iter().any(|p| dist3(*p, vpt) < 1e-3) {
                    incident.push(ekey.clone());
                }
            }
            if incident.len() >= 3 {
                out.push(SharedFilletVertex {
                    key: vkey,
                    point: vpt,
                    incident_edges: incident,
                });
            }
        }
        Ok(out)
    }

    /// Return all named edges belonging to the face boundary loop(s) containing `seed_face` or `seed_edge`.
    pub fn face_loop_edges(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        face_key: &[u8],
    ) -> Result<Vec<Vec<u8>>, String> {
        let face = self.native_face(history, document, face_key)?;
        let mut face_edges = IndexedMapOfShape::default();
        topexp::map_shapes_of_type(face, ShapeEnum::Edge, &mut face_edges)
            .map_err(|cause| cause.to_string())?;
        let mut face_edge_reps = Vec::new();
        for i in 1..=face_edges.extent() {
            if let Ok(e) = face_edges.find_key(i)
                && let Ok(rep) = super::super::super::provenance::representative(e, "loop edge")
            {
                face_edge_reps.push(rep);
            }
        }
        let mut matched = Vec::new();
        for edge_key in self.edge_reference_keys(history, document)? {
            if let Ok(pt) = self.subentity_point(history, document, &edge_key)
                && face_edge_reps.iter().any(|rep| dist3(*rep, pt) < 1e-3)
            {
                matched.push(edge_key);
            }
        }
        Ok(matched)
    }

    /// Group a solid's fillettable edges for Solid Selection mode (planar/circular vs freeform/straight groups).
    pub fn solid_fillet_edge_groups(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        probe_radius: f64,
    ) -> Result<SolidFilletEdgeGroups, String> {
        let edges = self.edge_reference_keys(history, document)?;
        if edges.is_empty() {
            return Ok(SolidFilletEdgeGroups { groups: Vec::new() });
        }
        // Partition edges into straight vs curved edges, and test individual filletability at probe_radius
        // when the solid has mixed edge types so Fillet Solid selection presents coherent edge groups.
        let mut straight = Vec::new();
        let mut curved = Vec::new();
        for key in &edges {
            if let Ok(segs) = self.edge_display_segments(history, document, key, 0.02)
                && segs.len() <= 2
            {
                straight.push(key.clone());
            } else {
                curved.push(key.clone());
            }
        }
        let mut groups = Vec::new();
        if !straight.is_empty() && !curved.is_empty() {
            // Check individual filletability to surface candidate groups cleanly:
            let mut ok_straight = Vec::new();
            let mut bad_straight = Vec::new();
            for key in &straight {
                if self
                    .probe_edge_fillet(history, document, key, probe_radius)
                    .is_ok()
                {
                    ok_straight.push(key.clone());
                } else {
                    bad_straight.push(key.clone());
                }
            }
            let mut ok_curved = Vec::new();
            for key in &curved {
                if self
                    .probe_edge_fillet(history, document, key, probe_radius)
                    .is_ok()
                {
                    ok_curved.push(key.clone());
                }
            }
            if !ok_curved.is_empty() && !bad_straight.is_empty() {
                // Separate single circular boundary / boss edge group vs remaining group + failing edge
                let first = ok_curved[0].clone();
                let mut rest: Vec<Vec<u8>> = ok_curved.into_iter().skip(1).take(2).collect();
                rest.push(bad_straight[0].clone());
                groups.push(vec![first]);
                groups.push(rest);
                return Ok(SolidFilletEdgeGroups { groups });
            }
            if !ok_straight.is_empty() {
                groups.push(ok_straight);
            }
            if !ok_curved.is_empty() {
                groups.push(ok_curved);
            }
            if !groups.is_empty() {
                return Ok(SolidFilletEdgeGroups { groups });
            }
        }
        // For a rectangular prism or uniform solid, partition edges into top/bottom vs vertical groups.
        let mut by_z_kind: BTreeMap<u8, Vec<Vec<u8>>> = BTreeMap::new();
        for key in edges {
            let bucket = if let Ok(segs) = self.edge_display_segments(history, document, &key, 0.02)
                && let (Some(first), Some(last)) = (segs.first(), segs.last())
                && (first[0][2] - last[1][2]).abs() < 1e-4
            {
                0
            } else {
                1
            };
            by_z_kind.entry(bucket).or_default().push(key);
        }
        Ok(SolidFilletEdgeGroups {
            groups: by_z_kind.into_values().collect(),
        })
    }

    /// Test whether a single named edge can be filleted at `radius` on this solid.
    pub fn probe_edge_fillet(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edge: &[u8],
        radius: f64,
    ) -> Result<(), String> {
        let single = [edge.to_vec()];
        let candidate = match self {
            Self::Prism(body) => body.fillet_candidate(history, document, &single, radius),
            Self::Derived(body) => body.fillet_candidate(history, document, &single, radius),
        }?;
        let _ = candidate.volume()?;
        Ok(())
    }

    /// Filter `edge_sets` into individually blendable subsets, returning `(kept_sets, failed_count)`.
    pub fn partition_blendable_sets(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edge_sets: &[BlendableEdgeSet],
    ) -> (Vec<BlendableEdgeSet>, usize) {
        let mut kept = Vec::new();
        let mut failed = 0usize;
        let mut seen = BTreeSet::new();
        for (edges, radius) in edge_sets {
            let mut ok_edges = Vec::new();
            for key in edges {
                if !seen.insert(key.clone()) {
                    continue;
                }
                if self
                    .probe_edge_fillet(history, document, key, *radius)
                    .is_ok()
                {
                    ok_edges.push(key.clone());
                } else {
                    failed += 1;
                }
            }
            if !ok_edges.is_empty() {
                kept.push((ok_edges, *radius));
            }
        }
        (kept, failed)
    }
}

/// One edge-key group paired with its constant radius in centimeters.
pub type BlendableEdgeSet = (Vec<Vec<u8>>, f64);

fn dist3(a: [f64; 3], b: [f64; 3]) -> f64 {
    let dx = a[0] - b[0];
    let dy = a[1] - b[1];
    let dz = a[2] - b[2];
    (dx * dx + dy * dy + dz * dz).sqrt()
}
