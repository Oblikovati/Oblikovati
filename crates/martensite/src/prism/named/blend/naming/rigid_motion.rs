// SPDX-License-Identifier: GPL-2.0-only
//! Retain named input lineage through a rigid motion (#4097).
use super::*;
use martensite_tkmath::gp::Transform3;
impl BlendNamedTopology {
    pub(in crate::prism::named) fn rigid_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        transform: Transform3,
        achieved: Option<PrismAchievedTolerance>,
    ) -> Result<BlendCandidate, String> {
        let input = self.source_topology(history, document)?;
        super::super::rigid_motion::build_rigid(
            super::super::BlendInputTopology {
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
            transform,
            achieved,
        )
    }
}
