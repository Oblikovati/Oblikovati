// SPDX-License-Identifier: GPL-2.0-only
//! Ordered evaluation of named prism and fillet bodies (#4045).
use super::*;
use oblikovati_model::feature::{
    part::{FilletEvaluator, SketchPrismEvaluator},
    sketch_prism::ResolvedSketchPrism,
};
use std::collections::BTreeSet;
type NameLists = (Vec<Vec<u8>>, Vec<Vec<u8>>);

/// A named current solid produced by ordered Part replay.
pub enum NamedSolid {
    Prism(Box<NamedPrismSnapshot>),
    Fillet(Box<BlendNamedTopology>),
}

impl NamedSolid {
    /// Borrow the source prism when this result is a prism; example: `body.as_prism()`.
    pub fn as_prism(&self) -> Option<&NamedPrismSnapshot> {
        match self {
            Self::Prism(body) => Some(body),
            Self::Fillet(_) => None,
        }
    }

    /// List the primary body and exact face keys; example: `body.reference_keys(&history, document)?`.
    pub fn reference_keys(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<Vec<u8>>, String> {
        let mut keys = self.face_reference_keys(history, document)?;
        keys.push(self.body_reference_key(history, document)?.to_vec());
        keys.sort();
        Ok(keys)
    }
    /// Compute visible face outlines from exact face names; example: `body.face_outlines(&history, document, None, options)?`.
    pub fn face_outlines(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        only: Option<&[u8]>,
        options: PrismMeshOptions,
    ) -> Result<Vec<[[f64; 3]; 2]>, String> {
        let keys = match only {
            Some(key) => vec![key.to_vec()],
            None => self.face_reference_keys(history, document)?,
        };
        let faces = keys
            .iter()
            .map(|key| self.native_face(history, document, key).cloned())
            .collect::<Result<Vec<_>, _>>()?;
        super::super::mesh::named_face_boundaries(
            self.native_shape(history, document)?,
            &faces,
            options,
        )
    }

    pub(super) fn native_face<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<&'a Shape, String> {
        match self {
            Self::Prism(body) => {
                let role = body
                    .bindings
                    .bind(history, document, key, 1)?
                    .ok_or_else(|| format!("face {key:?}: expected a face role"))?;
                unique_face(&body.snapshot, role)
            }
            Self::Fillet(body) => body.native_face(history, document, key),
        }
    }
    pub(super) fn native_shape<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<&'a Shape, String> {
        match self {
            Self::Prism(body) => {
                body.body_reference_key(history, document)?;
                Ok(&body.snapshot.shape)
            }
            Self::Fillet(body) => body.native_shape(history, document),
        }
    }
    /// Read analytic body bounds center; example: `body.body_bounds_center(&history, document, key)?`.
    pub fn body_bounds_center(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<[f64; 3], String> {
        match self {
            Self::Prism(body) => body.body_bounds_center(history, document, key),
            Self::Fillet(body) => body.body_bounds_center(history, document, key),
        }
    }

    /// Read analytic face bounds center; example: `body.face_bounds_center(&history, document, key)?`.
    pub fn face_bounds_center(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<[f64; 3], String> {
        match self {
            Self::Prism(body) => body.face_bounds_center(history, document, key),
            Self::Fillet(body) => body.face_bounds_center(history, document, key),
        }
    }
    /// Read analytic face information; example: `body.face_topology(&history, document, key)?`.
    pub fn face_topology(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<PrismFaceTopology, String> {
        match self {
            Self::Prism(body) => body.face_topology(history, document, key),
            Self::Fillet(body) => body.face_topology(history, document, key),
        }
    }

    /// Count a named face's loops; example: `body.face_loop_count(&history, document, key)?`.
    pub fn face_loop_count(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<usize, String> {
        match self {
            Self::Prism(body) => body
                .face_report(history, document, key)
                .map(|report| report.loops),
            Self::Fillet(body) => body.face_loop_count(history, document, key),
        }
    }
    /// Enumerate exact face names; example: `body.face_reference_keys(&history, document)?`.
    pub fn face_reference_keys(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<Vec<u8>>, String> {
        match self {
            Self::Prism(body) => body
                .reference_keys(history, document)?
                .into_iter()
                .filter_map(|key| {
                    match oblikovati_model::identity::ReferenceEnvelope::decode(&key) {
                        Ok(envelope) if envelope.kind == 1 => Some(Ok(key)),
                        Ok(_) => None,
                        Err(cause) => Some(Err(cause.to_string())),
                    }
                })
                .collect(),
            Self::Fillet(body) => body.keys_of_kinds(history, document, &[1]),
        }
    }
    /// Count exact body topology; example: `body.topology_summary(&history, document, key)?`.
    pub fn topology_summary(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<PrismTopologySummary, String> {
        match self {
            Self::Prism(body) => body
                .body_report(history, document, key)
                .map(|report| report.topology.clone()),
            Self::Fillet(body) => body.topology_summary(history, document, key),
        }
    }

    /// Integrate a named face; example: `body.face_area(&history, document, key)?`.
    pub fn face_area(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<f64, String> {
        match self {
            Self::Prism(body) => body
                .face_report(history, document, key)
                .map(|report| report.area),
            Self::Fillet(body) => body.face_area(history, document, key),
        }
    }

    /// Enumerate named edges and vertices; example: `body.subentity_reference_keys(&history, document)?`.
    pub fn subentity_reference_keys(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<Vec<Vec<u8>>, String> {
        match self {
            Self::Prism(body) => body.subentity_reference_keys(history, document),
            Self::Fillet(body) => body.keys_of_kinds(history, document, &[2, 3]),
        }
    }

    /// Integrate a named edge; example: `body.edge_length(&history, document, key)?`.
    pub fn edge_length(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<f64, String> {
        match self {
            Self::Prism(body) => body.edge_length(history, document, key),
            Self::Fillet(body) => body.edge_length(history, document, key),
        }
    }

    /// Read an exact edge or vertex's analytic bounds center; example: `body.subentity_point(&history, document, key)?`.
    pub fn subentity_point(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<[f64; 3], String> {
        match self {
            Self::Prism(body) => body
                .subentity_provenance(history, document, key)
                .map(|record| record.point),
            Self::Fillet(body) => body.subentity_point(history, document, key),
        }
    }
    /// Resolve the primary body key; example: `body.body_reference_key(&history, document)?`.
    pub fn body_reference_key<'a>(
        &'a self,
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<&'a [u8], String> {
        match self {
            Self::Prism(body) => body.body_reference_key(history, document),
            Self::Fillet(body) => body.body_reference_key(history, document),
        }
    }

    /// Integrate properties for an exact body name; example: `body.physical_properties(&history, document, key)?`.
    pub fn physical_properties(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        key: &[u8],
    ) -> Result<crate::AnalyticBodyProperties, String> {
        match self {
            Self::Prism(body) => body.physical_properties(history, document, key),
            Self::Fillet(body) => body.physical_properties(history, document, key),
        }
    }
    /// Integrate the current body; example: `body.volume(&history, document)?`.
    pub fn volume(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<f64, String> {
        match self {
            Self::Prism(body) => body
                .physical_properties(
                    history,
                    document,
                    body.body_reference_key(history, document)?,
                )
                .map(|properties| properties.geometry.volume),
            Self::Fillet(body) => body.volume(history, document),
        }
    }

    fn names(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
    ) -> Result<NameLists, String> {
        match self {
            Self::Prism(body) => {
                let mut live = body.reference_keys(history, document)?;
                live.extend(body.subentity_reference_keys(history, document)?);
                Ok((live, Vec::new()))
            }
            Self::Fillet(body) => Ok((body.live_keys().to_vec(), body.deleted_keys().to_vec())),
        }
    }
}

/// Integrate current named solids as one unit-density system.
/// Example: `named_solid_part_properties(&history, document, replay.ordered_bodies())?`.
pub fn named_solid_part_properties<'a>(
    history: &'a DocumentEditHistory,
    document: DocumentGuid,
    bodies: impl IntoIterator<Item = &'a NamedSolid>,
) -> Result<Option<crate::AnalyticBodyProperties>, String> {
    let shapes = bodies
        .into_iter()
        .map(|body| body.native_shape(history, document))
        .collect::<Result<Vec<_>, _>>()?;
    crate::analytic_body::integrate_system(shapes)
}

impl crate::BodyPresentationSource for NamedSolid {
    fn body_triangles(&self, options: PrismMeshOptions) -> Result<Vec<[[f64; 3]; 3]>, String> {
        match self {
            Self::Prism(body) => {
                super::super::BodyPresentationSource::body_triangles(body.as_ref(), options)
            }
            Self::Fillet(body) => body.display_triangles(options),
        }
    }
}

/// Evaluate certified prisms and named fillets in one ordered Part program.
pub struct CertifiedNamedSolidEvaluator<'a> {
    prism: Option<CertifiedNamedPrismEvaluator<'a>>,
    diagnostic: NamedPrismEvaluator<'a>,
    history: &'a DocumentEditHistory,
    document: DocumentGuid,
}
impl<'a> CertifiedNamedSolidEvaluator<'a> {
    /// Bind a document and resolution; example: `CertifiedNamedSolidEvaluator::new(&history, document, resolution)`.
    pub fn new(
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        resolution: PrismResolution,
    ) -> Self {
        Self {
            prism: Some(CertifiedNamedPrismEvaluator::new(
                history, document, resolution,
            )),
            diagnostic: NamedPrismEvaluator::new(history, document),
            history,
            document,
        }
    }
    /// Bind the host's per-sketch resolution policy; example: `CertifiedNamedSolidEvaluator::with_policy(&history, document, policy)`.
    pub fn with_policy(
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        policy: PrismResolutionPolicy,
    ) -> Self {
        Self {
            prism: Some(CertifiedNamedPrismEvaluator::with_policy(
                history, document, policy,
            )),
            diagnostic: NamedPrismEvaluator::new(history, document),
            history,
            document,
        }
    }
    /// Select the host's optional resolution policy; example: `CertifiedNamedSolidEvaluator::with_optional_policy(&history, document, None)`.
    pub fn with_optional_policy(
        history: &'a DocumentEditHistory,
        document: DocumentGuid,
        policy: Option<PrismResolutionPolicy>,
    ) -> Self {
        match policy {
            Some(policy) => Self::with_policy(history, document, policy),
            None => Self {
                prism: None,
                diagnostic: NamedPrismEvaluator::new(history, document),
                history,
                document,
            },
        }
    }
}

impl SketchPrismEvaluator for CertifiedNamedSolidEvaluator<'_> {
    type Body = NamedSolid;
    type Failure = EvaluationFailure;
    fn verify_body_reference(&self, body: &NamedSolid, key: &[u8]) -> Result<(), String> {
        match body {
            NamedSolid::Prism(body) => self.diagnostic.verify_body_reference(body, key),
            NamedSolid::Fillet(body) => body
                .contains(self.history, self.document, key, 4)
                .map(|_| ()),
        }
    }
    fn resolve_planar_face(
        &self,
        body: &NamedSolid,
        key: &[u8],
    ) -> Result<oblikovati_model::sketch::SketchPlane, String> {
        match body {
            NamedSolid::Prism(body) => self.diagnostic.resolve_planar_face(body, key),
            NamedSolid::Fillet(body) => body.face_plane(self.history, self.document, key),
        }
    }
    fn evaluate_prism(
        &mut self,
        feature: FeatureId,
        input: ResolvedSketchPrism<'_>,
    ) -> Result<NamedSolid, EvaluationFailure> {
        let body = match &mut self.prism {
            Some(prism) => prism.evaluate_prism(feature, input),
            None => self.diagnostic.evaluate_prism(feature, input),
        }?;
        Ok(NamedSolid::Prism(Box::new(body)))
    }
    fn fillet_evaluator(
        &mut self,
    ) -> Option<&mut dyn FilletEvaluator<NamedSolid, EvaluationFailure>> {
        Some(self)
    }
}

impl FilletEvaluator<NamedSolid, EvaluationFailure> for CertifiedNamedSolidEvaluator<'_> {
    fn evaluate_fillet(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        edges: &[Vec<u8>],
        radius: f64,
    ) -> Result<NamedSolid, EvaluationFailure> {
        let candidate = match body {
            NamedSolid::Prism(body) => {
                body.fillet_candidate(self.history, self.document, edges, radius)
            }
            NamedSolid::Fillet(body) => {
                body.fillet_candidate(self.history, self.document, edges, radius)
            }
        }
        .map_err(|message| failure(Phase::Fillet, feature, message))?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| failure(Phase::Naming, feature, message))?;
        Ok(NamedSolid::Fillet(Box::new(named)))
    }

    fn verify_fillet_lineage(&self, before: &NamedSolid, after: &NamedSolid) -> Result<(), String> {
        let (prior_live, prior_deleted) = before.names(self.history, self.document)?;
        let (live, deleted) = after.names(self.history, self.document)?;
        let accounted: BTreeSet<_> = live.iter().chain(&deleted).collect();
        for key in &prior_live {
            if !accounted.contains(key) {
                return Err(format!(
                    "fillet input name {key:?}: expected a live or deleted result record"
                ));
            }
        }
        for key in &prior_deleted {
            if !deleted.contains(key) {
                return Err(format!(
                    "fillet deleted name {key:?}: expected a retained deletion record"
                ));
            }
        }
        Ok(())
    }
}

fn failure(phase: Phase, feature: FeatureId, message: String) -> EvaluationFailure {
    phase
        .refused(Kind::InvalidResult, Entity::Body, message)
        .for_feature(feature)
}
