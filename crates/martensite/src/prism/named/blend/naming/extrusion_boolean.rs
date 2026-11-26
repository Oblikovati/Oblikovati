// SPDX-License-Identifier: GPL-2.0-only
use super::super::solid_boolean::{BooleanFailure, BooleanTool};
use super::*;
use oblikovati_model::feature::sketch_prism::PrismBoolean;

impl BlendNamedTopology {
    pub(in crate::prism::named::blend) fn boolean_tools(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        tools: Vec<BooleanTool>,
        kind: PrismBoolean,
        operation: &str,
    ) -> Result<BlendCandidate, BooleanFailure> {
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
            tools,
            kind,
            operation,
        )
    }
}
