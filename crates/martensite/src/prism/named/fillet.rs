// SPDX-License-Identifier: GPL-2.0-only
//! Detached fillet construction and exact kernel history (#4045).
use super::*;
use martensite_tkfillet::{brep_fillet_api::BRepFilletAPIMakeFillet, chfi3d::FilletShape};
use martensite_tktopalgo::brep_builder_api::{BRepBuilderAPICopy, BRepBuilderAPIMakeShape};
use oblikovati_model::identity::ReferenceEnvelope;
mod naming;
mod queries;
pub use naming::FilletNamedTopology;

/// A validated detached result and its input history, before assigning result names.
pub struct FilletCandidate {
    shape: Shape,
    history: Vec<FilletHistoryRecord>,
    source_bodies: Vec<Vec<u8>>,
    body_parent: Vec<u8>,
    primary_body: Vec<u8>,
    inherited_deleted: Vec<Vec<u8>>,
    document: DocumentGuid,
    scope: EvaluationScope,
}

/// The kernel history for one exact named input entity.
pub struct FilletHistoryRecord {
    name: Vec<u8>,
    source: Shape,
    generated: Vec<Shape>,
    modified: Vec<Shape>,
    deleted: bool,
}

impl FilletCandidate {
    /// Read input records in name order; example: `candidate.history()`.
    pub fn history(&self) -> &[FilletHistoryRecord] {
        &self.history
    }
    /// Integrate the detached result analytically; example: `candidate.volume()?`.
    pub fn volume(&self) -> Result<f64, String> {
        crate::checked_shape::analytic_properties(&self.shape, "fillet candidate")
            .map(|properties| properties.volume)
            .map_err(|cause| cause.to_string())
    }
}

impl FilletHistoryRecord {
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
    ) -> Result<FilletCandidate, String> {
        validate_inputs(edges, radius)?;
        let source = self.fillet_sources(history, document)?;
        let source_body = self.body_reference_key(history, document)?.to_vec();
        build_candidate(
            FilletInputTopology {
                body: &self.snapshot.shape,
                source,
                source_bodies: vec![source_body.clone()],
                body_parent: source_body.clone(),
                primary_body: source_body,
                inherited_deleted: Vec::new(),
            },
            history,
            document,
            edges,
            radius,
        )
    }

    fn fillet_sources(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<(Vec<u8>, Shape)>, String> {
        let mut sources = self.fillet_face_sources(history, document)?;
        sources.extend(self.fillet_lower_sources(history, document)?);
        sources.sort_by(|left, right| left.0.cmp(&right.0));
        Ok(sources)
    }

    fn fillet_face_sources(
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

    fn fillet_lower_sources(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<(Vec<u8>, Shape)>, String> {
        let mut sources = Vec::new();
        let subentities = self
            .subentity_bindings
            .as_ref()
            .ok_or("fillet input: expected certified named edges and vertices")?;
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

struct FilletInputTopology<'a> {
    body: &'a Shape,
    source: Vec<(Vec<u8>, Shape)>,
    source_bodies: Vec<Vec<u8>>,
    body_parent: Vec<u8>,
    primary_body: Vec<u8>,
    inherited_deleted: Vec<Vec<u8>>,
}

fn build_candidate(
    input: FilletInputTopology<'_>,
    history: &DocumentEditHistory,
    document: DocumentGuid,
    edges: &[Vec<u8>],
    radius: f64,
) -> Result<FilletCandidate, String> {
    validate_inputs(edges, radius)?;
    validate_selected_edges(&input.source, edges)?;
    let mut copier = crate::checked_shape::copy_shape(input.body, "fillet input")
        .map_err(|cause| cause.to_string())?;
    let copied = copy_named_sources(&mut copier, input.source)?;
    let shape = copier.shape().map_err(|cause| cause.to_string())?.clone();
    let (shape, history_records) = construct_candidate(&shape, &copied, edges, radius)?;
    Ok(FilletCandidate {
        shape,
        history: history_records,
        source_bodies: input.source_bodies,
        body_parent: input.body_parent,
        primary_body: input.primary_body,
        inherited_deleted: input.inherited_deleted,
        document,
        scope: history.evaluation_scope(),
    })
}

fn validate_inputs(edges: &[Vec<u8>], radius: f64) -> Result<(), String> {
    let unique: std::collections::BTreeSet<_> = edges.iter().collect();
    if edges.is_empty() || unique.len() != edges.len() || !radius.is_finite() || radius <= 0. {
        return Err(format!(
            "fillet edges {}, unique {}, radius {radius}: expected unique nonempty edges and finite positive radius",
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
            .map_err(|cause| format!("fillet edge {key:?}: expected an edge: {cause}"))?;
    }
    Ok(())
}

fn named_edge_source<'a>(sources: &'a [(Vec<u8>, Shape)], key: &[u8]) -> Result<&'a Shape, String> {
    sources
        .iter()
        .find(|(candidate, _)| candidate == key)
        .map(|(_, shape)| shape)
        .ok_or_else(|| format!("fillet edge {key:?}: expected a named edge in the source body"))
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
                    "fillet input {key:?}, copies {}: expected one detached entity",
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
    radius: f64,
) -> Result<(Shape, Vec<FilletHistoryRecord>), String> {
    let mut maker = BRepFilletAPIMakeFillet::new(body, FilletShape::Rational)
        .map_err(|cause| cause.to_string())?;
    for key in edges {
        let shape = named_edge_source(sources, key)?;
        let edge = martensite_tkbrep::topods::edge(shape).map_err(|cause| cause.to_string())?;
        maker.add_with_radius(radius, &edge).map_err(|cause| {
            format!("fillet edge {key:?}, radius {radius}: expected an admissible fillet: {cause}")
        })?;
    }
    let shape = maker
        .shape()
        .map_err(|cause| format!("fillet radius {radius}: expected a completed shape: {cause}"))?
        .clone();
    crate::checked_shape::require_geometry(&shape, "fillet result")
        .map_err(|cause| cause.to_string())?;
    let history = sources
        .iter()
        .map(|(name, source)| history_record(&mut maker, name, source))
        .collect::<Result<_, _>>()?;
    Ok((shape, history))
}

fn history_record(
    maker: &mut BRepFilletAPIMakeFillet,
    name: &[u8],
    source: &Shape,
) -> Result<FilletHistoryRecord, String> {
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
    Ok(FilletHistoryRecord {
        name: name.to_vec(),
        source: source.clone(),
        generated,
        modified,
        deleted,
    })
}
