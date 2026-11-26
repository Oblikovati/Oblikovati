// SPDX-License-Identifier: GPL-2.0-only
//! Carry exact lineage into an inward Shell operation (#4106).
use super::*;

impl BlendNamedTopology {
    /// Build a validated uniform inward shell; example: `body.shell_candidate(&history, document, &faces, 0.1)?`.
    pub fn shell_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        openings: &[Vec<u8>],
        thickness: f64,
    ) -> Result<BlendCandidate, String> {
        let input = self.source_topology(history, document)?;
        super::super::shell::build_shell_candidate(
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
            openings,
            thickness,
        )
    }
}
