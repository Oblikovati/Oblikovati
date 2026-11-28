// SPDX-License-Identifier: GPL-2.0-only
//! Surface boundary edge chaining and shell assembly for Extend Surface (#3918).
use super::extend_surface_geom::{edge_endpoints, extend_planar_face, points_near};
use super::thicken_offset_chain::collect_shapes;
use super::*;
use martensite_tkbrep::topods::{Builder, Shape, Shell};
use martensite_tkg3d::top_abs::ShapeEnum;
use oblikovati_model::feature::{FeatureId, extend_surface::ResolvedExtendSurface};
use std::collections::{BTreeSet, VecDeque};

impl NamedSolid {
    /// Compute total analytic surface area of this body in `mm²`.
    /// Example: `surface_body.surface_area_mm2(&history, document)?`.
    pub fn surface_area_mm2(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<f64, String> {
        let mut cm2 = 0.0_f64;
        for key in self.face_reference_keys(history, document)? {
            cm2 += self.face_area(history, document, &key)?;
        }
        Ok(cm2 * 100.0)
    }

    /// Resolve a free boundary edge on a surface body, optionally chaining connected boundary edges.
    /// Example: `body.surface_boundary_edge_chain(&history, document, &edge, true)?`.
    pub fn surface_boundary_edge_chain(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        seed_edge: &[u8],
        edge_chain: bool,
    ) -> Result<Vec<Vec<u8>>, String> {
        if !self.is_surface() {
            return Err("extend surface: expected an open surface body".into());
        }
        let all_keys = self.edge_reference_keys(history, document)?;
        let face_keys = self.face_reference_keys(history, document)?;
        let face_edges: Vec<Vec<Shape>> = face_keys
            .iter()
            .map(|fk| {
                let face = self.native_face(history, document, fk)?;
                collect_shapes(face, ShapeEnum::Edge)
            })
            .collect::<Result<_, _>>()?;
        let mut boundary_keys = Vec::new();
        let mut boundary_ends = Vec::new();
        for key in &all_keys {
            let edge = self.native_edge_shape(history, document, key)?;
            let incident_faces = face_edges
                .iter()
                .filter(|edges| edges.iter().any(|candidate| candidate.is_same(edge)))
                .count();
            if incident_faces <= 1 {
                boundary_keys.push(key.clone());
                boundary_ends.push(edge_endpoints(edge)?);
            }
        }
        let Some(seed_idx) = boundary_keys
            .iter()
            .position(|key| key.as_slice() == seed_edge)
        else {
            return Err(
                "extend surface edge: expected a free boundary edge on the surface body".into(),
            );
        };
        if !edge_chain {
            return Ok(vec![boundary_keys[seed_idx].clone()]);
        }
        let mut visited = BTreeSet::new();
        let mut order = Vec::new();
        let mut queue = VecDeque::new();
        visited.insert(seed_idx);
        order.push(seed_idx);
        queue.push_back(seed_idx);
        while let Some(current) = queue.pop_front() {
            let [a0, a1] = boundary_ends[current];
            for (other, [b0, b1]) in boundary_ends.iter().enumerate() {
                if visited.contains(&other) {
                    continue;
                }
                if points_near(a0, *b0)
                    || points_near(a0, *b1)
                    || points_near(a1, *b0)
                    || points_near(a1, *b1)
                {
                    visited.insert(other);
                    order.push(other);
                    queue.push_back(other);
                }
            }
        }
        Ok(order
            .into_iter()
            .map(|idx| boundary_keys[idx].clone())
            .collect())
    }

    pub(in crate::prism::named) fn evaluate_extend_surface_result(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        input: &ResolvedExtendSurface,
    ) -> Result<NamedSolid, String> {
        if !self.is_surface() {
            return Err("extend surface: expected an open surface body".into());
        }
        let effective_keys = self.expand_extend_edge_keys(history, document, input)?;
        let selected_shapes: Vec<Shape> = effective_keys
            .iter()
            .map(|key| self.native_edge_shape(history, document, key).cloned())
            .collect::<Result<_, _>>()?;

        let mut shell = Shell::default();
        let builder = Builder;
        builder.make_shell(&mut shell);
        let mut shell_shape = shell.into_shape();
        let mut extended_any = false;

        for face_key in self.face_reference_keys(history, document)? {
            let face = self.native_face(history, document, &face_key)?;
            let edges_on_face = collect_shapes(face, ShapeEnum::Edge)?;
            let touches_selected = edges_on_face
                .iter()
                .any(|edge| selected_shapes.iter().any(|sel| sel.is_same(edge)));
            if !touches_selected {
                builder
                    .add(&mut shell_shape, face)
                    .map_err(|cause| cause.to_string())?;
                continue;
            }
            let new_faces =
                extend_planar_face(face, &selected_shapes, &input.extent, input.method)?;
            for built in new_faces {
                builder
                    .add(&mut shell_shape, &built)
                    .map_err(|cause| cause.to_string())?;
            }
            extended_any = true;
        }
        if !extended_any {
            return Err("extend surface: selected edges do not belong to any surface face".into());
        }
        let primary_body = self.body_reference_key(history, document)?.to_vec();
        let deleted = self.extend_prior_non_body_keys(history, document, &primary_body)?;
        let resolution = match self {
            Self::Prism(body) => body.certification.as_ref().map(|c| c.resolution),
            Self::Derived(body) => body.resolution(),
        };
        let topology = BlendNamedTopology::from_extended_shell_preserving_body(
            history,
            document,
            feature,
            primary_body,
            deleted,
            shell_shape,
            resolution,
        )?;
        let result = NamedSolid::Derived(Box::new(topology));
        let before_area = self.surface_area_mm2(history, document)?;
        let after_area = result.surface_area_mm2(history, document)?;
        if !after_area.is_finite() || after_area <= before_area {
            return Err(format!(
                "extend surface area {after_area:.4} mm^2 <= {before_area:.4} mm^2: expected an enlarged surface"
            ));
        }
        Ok(result)
    }

    fn expand_extend_edge_keys(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        input: &ResolvedExtendSurface,
    ) -> Result<Vec<Vec<u8>>, String> {
        let mut out = Vec::new();
        let mut seen = BTreeSet::new();
        for key in &input.edges {
            let group =
                self.surface_boundary_edge_chain(history, document, key, input.edge_chain)?;
            for member in group {
                if seen.insert(member.clone()) {
                    out.push(member);
                }
            }
        }
        if out.is_empty() {
            return Err("extend surface: expected at least one valid boundary edge".into());
        }
        Ok(out)
    }

    fn native_edge_shape<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<&'a Shape, String> {
        match self {
            Self::Prism(body) => body
                .subentity_bindings
                .as_ref()
                .ok_or("extend surface edge: expected certified subentities")?
                .bind(history, document, key, 2)
                .map(|entry| &entry.shape),
            Self::Derived(body) => body.native_edge(history, document, key),
        }
    }

    fn extend_prior_non_body_keys(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        primary_body: &[u8],
    ) -> Result<Vec<Vec<u8>>, String> {
        let mut keys = self.reference_keys(history, document)?;
        keys.extend(self.subentity_reference_keys(history, document)?);
        if let Self::Derived(body) = self {
            keys.extend(body.live_keys().iter().cloned());
            keys.extend(body.deleted_keys().iter().cloned());
        }
        keys.retain(|k| k.as_slice() != primary_body);
        keys.sort();
        keys.dedup();
        Ok(keys)
    }
}
