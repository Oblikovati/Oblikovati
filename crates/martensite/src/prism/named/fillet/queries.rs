// SPDX-License-Identifier: GPL-2.0-only
//! Exact named topology queries for the current fillet result (#4045).
use super::*;
use martensite_tkgeombase::gprop::GProps;
use martensite_tktopalgo::brep_gprop::BRepGProp;

impl FilletNamedTopology {
    /// Read analytic bounds center for a body; example: `body.body_bounds_center(&history, document, key)?`.
    pub fn body_bounds_center(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<[f64; 3], String> {
        let body = self.bindings.bind(history, document, key, 4)?;
        super::super::bounds::shape_bounds_center(body)
    }

    /// Read analytic bounds center for a face; example: `body.face_bounds_center(&history, document, key)?`.
    pub fn face_bounds_center(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<[f64; 3], String> {
        let face = self.bindings.bind(history, document, key, 1)?;
        super::super::bounds::shape_bounds_center(face)
    }
    /// Read analytic face type and bounds center; example: `body.face_topology(&history, document, key)?`.
    pub fn face_topology(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<PrismFaceTopology, String> {
        let face = self.bindings.bind(history, document, key, 1)?;
        super::super::face_topology::face_topology_from_shape(face)
    }

    /// Count exact loops on a named face; example: `body.face_loop_count(&history, document, key)?`.
    pub fn face_loop_count(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<usize, String> {
        use martensite_tkbrep::{topexp, toptools::IndexedMapOfShape};
        use martensite_tkg3d::top_abs::ShapeEnum;
        let face = self.bindings.bind(history, document, key, 1)?;
        let mut wires = IndexedMapOfShape::default();
        topexp::map_shapes_of_type(face, ShapeEnum::Wire, &mut wires)
            .map_err(|cause| cause.to_string())?;
        Ok(wires.extent())
    }
    pub(in crate::prism::named) fn keys_of_kinds(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        kinds: &[u32],
    ) -> Result<Vec<Vec<u8>>, String> {
        let mut result = Vec::new();
        for key in self.bindings.encoded_keys(history, document)? {
            let kind = ReferenceEnvelope::decode(key)
                .map_err(|cause| cause.to_string())?
                .kind;
            if kinds.contains(&kind) {
                result.push(key.to_vec());
            }
        }
        Ok(result)
    }
    /// Count exact body topology; example: `body.topology_summary(&history, document, key)?`.
    pub fn topology_summary(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<PrismTopologySummary, String> {
        let shape = self.bindings.bind(history, document, key, 4)?;
        super::super::super::report::topology_summary(shape).map_err(|cause| cause.to_string())
    }

    /// Integrate one exact named face; example: `body.face_area(&history, document, key)?`.
    pub fn face_area(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<f64, String> {
        let face = self.bindings.bind(history, document, key, 1)?;
        let mut properties = GProps::new();
        BRepGProp::surface_properties(face, &mut properties, false, false)
            .map_err(|cause| cause.to_string())?;
        let area = properties.mass();
        if !area.is_finite() || area <= 0. {
            return Err(format!(
                "face {key:?}, area {area}: expected finite positive area"
            ));
        }
        Ok(area)
    }

    /// Integrate one exact named edge; example: `body.edge_length(&history, document, key)?`.
    pub fn edge_length(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<f64, String> {
        let edge = self.bindings.bind(history, document, key, 2)?;
        super::super::subentities::analytic_edge_length(edge)
    }

    /// Read an analytic bounds center for an edge or vertex; example: `body.subentity_point(&history, document, key)?`.
    pub fn subentity_point(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<[f64; 3], String> {
        let kind = ReferenceEnvelope::decode(key)
            .map_err(|cause| cause.to_string())?
            .kind;
        if !matches!(kind, 2 | 3) {
            return Err(format!(
                "subentity key {key:?}, kind {kind}: expected edge 2 or vertex 3"
            ));
        }
        let entity = self.bindings.bind(history, document, key, kind)?;
        super::super::super::provenance::representative(entity, "fillet subentity")
            .map_err(|cause| cause.to_string())
    }
}
