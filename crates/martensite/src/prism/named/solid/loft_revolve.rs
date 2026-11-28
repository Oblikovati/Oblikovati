// SPDX-License-Identifier: GPL-2.0-only
//! Loft and Revolve evaluation helpers for `CertifiedNamedSolidEvaluator`.
use super::*;
use oblikovati_model::feature::part::PartFeatureFailure;

impl<'a> CertifiedNamedSolidEvaluator<'a> {
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
            derived_sources: None,
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
                derived_sources: None,
            },
        }
    }
    /// Attach loaded source Part snapshots for Derived Part evaluation.
    pub fn with_derived_sources(
        mut self,
        sources: &'a std::collections::BTreeMap<
            String,
            oblikovati_model::history::DocumentSnapshot,
        >,
    ) -> Self {
        self.derived_sources = Some(sources);
        self
    }
    pub(crate) fn resolution_policy(&self) -> Option<PrismResolutionPolicy> {
        self.prism.as_ref().map(|prism| prism.resolution_policy())
    }
}

impl CertifiedNamedSolidEvaluator<'_> {
    pub(super) fn evaluate_named_loft(
        &mut self,
        feature: FeatureId,
        input: oblikovati_model::feature::sketch_loft::ResolvedSketchLoft,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let Some(prism) = &self.prism else {
            return Err(PartFeatureFailure::Unsupported(
                "loft: expected an explicit model resolution policy".into(),
            ));
        };
        let failure =
            |error: EvaluationFailure| PartFeatureFailure::Kernel(error.for_feature(feature));
        let candidate =
            crate::loft::build_loft(&input, prism.resolution_policy()).map_err(failure)?;
        let topology = BlendNamedTopology::from_loft(
            self.history,
            self.document,
            feature,
            input.sections.map(|section| section.source),
            input.center_line.map(|path| path.source),
            candidate,
        )
        .map_err(|cause| {
            failure(crate::EvaluationPhase::Naming.refused(
                crate::EvaluationFailureKind::InvalidReference,
                crate::EvaluationEntity::Body,
                format!("loft topology: {cause}"),
            ))
        })?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }

    pub(super) fn evaluate_named_revolve(
        &mut self,
        feature: FeatureId,
        input: oblikovati_model::feature::sketch_revolve::ResolvedSketchRevolve<'_>,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        let Some(prism) = &self.prism else {
            return Err(PartFeatureFailure::Unsupported(
                "revolve: expected an explicit model resolution policy".into(),
            ));
        };
        let sketch_id = input.sketch.sketch.id().0;
        let policy = prism.resolution_policy();
        let failure =
            |error: EvaluationFailure| PartFeatureFailure::Kernel(error.for_feature(feature));
        let topology = if input.surface {
            let candidate =
                crate::revolve::build_surface_revolve(input, policy).map_err(failure)?;
            BlendNamedTopology::from_surface_revolve(
                self.history,
                self.document,
                feature,
                sketch_id,
                candidate,
            )
        } else {
            let candidate = crate::revolve::build_revolve(input, policy).map_err(failure)?;
            BlendNamedTopology::from_revolve(
                self.history,
                self.document,
                feature,
                sketch_id,
                candidate,
            )
        }
        .map_err(|cause| {
            failure(crate::EvaluationPhase::Naming.refused(
                crate::EvaluationFailureKind::InvalidReference,
                crate::EvaluationEntity::Body,
                format!("revolve topology: {cause}"),
            ))
        })?;
        Ok(NamedSolid::Derived(Box::new(topology)))
    }

    pub(super) fn apply_derived_tool_boolean(
        &mut self,
        feature: FeatureId,
        body: &NamedSolid,
        tool: Box<BlendNamedTopology>,
        operation: oblikovati_model::feature::sketch_prism::PrismBoolean,
        label: &str,
    ) -> Result<NamedSolid, PartFeatureFailure<EvaluationFailure>> {
        use super::super::blend::extrusion_boolean::BooleanFailure;
        let failure = |kind, message| {
            PartFeatureFailure::Kernel(
                crate::EvaluationPhase::Prism
                    .refused(kind, crate::EvaluationEntity::Body, message)
                    .for_feature(feature),
            )
        };
        let tool = tool
            .sweep_boolean_tool(self.history, self.document)
            .map_err(|message| failure(crate::EvaluationFailureKind::InvalidResult, message))?;
        let candidate = body
            .boolean_tools(
                self.history,
                self.document,
                vec![tool],
                operation,
                &format!("{label} {operation}"),
            )
            .map_err(|decline| match decline {
                BooleanFailure::UnchangedCut { faces, .. } => failure(
                    crate::EvaluationFailureKind::UnchangedCut,
                    format!("The feature did not change the number of faces ({faces})."),
                ),
                BooleanFailure::Refused(message) => {
                    failure(crate::EvaluationFailureKind::InvalidResult, message)
                }
            })?;
        let named = candidate
            .name_topology(self.history, self.document, feature)
            .map_err(|message| failure(crate::EvaluationFailureKind::InvalidResult, message))?;
        let result = NamedSolid::Derived(Box::new(named));
        self.verify_result_lineage(body, &result)
            .map_err(PartFeatureFailure::BodyReference)?;
        Ok(result)
    }
}
