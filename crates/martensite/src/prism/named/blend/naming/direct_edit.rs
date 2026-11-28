// SPDX-License-Identifier: GPL-2.0-only
//! Retain named input lineage through Direct Edit transformations (#3918).
use super::*;
use oblikovati_model::feature::direct_edit::ResolvedDirectEdit;

impl BlendNamedTopology {
    pub(in crate::prism::named) fn direct_edit_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        resolved: &ResolvedDirectEdit,
        achieved: Option<PrismAchievedTolerance>,
    ) -> Result<BlendCandidate, String> {
        let input = self.source_topology(history, document)?;
        super::super::direct_edit::build_direct_edit(
            super::super::BlendInputTopology {
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
            resolved,
            achieved,
        )
    }
}
