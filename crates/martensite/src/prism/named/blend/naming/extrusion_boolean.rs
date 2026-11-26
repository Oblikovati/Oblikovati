// SPDX-License-Identifier: GPL-2.0-only
use super::super::solid_boolean::BooleanTool;
use super::*;
use oblikovati_model::feature::sketch_prism::PrismBoolean;

impl BlendNamedTopology {
    pub(in crate::prism::named::blend) fn extrusion_boolean_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tool: BooleanTool<'_>,
        kind: PrismBoolean,
    ) -> Result<BlendCandidate, String> {
        let input = self.source_topology(history, document)?;
        super::super::solid_boolean::build_boolean_candidate(
            BlendInputTopology {
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
            tool,
            kind,
            &format!("extrude {kind}"),
        )
    }
}
