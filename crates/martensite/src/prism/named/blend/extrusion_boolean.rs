// SPDX-License-Identifier: GPL-2.0-only
use super::{solid_boolean::BooleanTool, *};
use oblikovati_model::feature::sketch_prism::PrismBoolean;

impl NamedSolid {
    pub(in crate::prism::named) fn extrusion_boolean_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tool: &NamedPrismSnapshot,
        kind: PrismBoolean,
    ) -> Result<BlendCandidate, String> {
        let resolution = tool
            .certification
            .as_ref()
            .ok_or("extrude Boolean: expected a certified tool prism")?
            .resolution;
        let tool = BooleanTool {
            shape: &tool.snapshot.shape,
            sources: tool.blend_sources(history, document)?,
            resolution: Some(resolution),
        };
        match self {
            Self::Prism(body) => body.extrusion_boolean_candidate(history, document, tool, kind),
            Self::Derived(body) => body.extrusion_boolean_candidate(history, document, tool, kind),
        }
    }
}

impl NamedPrismSnapshot {
    fn extrusion_boolean_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tool: BooleanTool<'_>,
        kind: PrismBoolean,
    ) -> Result<BlendCandidate, String> {
        let body = self.body_reference_key(history, document)?.to_vec();
        super::solid_boolean::build_boolean_candidate(
            BlendInputTopology {
                body: &self.snapshot.shape,
                source: self.blend_sources(history, document)?,
                source_bodies: vec![body.clone()],
                body_parent: body.clone(),
                primary_body: body,
                inherited_deleted: Vec::new(),
                resolution: self.certification.as_ref().map(|value| value.resolution),
            },
            history,
            document,
            tool,
            kind,
            &format!("extrude {kind}"),
        )
    }
}
