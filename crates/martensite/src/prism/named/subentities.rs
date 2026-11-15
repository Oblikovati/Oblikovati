// SPDX-License-Identifier: GPL-2.0-only
//! Exact document-scoped names for copied extrusion edges and vertices.
use super::*;
use martensite_tkbrep::{brep::BRepTool, topods};
use martensite_tkgeombase::gprop::GProps;
use martensite_tktopalgo::brep_gprop::BRepGProp;
use oblikovati_model::identity::{ReferenceEnvelope, SemanticReferenceName};

pub(super) struct BoundSubentity {
    provenance: PrismSubentityProvenance,
    shape: Shape,
}
impl NamedPrismSnapshot {
    /// Measure the current named edge using its analytic curve, in database centimeters.
    /// Example: `body.edge_length(history, document, &key)?`.
    pub fn edge_length(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<f64, String> {
        let bindings = self
            .subentity_bindings
            .as_ref()
            .ok_or("prism subentities: expected certified copied topology for edge measurement")?;
        let bound = bindings.bind(history, document, key, 2)?;
        analytic_edge_length(&bound.shape)
    }
    pub(super) fn bind_subentities(
        &mut self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        sketch: u64,
        lineage: &PrismSubentityHistory,
        shapes: Vec<Shape>,
    ) -> Result<(), String> {
        let records: Vec<_> = lineage.edges.iter().chain(&lineage.vertices).collect();
        if records.len() != shapes.len() {
            return Err(format!(
                "prism subentity records={}, shapes={}: expected one copied shape per record",
                records.len(),
                shapes.len()
            ));
        }
        let entries = records
            .into_iter()
            .zip(shapes)
            .map(|(record, shape)| {
                Ok((
                    subentity_key(document, feature, sketch, record)?,
                    BoundSubentity {
                        provenance: record.clone(),
                        shape,
                    },
                ))
            })
            .collect::<Result<Vec<_>, String>>()?;
        let expected: Vec<_> = entries
            .iter()
            .map(|(key, entity)| (key.clone(), entity.shape.clone()))
            .collect();
        let bindings =
            ExactReferenceBindings::from_entries(document, history.evaluation_scope(), entries)?;
        verify_subentity_bindings(&bindings, history, document, &expected)?;
        self.subentity_bindings = Some(bindings);
        Ok(())
    }
    /// Enumerate exact edge and vertex keys for a current certified result.
    /// Example: `body.subentity_reference_keys(history, document)?`.
    pub fn subentity_reference_keys(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<Vec<u8>>, String> {
        let bindings = self
            .subentity_bindings
            .as_ref()
            .ok_or("prism subentities: expected certified copied topology")?;
        Ok(bindings
            .encoded_keys(history, document)?
            .map(<[u8]>::to_vec)
            .collect())
    }
    /// Resolve a current edge or vertex key; example: `body.subentity_provenance(history, document, &key)?`.
    pub fn subentity_provenance<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<&'a PrismSubentityProvenance, String> {
        let bindings = self
            .subentity_bindings
            .as_ref()
            .ok_or("prism subentities: expected certified copied topology")?;
        let kind = ReferenceEnvelope::decode(key)
            .map_err(|cause| cause.to_string())?
            .kind;
        if !matches!(kind, 2 | 3) {
            return Err(format!(
                "prism subentity key kind {kind}: expected edge 2 or vertex 3"
            ));
        }
        Ok(&bindings.bind(history, document, key, kind)?.provenance)
    }
}

fn analytic_edge_length(shape: &Shape) -> Result<f64, String> {
    let edge = topods::edge(shape).map_err(|cause| cause.to_string())?;
    if !BRepTool::is_geometric_edge(&edge).map_err(|cause| cause.to_string())? {
        return Err("edge: expected an analytic curve for length measurement".into());
    }
    let mut properties = GProps::default();
    BRepGProp::linear_properties(shape, &mut properties, false, false)
        .map_err(|cause| cause.to_string())?;
    let length = properties.mass();
    if !length.is_finite() || length <= 0. {
        return Err(format!(
            "edge length {length}: expected finite positive centimeters"
        ));
    }
    Ok(length)
}
fn subentity_key(
    document: DocumentGuid,
    feature: FeatureId,
    sketch: u64,
    record: &PrismSubentityProvenance,
) -> Result<Vec<u8>, String> {
    let (source, source_tag) = match record.source {
        PrismHistorySource::Curve(id) => (id, "curve"),
        PrismHistorySource::Vertex(id) => (id, "vertex"),
    };
    let (kind, role) = match record.role {
        PrismSubentityRole::StartEdge => (2, "start-edge"),
        PrismSubentityRole::EndEdge => (2, "end-edge"),
        PrismSubentityRole::LateralEdge => (2, "lateral-edge"),
        PrismSubentityRole::StartVertex => (3, "start-vertex"),
        PrismSubentityRole::EndVertex => (3, "end-vertex"),
    };
    SemanticReferenceName::new(
        document,
        feature.0,
        kind,
        "sketch-prism",
        &format!("{role}-{source_tag}"),
        &[sketch, source],
    )?
    .encode()
}
fn verify_subentity_bindings(
    bindings: &ExactReferenceBindings<BoundSubentity>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    expected: &[(Vec<u8>, Shape)],
) -> Result<(), String> {
    for (key, shape) in expected {
        let kind = ReferenceEnvelope::decode(key)
            .map_err(|cause| cause.to_string())?
            .kind;
        let bound = bindings.bind(history, document, key, kind)?;
        if !bound.shape.is_same(shape) {
            return Err(format!(
                "prism subentity {key:?}: expected exact copied-shape round trip"
            ));
        }
    }
    Ok(())
}
