// SPDX-License-Identifier: GPL-2.0-only
//! Assign names from exact history and named incident entities (#4045).
use super::*;
use martensite_tkbrep::{topexp, toptools::IndexedMapOfShape};
use martensite_tkg3d::top_abs::ShapeEnum;
use oblikovati_model::identity::{DerivedReferenceName, ExactReferenceBindings};
use std::collections::{BTreeMap, BTreeSet};

/// Exact live, split and deleted names for a validated candidate result.
#[derive(Clone)]
pub struct BlendNamedTopology {
    resolution: Option<PrismResolution>,
    achieved_tolerance: Option<PrismAchievedTolerance>,
    live: Vec<Vec<u8>>,
    deleted: Vec<Vec<u8>>,
    splits: SplitTopologyNames,
    pub(super) bindings: ExactReferenceBindings<Shape>,
    active_body: Vec<u8>,
    primary_body: Vec<u8>,
    shape: Shape,
    thread_overlays: Vec<crate::DisplayTriangle>,
    mark_overlays: Vec<crate::DisplayTriangle>,
}

mod capture;
mod coil;
mod derived;
mod extrusion_boolean;
mod hole;
mod loft;
mod result_names;
mod revolve;
mod rigid_motion;
mod section_body;
mod shell;
mod source;
mod sweep;
pub(in crate::prism::named) use capture::CapturedBlendTopology;

impl BlendNamedTopology {
    pub(in crate::prism::named) fn is_surface(&self) -> bool {
        matches!(self.shape.shape_type(), Ok(ShapeEnum::Shell))
    }

    pub(crate) fn resolution(&self) -> Option<PrismResolution> {
        self.resolution
    }
    pub(crate) fn thread_overlay_triangles(&self) -> &[crate::DisplayTriangle] {
        &self.thread_overlays
    }
    pub(crate) fn set_thread_overlays(&mut self, overlays: Vec<crate::DisplayTriangle>) {
        self.thread_overlays = overlays;
    }
    pub(crate) fn mark_overlay_triangles(&self) -> &[crate::DisplayTriangle] {
        &self.mark_overlays
    }
    pub(crate) fn set_mark_overlays(&mut self, overlays: Vec<crate::DisplayTriangle>) {
        self.mark_overlays = overlays;
    }
    pub(in crate::prism::named) fn capture_current(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Self, String> {
        Ok(Self {
            resolution: self.resolution,
            achieved_tolerance: self.achieved_tolerance,
            live: self.live.clone(),
            deleted: self.deleted.clone(),
            splits: self.splits.clone(),
            bindings: self.bindings.clone_current(history, document)?,
            active_body: self.active_body.clone(),
            primary_body: self.primary_body.clone(),
            shape: self.shape.clone(),
            thread_overlays: self.thread_overlays.clone(),
            mark_overlays: self.mark_overlays.clone(),
        })
    }
    pub(in crate::prism::named) fn native_edge<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<&'a Shape, String> {
        self.bindings.bind(history, document, key, 2)
    }
    pub(in crate::prism::named) fn native_face<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<&'a Shape, String> {
        self.bindings.bind(history, document, key, 1)
    }
    pub(in crate::prism::named) fn native_shape<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<&'a Shape, String> {
        self.bindings.bind(history, document, &self.active_body, 4)
    }
    /// Resolve the current body's primary key; example: `named.body_reference_key(&history, document)?`.
    pub fn body_reference_key<'a>(
        &'a self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<&'a [u8], String> {
        self.bindings
            .bind(history, document, &self.primary_body, 4)?;
        Ok(&self.primary_body)
    }

    /// Integrate analytic properties of a named body; example: `named.physical_properties(&history, document, &body_key)?`.
    pub fn physical_properties(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<crate::AnalyticBodyProperties, String> {
        let body = self.bindings.bind(history, document, key, 4)?;
        crate::analytic_body::integrate_body(body)
    }
    /// Tessellate a detached copy for display; example: `named.display_triangles(options)?`.
    pub fn display_triangles(
        &self,
        options: PrismMeshOptions,
    ) -> Result<Vec<[[f64; 3]; 3]>, String> {
        Ok(self
            .display_face_triangles(options)?
            .into_iter()
            .flatten()
            .collect())
    }

    pub(crate) fn display_face_triangles(
        &self,
        options: PrismMeshOptions,
    ) -> Result<Vec<Vec<[[f64; 3]; 3]>>, String> {
        super::super::super::mesh::body_face_triangles(&self.shape, options)
    }

    pub(crate) fn display_face_shaded(
        &self,
        options: PrismMeshOptions,
    ) -> Result<Vec<Vec<crate::DisplayTriangle>>, String> {
        super::super::super::mesh::body_face_display_triangles(&self.shape, options)
    }
    /// Integrate the current result; example: `named.volume(&history, document)?`.
    pub fn volume(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<f64, String> {
        let body = self
            .bindings
            .bind(history, document, &self.active_body, 4)?;
        crate::checked_shape::analytic_properties(body, "named blend result")
            .map(|properties| properties.volume)
            .map_err(|cause| cause.to_string())
    }
    /// Read canonical live names; example: `named.live_keys()`.
    pub fn live_keys(&self) -> &[Vec<u8>] {
        &self.live
    }
    /// Read names of deleted inputs; example: `named.deleted_keys()`.
    pub fn deleted_keys(&self) -> &[Vec<u8>] {
        &self.deleted
    }
    /// Read split input names and their immediate descendants.
    /// Example: `named.split_keys().get(&old_key)` after an overlapping Join.
    pub fn split_keys(&self) -> &BTreeMap<Vec<u8>, Vec<Vec<u8>>> {
        &self.splits
    }
    /// Check an exact live name in the owning evaluation; example: `named.contains(&history, doc, &key, 1)?`.
    pub fn contains(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
        kind: u32,
    ) -> Result<bool, String> {
        self.bindings
            .bind(history, document, key, kind)
            .map(|_| true)
    }

    /// Resolve a current planar face for a later feature; example: `named.face_plane(&history, doc, &key)?`.
    pub fn face_plane(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        let body = self
            .bindings
            .bind(history, document, &self.active_body, 4)?;
        let face = self.bindings.bind(history, document, key, 1)?;
        super::super::face_topology::planar_shape_frame(body, face)
    }

    /// Construct another detached fillet from this current named result.
    /// Example: `named.fillet_candidate(&history, document, &[edge], 0.1)?`.
    pub fn fillet_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
        radius: f64,
    ) -> Result<BlendCandidate, String> {
        self.blend_candidate(
            history,
            document,
            edges,
            super::BlendOperation::Fillet(radius),
        )
    }

    /// Construct a detached chamfer from this current named result.
    /// Example: `named.chamfer_candidate(&history, document, &[edge], 0.1)?`.
    pub fn chamfer_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
        distance: f64,
    ) -> Result<BlendCandidate, String> {
        self.blend_candidate(
            history,
            document,
            edges,
            super::BlendOperation::Chamfer(distance),
        )
    }

    fn blend_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
        operation: super::BlendOperation,
    ) -> Result<BlendCandidate, String> {
        let input = self.source_topology(history, document)?;
        super::build_candidate(
            super::BlendInputTopology {
                body: &input.body,
                source: input.entities,
                source_bodies: input.body_keys,
                body_parent: self.active_body.clone(),
                primary_body: self.primary_body.clone(),
                inherited_deleted: self.deleted.clone(),
                inherited_splits: self.splits.clone(),
                resolution: self.resolution,
            },
            history,
            document,
            edges,
            operation,
        )
    }
}

impl BlendCandidate {
    /// Name result topology from kernel history and exact incidence.
    /// Example: `candidate.name_topology(&history, document, feature)?`.
    pub fn name_topology(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
    ) -> Result<BlendNamedTopology, String> {
        if document != self.document || !self.scope.is_current(history) {
            return Err(format!(
                "blend document {document}, feature {feature:?}: expected the current source evaluation in {}",
                self.document
            ));
        }
        let names = self.result_names(document, feature)?;
        self.bind_topology(history, document, feature, names)
    }

    fn bind_topology(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        feature: FeatureId,
        names: result_names::ResultNames,
    ) -> Result<BlendNamedTopology, String> {
        let body = DerivedReferenceName::new(
            document,
            feature.0,
            4,
            "body",
            std::slice::from_ref(&self.body_parent),
        )?
        .encode()?;
        let mut entries: Vec<_> = self
            .source_bodies
            .iter()
            .filter(|key| *key != &body)
            .map(|key| (key.clone(), self.shape.clone()))
            .collect();
        entries.push((body.clone(), self.shape.clone()));
        entries.extend(names.entries);
        let live: Vec<_> = entries.iter().map(|(key, _)| key.clone()).collect();
        let deleted = self.deleted_inputs(&live, &names.splits)?;
        let mut splits = self.inherited_splits.clone();
        splits.extend(names.splits);
        let expected = entries.clone();
        let bindings = ExactReferenceBindings::from_entries(document, self.scope.clone(), entries)?;
        verify_round_trip(&bindings, history, document, &expected)?;
        let live = bindings
            .encoded_keys(history, document)?
            .map(<[u8]>::to_vec)
            .collect();
        Ok(BlendNamedTopology {
            resolution: self.resolution,
            achieved_tolerance: self.achieved_tolerance,
            live,
            deleted,
            splits,
            bindings,
            active_body: body,
            primary_body: self.primary_body.clone(),
            shape: self.shape.clone(),
            thread_overlays: Vec::new(),
            mark_overlays: Vec::new(),
        })
    }

    fn deleted_inputs(
        &self,
        live: &[Vec<u8>],
        splits: &SplitTopologyNames,
    ) -> Result<Vec<Vec<u8>>, String> {
        let live: BTreeSet<_> = live.iter().collect();
        let mut deleted: BTreeSet<_> = self.inherited_deleted.iter().cloned().collect();
        for record in &self.history {
            if record.transient {
                continue;
            }
            if live.contains(&record.name) || splits.contains_key(&record.name) {
                continue;
            }
            if !record.deleted {
                return Err(format!(
                    "blend input {:?}: expected a preserved name, a split, or kernel deletion",
                    record.name
                ));
            }
            deleted.insert(record.name.clone());
        }
        Ok(deleted.into_iter().collect())
    }
}

fn shapes(shape: &Shape, kind: ShapeEnum) -> Result<Vec<Shape>, String> {
    let mut indexed = IndexedMapOfShape::default();
    topexp::map_shapes_of_type(shape, kind, &mut indexed).map_err(|cause| cause.to_string())?;
    (1..=indexed.extent())
        .map(|index| {
            indexed
                .find_key(index)
                .cloned()
                .map_err(|cause| cause.to_string())
        })
        .collect()
}

fn direct_name(
    history: &[BlendHistoryRecord],
    output: &Shape,
    kind: u32,
    document: DocumentGuid,
    feature: FeatureId,
) -> Result<Option<Vec<u8>>, String> {
    let mut retained = Vec::new();
    let mut generated = Vec::new();
    for record in history {
        let source_kind = ReferenceEnvelope::decode(&record.name)
            .map_err(|cause| cause.to_string())?
            .kind;
        if source_kind == kind
            && (record.source.is_same(output)
                || record.modified.iter().any(|shape| shape.is_same(output)))
        {
            retained.push(record);
        }
        if record.generated.iter().any(|shape| shape.is_same(output)) {
            generated.push(record.name.clone());
        }
    }
    // The temporary tool has no prior owner references. A sole saved name
    // can survive this merge, subject to uniqueness across result entities.
    if generated.is_empty() && retained.len() > 1 {
        let mut persistent = retained.iter().filter(|record| !record.transient);
        if let (Some(target), None) = (persistent.next(), persistent.next()) {
            return Ok(Some(target.name.clone()));
        }
    }
    match (retained.len(), generated.is_empty()) {
        (1, true) => Ok(retained.pop().map(|record| record.name.clone())),
        (0, false) => {
            DerivedReferenceName::new(document, feature.0, kind, "generated", &generated)?
                .encode()
                .map(Some)
        }
        (0, true) => Ok(None),
        _ => Err(format!(
            "blend result kind {kind}, retained {}, generated {}: expected one unambiguous history",
            retained.len(),
            generated.len()
        )),
    }
}

fn verify_round_trip(
    bindings: &ExactReferenceBindings<Shape>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    expected: &[(Vec<u8>, Shape)],
) -> Result<(), String> {
    for (key, shape) in expected {
        let kind = ReferenceEnvelope::decode(key)
            .map_err(|cause| cause.to_string())?
            .kind;
        if !bindings.bind(history, document, key, kind)?.is_same(shape) {
            return Err(format!(
                "blend name {key:?}: expected exact entity round trip"
            ));
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::identity::SemanticReferenceName;

    #[test]
    fn ambiguous_direct_history_is_refused() {
        let document: DocumentGuid = "11111111-2222-3333-4444-555555555555".parse().unwrap();
        let shape = martensite_tkprim::brep_prim_api::BRepPrimAPIMakeBox::from_sizes(1., 1., 1.)
            .unwrap()
            .shape()
            .unwrap()
            .clone();
        let face = shapes(&shape, ShapeEnum::Face).unwrap().remove(0);
        let record = |id| BlendHistoryRecord {
            name: SemanticReferenceName::new(document, id, 1, "fixture", "face", &[1])
                .unwrap()
                .encode()
                .unwrap(),
            source: face.clone(),
            generated: Vec::new(),
            modified: Vec::new(),
            deleted: false,
            transient: false,
        };
        let target = record(1);
        let expected = target.name.clone();
        let tool = BlendHistoryRecord {
            transient: true,
            ..record(2)
        };
        assert_eq!(
            direct_name(&[target, tool], &face, 1, document, FeatureId(3)).unwrap(),
            Some(expected),
        );
        assert!(
            direct_name(&[record(1), record(2)], &face, 1, document, FeatureId(3))
                .unwrap_err()
                .contains("unambiguous")
        );
    }
}
