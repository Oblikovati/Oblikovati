// SPDX-License-Identifier: GPL-2.0-only
//! Exact edge picking for the current part.
use super::*;

impl DesktopPartScenes {
    /// Draw one exact selected edge; example: `scenes.edge_outline(session, document, feature, key)?`.
    pub(crate) fn edge_outline<O: TransactionObserver>(
        &self,
        session: &oblikovati_session::DocumentSession<O>,
        document: oblikovati_model::identity::DocumentGuid,
        feature: oblikovati_model::feature::FeatureId,
        key: &[u8],
    ) -> Result<Vec<[[f64; 3]; 2]>, String> {
        let body = self
            .current(session)?
            .body(feature)
            .ok_or_else(|| format!("edge feature {feature:?}: expected a current body"))?;
        body.edge_display_segments(session.history(), document, key, DISPLAY_MESH.chord_error)
    }
    /// Pick a named edge by exact analytic distance to the view ray.
    /// Example: `scenes.edge_on_line(files, document, ray, tolerance)?`.
    #[cfg(test)]
    pub(crate) fn edge_on_line<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        line: [[f64; 3]; 2],
        tolerance: f64,
    ) -> Result<Option<(oblikovati_model::feature::FeatureId, Vec<u8>)>, String> {
        let mut candidates =
            pick_candidates::PickCandidates::new(pick_candidates::PickCoverage::NamedEdges);
        self.append_edge_candidates(files, document, line, tolerance, &mut candidates)?;
        match candidates.nearest()? {
            Some(pick_candidates::PickTarget::Edge(feature, key)) => Ok(Some((feature, key))),
            None => Ok(None),
            Some(other) => Err(format!("edge hit {other:?}: expected a named edge")),
        }
    }

    pub(crate) fn append_edge_candidates<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
        line: [[f64; 3]; 2],
        tolerance: f64,
        candidates: &mut pick_candidates::PickCandidates,
    ) -> Result<(), String> {
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        for record in self.current(session)?.health() {
            let Some(body) = self.current(session)?.body(record.feature) else {
                continue;
            };
            let key =
                bodies::body_reference_text(body.body_reference_key(session.history(), guid)?);
            if !session.body_visible(&key) {
                continue;
            }
            for hit in body.edge_hits_on_ray(session.history(), guid, line, tolerance)? {
                if hit.distance <= tolerance {
                    candidates.push(
                        pick_candidates::PickTarget::Edge(record.feature, hit.key),
                        hit.reach,
                    )?;
                }
            }
        }
        Ok(())
    }
}
