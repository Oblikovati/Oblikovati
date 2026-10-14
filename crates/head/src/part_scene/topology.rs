// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{BodyTopology, ReferenceKeysResult, TopologyRef};
use oblikovati_model::identity::ReferenceEnvelope;

impl DesktopPartScenes {
    pub(super) fn current_reference_keys<
        O: TransactionObserver,
        W: WorkspaceObserver,
        F: DocumentFiles,
    >(
        &self,
        files: &FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<ReferenceKeysResult, String> {
        let workspace = files.workspace();
        let session = workspace
            .session(document)
            .map_err(|cause| cause.to_string())?;
        let guid = workspace
            .identity(document)
            .map_err(|cause| cause.to_string())?
            .internal_name;
        let evaluation = self.current(session)?;
        let bodies = evaluation
            .ordered_bodies()
            .map(|body| topology(body, session.history(), guid))
            .collect::<Result<_, _>>()?;
        Ok(ReferenceKeysResult {
            bodies: Some(bodies),
        })
    }
}

fn topology(
    body: &NamedPrismSnapshot,
    history: &oblikovati_model::history::DocumentEditHistory,
    guid: oblikovati_model::identity::DocumentGuid,
) -> Result<BodyTopology, String> {
    let faces = body
        .reference_keys(history, guid)?
        .into_iter()
        .filter_map(face)
        .collect::<Result<Vec<_>, _>>()?;
    Ok(BodyTopology {
        faces: Some(faces),
        edges: None,
        vertices: None,
    })
}

fn face(key: Vec<u8>) -> Option<Result<TopologyRef, String>> {
    let envelope = ReferenceEnvelope::decode(&key).map_err(|cause| cause.to_string());
    match envelope {
        Ok(envelope) if envelope.kind == 1 => Some(Ok(TopologyRef {
            key: super::bodies::body_reference_text(&key),
            point: None,
            kind: "face".into(),
        })),
        Ok(_) => None,
        Err(cause) => Some(Err(cause)),
    }
}
