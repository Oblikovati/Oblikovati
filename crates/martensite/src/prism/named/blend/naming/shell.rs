// SPDX-License-Identifier: GPL-2.0-only
//! Carry exact lineage into directional and override Shell operations (#4106).
use super::*;
use oblikovati_model::feature::shell::{ResolvedShell, ShellDirection, ShellOptions};

impl BlendNamedTopology {
    /// Build a validated uniform inward shell; example: `body.shell_candidate(&history, document, &faces, 0.1)?`.
    pub fn shell_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        openings: &[Vec<u8>],
        thickness: f64,
    ) -> Result<BlendCandidate, String> {
        self.shell_v2_candidate(
            history,
            document,
            openings,
            &ResolvedShell {
                thickness,
                direction: ShellDirection::Inside,
                face_thicknesses: Vec::new(),
                options: ShellOptions::default(),
            },
        )
    }

    /// Build a validated directional/override shell candidate.
    pub(in crate::prism::named) fn shell_v2_candidate(
        &self,
        history: &DocumentEditHistory,
        document: DocumentGuid,
        openings: &[Vec<u8>],
        resolved: &ResolvedShell,
    ) -> Result<BlendCandidate, String> {
        let input = self.source_topology(history, document)?;
        super::super::shell::build_shell_v2_candidate(
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
            resolved,
        )
    }
}
