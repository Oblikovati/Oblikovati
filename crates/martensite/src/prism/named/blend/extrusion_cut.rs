// SPDX-License-Identifier: GPL-2.0-only
use super::{subtraction::SubtractionTool, *};

impl NamedSolid {
    pub(in crate::prism::named) fn extrusion_cut_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tool: &NamedPrismSnapshot,
    ) -> Result<BlendCandidate, String> {
        let resolution = tool
            .certification
            .as_ref()
            .ok_or("extrude Cut: expected a certified tool prism")?
            .resolution;
        let tool = SubtractionTool {
            shape: &tool.snapshot.shape,
            sources: tool.blend_sources(history, document)?,
            resolution: Some(resolution),
        };
        match self {
            Self::Prism(body) => body.extrusion_cut_candidate(history, document, tool),
            Self::Derived(body) => body.extrusion_cut_candidate(history, document, tool),
        }
    }
}

impl NamedPrismSnapshot {
    fn extrusion_cut_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tool: SubtractionTool<'_>,
    ) -> Result<BlendCandidate, String> {
        let body = self.body_reference_key(history, document)?.to_vec();
        super::subtraction::build_subtraction_candidate(
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
            "extrude Cut",
        )
    }
}
