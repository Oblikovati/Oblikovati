// SPDX-License-Identifier: GPL-2.0-only
//! Carry named input lineage into a detached Hole cut (#4093).
use super::*;

impl BlendNamedTopology {
    pub(in crate::prism::named) fn achieved_tolerance(&self) -> Option<PrismAchievedTolerance> {
        self.achieved_tolerance
    }
    pub(in crate::prism::named) fn hole_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        request: &super::super::HoleRequest<'_>,
    ) -> Result<BlendCandidate, String> {
        let input = self.source_topology(history, document)?;
        super::super::hole::build_hole_candidate(
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
            request,
        )
    }
}
