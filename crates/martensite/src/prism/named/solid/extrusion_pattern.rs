// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::prism::named::blend::extrusion_boolean::BooleanTool;
use oblikovati_model::{
    feature::{
        extrusion_pattern::{ExtrusionPatternOccurrence, PatternPrismSource},
        move_body::ResolvedBodyMotion,
        sketch_prism::PrismBoolean,
    },
    history::ResolvedDocumentSketch,
    identity::SemanticReferenceName,
};

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_extrusion_pattern(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        sources: &[PatternPrismSource<'_>],
        occurrences: &[ExtrusionPatternOccurrence],
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        if occurrences.is_empty() {
            return Ok(body.clone());
        }
        let body_parent = body
            .body_reference_key(self.history, self.document)
            .map_err(PartFeatureFailure::BodyReference)?
            .to_vec();
        let failure = |cause| pattern_failure(feature, cause);
        let prism = self
            .prism
            .as_mut()
            .ok_or_else(|| failure("expected certified prism replay".into()))?;
        let seeds = sources
            .iter()
            .map(|source| {
                let seed = &source.prism;
                prism
                    .evaluate_prism(
                        source.source,
                        ResolvedSketchPrism {
                            sketch: ResolvedDocumentSketch { ..seed.sketch },
                            sweep_start: seed.sweep_start,
                            selection: seed.selection,
                            displacement: seed.displacement,
                        },
                    )
                    .map_err(|cause| PartFeatureFailure::Kernel(cause.for_feature(feature)))
            })
            .collect::<Result<Vec<_>, _>>()?;

        let mut current = body.clone();
        let mut tools = Vec::new();
        let mut operation = sources
            .first()
            .ok_or_else(|| failure("expected a source extrusion".into()))?
            .operation;
        for occurrence in occurrences {
            for (source, seed) in sources.iter().zip(&seeds) {
                if source.operation != operation {
                    current = self.apply_pattern_tools(
                        feature,
                        &current,
                        std::mem::take(&mut tools),
                        operation,
                        &body_parent,
                    )?;
                    operation = source.operation;
                }
                let marker = occurrence_name(self.document, feature, source.source, *occurrence)
                    .map_err(failure)?;
                let tool = seed
                    .transformed_boolean_tool(
                        self.history,
                        self.document,
                        super::move_body_evaluation::to_rigid_motion(&occurrence.motion),
                        &marker,
                    )
                    .map_err(|cause| {
                        failure(format!(
                            "source {:?}, cell {:?}: {cause}",
                            source.source, occurrence.offset
                        ))
                    })?;
                tools.push(tool);
            }
        }
        current = self.apply_pattern_tools(feature, &current, tools, operation, &body_parent)?;
        self.verify_result_lineage(body, &current)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(current)
    }

    fn apply_pattern_tools(
        &self,
        feature: FeatureId,
        body: &NamedSolid,
        tools: Vec<BooleanTool>,
        operation: PrismBoolean,
        body_parent: &[u8],
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let size = tools.len();
        let failure =
            |cause| pattern_failure(feature, format!("{size} {operation} tools: {cause}"));
        let result = body
            .extrusion_boolean_tools(self.history, self.document, tools, operation)
            .map_err(failure)?
            .with_body_parent(body_parent)
            .name_topology(self.history, self.document, feature)
            .map_err(failure)?;
        let result = NamedSolid::Derived(Box::new(result));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}

fn pattern_failure(feature: FeatureId, cause: String) -> PartFeatureFailure<EvaluationFailure> {
    PartFeatureFailure::Kernel(
        Phase::Prism
            .refused(
                Kind::InvalidResult,
                Entity::Body,
                format!("extrusion pattern: {cause}"),
            )
            .for_feature(feature),
    )
}

fn occurrence_name(
    document: DocumentGuid,
    feature: FeatureId,
    source: FeatureId,
    occurrence: ExtrusionPatternOccurrence,
) -> Result<Vec<u8>, String> {
    let [column, row] = occurrence
        .offset
        .map(|offset| (offset + i64::from(u32::MAX) + 1) as u64);
    SemanticReferenceName::new(
        document,
        feature.0,
        4,
        match occurrence.motion {
            ResolvedBodyMotion::Translation(_) => "rectangular-pattern",
            ResolvedBodyMotion::Rotation { .. } => "circular-pattern",
            ResolvedBodyMotion::Reflection { .. } => "mirror",
        },
        "occurrence",
        &[source.0, column, row],
    )?
    .encode()
}
