// SPDX-License-Identifier: GPL-2.0-only
//! Detached blend construction and exact kernel history (#4045, #4046).
use super::*;
use martensite_tkfillet::{
    brep_fillet_api::{BRepFilletAPIMakeChamfer, BRepFilletAPIMakeFillet},
    chfi3d::FilletShape,
};
use martensite_tktopalgo::brep_builder_api::{BRepBuilderAPICopy, BRepBuilderAPIMakeShape};
use oblikovati_model::identity::ReferenceEnvelope;
mod chamfer_v2;
mod combine;
mod delete_face;
mod direct_edit;
mod draft;
pub(super) mod extrusion_boolean;
mod fillet_topology;
mod fillet_v2;
mod hole;
pub(super) mod mark;
mod rigid_motion;
mod shell;
mod slot;
mod solid_boolean;
mod split;
mod split_tool;
mod thicken_offset;
mod thicken_offset_chain;
pub(super) mod thread;
mod thread_overlay;
mod tolerances;
pub use fillet_topology::{SharedFilletVertex, SolidFilletEdgeGroups};
pub use mark::{MarkEligibility, check_mark_eligibility};
pub use rigid_motion::RigidMotion;
pub use thread::ThreadableFaceInfo;
mod naming;
pub(super) use delete_face::DeleteFaceRequest;
pub(super) use draft::DraftRequest;
pub(super) use hole::{HoleRequest, build_hole_tool};
pub(super) use slot::{SlotRequest, build_slot_tools};
mod queries;
pub use naming::BlendNamedTopology;
pub(super) use naming::CapturedBlendTopology;
type SplitTopologyNames = std::collections::BTreeMap<Vec<u8>, Vec<Vec<u8>>>;

/// A validated detached result and its input history, before assigning result names.
pub struct BlendCandidate {
    resolution: Option<PrismResolution>,
    achieved_tolerance: Option<PrismAchievedTolerance>,
    shape: Shape,
    history: Vec<BlendHistoryRecord>,
    source_bodies: Vec<Vec<u8>>,
    body_parent: Vec<u8>,
    primary_body: Vec<u8>,
    inherited_deleted: Vec<Vec<u8>>,
    inherited_splits: SplitTopologyNames,
    document: DocumentGuid,
    scope: EvaluationScope,
}

/// The kernel history for one exact named input entity.
pub struct BlendHistoryRecord {
    name: Vec<u8>,
    source: Shape,
    generated: Vec<Shape>,
    modified: Vec<Shape>,
    deleted: bool,
    transient: bool,
}

impl BlendCandidate {
    pub(in crate::prism::named) fn with_body_parent(mut self, key: &[u8]) -> Self {
        self.body_parent = key.to_vec();
        self
    }
    /// Read input records in name order; example: `candidate.history()`.
    pub fn history(&self) -> &[BlendHistoryRecord] {
        &self.history
    }
    /// Integrate the detached result analytically; example: `candidate.volume()?`.
    pub fn volume(&self) -> Result<f64, String> {
        crate::checked_shape::analytic_properties(&self.shape, "fillet candidate")
            .map(|properties| properties.volume)
            .map_err(|cause| cause.to_string())
    }
}

impl BlendHistoryRecord {
    /// Read the input name; example: `record.name()`.
    pub fn name(&self) -> &[u8] {
        &self.name
    }
    /// Count generated entities; example: `record.generated_count()`.
    pub fn generated_count(&self) -> usize {
        self.generated.len()
    }
    /// Count modified entities; example: `record.modified_count()`.
    pub fn modified_count(&self) -> usize {
        self.modified.len()
    }
    /// Report kernel deletion; example: `record.is_deleted()`.
    pub fn is_deleted(&self) -> bool {
        self.deleted
    }
}

impl NamedPrismSnapshot {
    /// Construct a detached candidate from exact current edge names.
    /// Example: `body.fillet_candidate(&history, document, &[edge_key], 0.1)?`.
    pub fn fillet_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
        radius: f64,
    ) -> Result<BlendCandidate, String> {
        self.blend_candidate(history, document, edges, BlendOperation::Fillet(radius))
    }

    /// Construct a detached equal-distance chamfer from current named edges.
    /// Example: `body.chamfer_candidate(&history, document, &[edge_key], 0.1)?`.
    pub fn chamfer_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
        distance: f64,
    ) -> Result<BlendCandidate, String> {
        self.blend_candidate(history, document, edges, BlendOperation::Chamfer(distance))
    }

    fn blend_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        edges: &[Vec<u8>],
        operation: BlendOperation,
    ) -> Result<BlendCandidate, String> {
        validate_inputs(edges, operation.distance())?;
        let source = self.blend_sources(history, document)?;
        let source_body = self.body_reference_key(history, document)?.to_vec();
        build_candidate(
            BlendInputTopology {
                body: &self.snapshot.shape,
                source,
                source_bodies: vec![source_body.clone()],
                body_parent: source_body.clone(),
                primary_body: source_body,
                inherited_deleted: Vec::new(),
                inherited_splits: Default::default(),
                resolution: self.certification.as_ref().map(|value| value.resolution),
            },
            history,
            document,
            edges,
            operation,
        )
    }

    fn blend_sources(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<(Vec<u8>, Shape)>, String> {
        let mut sources = self.blend_face_sources(history, document)?;
        sources.extend(self.blend_lower_sources(history, document)?);
        sources.sort_by(|left, right| left.0.cmp(&right.0));
        Ok(sources)
    }

    fn blend_face_sources(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<(Vec<u8>, Shape)>, String> {
        let mut sources = Vec::new();
        for key in self.bindings.encoded_keys(history, document)? {
            let role = self.bindings.bind(
                history,
                document,
                key,
                ReferenceEnvelope::decode(key)
                    .map_err(|e| e.to_string())?
                    .kind,
            )?;
            if let Some(role) = role {
                sources.push((key.to_vec(), unique_face(&self.snapshot, *role)?.clone()));
            }
        }
        Ok(sources)
    }

    fn blend_lower_sources(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<(Vec<u8>, Shape)>, String> {
        let mut sources = Vec::new();
        let subentities = self
            .subentity_bindings
            .as_ref()
            .ok_or("blend input: expected certified named edges and vertices")?;
        for key in subentities.encoded_keys(history, document)? {
            let kind = ReferenceEnvelope::decode(key)
                .map_err(|e| e.to_string())?
                .kind;
            sources.push((
                key.to_vec(),
                subentities
                    .bind(history, document, key, kind)?
                    .shape
                    .clone(),
            ));
        }
        Ok(sources)
    }
}

struct BlendInputTopology<'a> {
    resolution: Option<PrismResolution>,
    body: &'a Shape,
    source: Vec<(Vec<u8>, Shape)>,
    source_bodies: Vec<Vec<u8>>,
    body_parent: Vec<u8>,
    primary_body: Vec<u8>,
    inherited_deleted: Vec<Vec<u8>>,
    inherited_splits: SplitTopologyNames,
}

#[derive(Clone, Copy)]
enum BlendOperation {
    Fillet(f64),
    Chamfer(f64),
}

impl BlendOperation {
    fn distance(self) -> f64 {
        match self {
            Self::Fillet(value) | Self::Chamfer(value) => value,
        }
    }
}

fn build_candidate(
    input: BlendInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    edges: &[Vec<u8>],
    operation: BlendOperation,
) -> Result<BlendCandidate, String> {
    validate_inputs(edges, operation.distance())?;
    validate_selected_edges(&input.source, edges)?;
    let mut copier = crate::checked_shape::copy_shape(input.body, "blend input")
        .map_err(|cause| cause.to_string())?;
    let copied = copy_named_sources(&mut copier, input.source)?;
    let shape = copier.shape().map_err(|cause| cause.to_string())?.clone();
    let (shape, history_records) = construct_candidate(&shape, &copied, edges, operation)?;
    Ok(BlendCandidate {
        resolution: input.resolution,
        achieved_tolerance: None,
        shape,
        history: history_records,
        source_bodies: input.source_bodies,
        body_parent: input.body_parent,
        primary_body: input.primary_body,
        inherited_deleted: input.inherited_deleted,
        inherited_splits: input.inherited_splits,
        document,
        scope: history.evaluation_scope(),
    })
}

fn validate_inputs(edges: &[Vec<u8>], radius: f64) -> Result<(), String> {
    let unique: std::collections::BTreeSet<_> = edges.iter().collect();
    if edges.is_empty() || unique.len() != edges.len() || !radius.is_finite() || radius <= 0. {
        return Err(format!(
            "blend edges {}, unique {}, distance {radius}: expected unique nonempty edges and finite positive distance",
            edges.len(),
            unique.len()
        ));
    }
    Ok(())
}

fn validate_selected_edges(sources: &[(Vec<u8>, Shape)], edges: &[Vec<u8>]) -> Result<(), String> {
    for key in edges {
        let shape = named_edge_source(sources, key)?;
        martensite_tkbrep::topods::edge(shape)
            .map_err(|cause| format!("blend edge {key:?}: expected an edge: {cause}"))?;
    }
    Ok(())
}

fn named_edge_source<'a>(sources: &'a [(Vec<u8>, Shape)], key: &[u8]) -> Result<&'a Shape, String> {
    sources
        .iter()
        .find(|(candidate, _)| candidate == key)
        .map(|(_, shape)| shape)
        .ok_or_else(|| format!("blend edge {key:?}: expected a named edge in the source body"))
}

fn copy_named_sources(
    copier: &mut BRepBuilderAPICopy,
    sources: Vec<(Vec<u8>, Shape)>,
) -> Result<Vec<(Vec<u8>, Shape)>, String> {
    sources
        .into_iter()
        .map(|(key, shape)| {
            let copies = copier.modified(&shape).map_err(|cause| cause.to_string())?;
            if copies.len() != 1 {
                return Err(format!(
                    "blend input {key:?}, copies {}: expected one detached entity",
                    copies.len()
                ));
            }
            Ok((key, copies.first().expect("one verified copy").clone()))
        })
        .collect()
}

fn construct_candidate(
    body: &Shape,
    sources: &[(Vec<u8>, Shape)],
    edges: &[Vec<u8>],
    operation: BlendOperation,
) -> Result<(Shape, Vec<BlendHistoryRecord>), String> {
    match operation {
        BlendOperation::Fillet(radius) => construct_fillet(body, sources, edges, radius),
        BlendOperation::Chamfer(distance) => construct_chamfer(body, sources, edges, distance),
    }
}

fn construct_fillet(
    body: &Shape,
    sources: &[(Vec<u8>, Shape)],
    edges: &[Vec<u8>],
    radius: f64,
) -> Result<(Shape, Vec<BlendHistoryRecord>), String> {
    let mut maker = BRepFilletAPIMakeFillet::new(body, FilletShape::Rational)
        .map_err(|cause| cause.to_string())?;
    for key in edges {
        let shape = named_edge_source(sources, key)?;
        let edge = martensite_tkbrep::topods::edge(shape).map_err(|cause| cause.to_string())?;
        maker.add_with_radius(radius, &edge).map_err(|cause| {
            format!("fillet edge {key:?}, radius {radius}: expected an admissible fillet: {cause}")
        })?;
    }
    finish_blend(&mut maker, sources, "fillet", radius)
}

fn construct_chamfer(
    body: &Shape,
    sources: &[(Vec<u8>, Shape)],
    edges: &[Vec<u8>],
    distance: f64,
) -> Result<(Shape, Vec<BlendHistoryRecord>), String> {
    let mut maker = BRepFilletAPIMakeChamfer::new(body).map_err(|cause| cause.to_string())?;
    for key in edges {
        let shape = named_edge_source(sources, key)?;
        let edge = martensite_tkbrep::topods::edge(shape).map_err(|cause| cause.to_string())?;
        maker.add_with_dist(distance, &edge).map_err(|cause| {
            format!(
                "chamfer edge {key:?}, distance {distance}: expected an admissible chamfer: {cause}"
            )
        })?;
    }
    finish_blend(&mut maker, sources, "chamfer", distance)
}

fn finish_blend(
    maker: &mut impl BRepBuilderAPIMakeShape,
    sources: &[(Vec<u8>, Shape)],
    operation: &str,
    distance: f64,
) -> Result<(Shape, Vec<BlendHistoryRecord>), String> {
    let shape = maker
        .shape()
        .map_err(|cause| {
            format!("{operation} distance {distance}: expected a completed shape: {cause}")
        })?
        .clone();
    crate::checked_shape::require_geometry(&shape, operation).map_err(|cause| cause.to_string())?;
    let history = sources
        .iter()
        .map(|(name, source)| history_record(maker, name, source))
        .collect::<Result<_, _>>()?;
    Ok((shape, history))
}

fn history_record(
    maker: &mut impl BRepBuilderAPIMakeShape,
    name: &[u8],
    source: &Shape,
) -> Result<BlendHistoryRecord, String> {
    let generated = maker
        .generated(source)
        .map_err(|cause| cause.to_string())?
        .iter()
        .cloned()
        .collect();
    let modified = maker
        .modified(source)
        .map_err(|cause| cause.to_string())?
        .iter()
        .cloned()
        .collect();
    let deleted = maker
        .is_deleted(source)
        .map_err(|cause| cause.to_string())?;
    Ok(BlendHistoryRecord {
        name: name.to_vec(),
        source: source.clone(),
        generated,
        modified,
        deleted,
        transient: false,
    })
}
