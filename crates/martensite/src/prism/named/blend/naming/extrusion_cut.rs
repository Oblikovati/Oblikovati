// SPDX-License-Identifier: GPL-2.0-only
use super::super::subtraction::SubtractionTool;
use super::*;

impl BlendNamedTopology {
    pub(in crate::prism::named::blend) fn extrusion_cut_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tool: SubtractionTool<'_>,
    ) -> Result<BlendCandidate, String> {
        let input = self.source_topology(history, document)?;
        super::super::subtraction::build_subtraction_candidate(
            BlendInputTopology {
                body: &input.body,
                source: input.entities,
                source_bodies: input.body_keys,
                body_parent: self.active_body.clone(),
                primary_body: self.primary_body.clone(),
                inherited_deleted: self.deleted.clone(),
                resolution: self.resolution,
            },
            history,
            document,
            tool,
            "extrude Cut",
        )
    }
}
