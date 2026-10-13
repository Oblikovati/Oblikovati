// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::{Error, json::EmptyObject, wire::*};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
/// The entity kind required by an authored asset override.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum TopologyEntityKind {
    Body,
    Face,
}
/// Injected native ownership boundary; the automation crate never constructs geometry.
pub trait PartTopologyAccess<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> {
    /// Enumerate current native bodies; example: `access.list_bodies(files, document)?`.
    fn list_bodies(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<BodyInfo>, String>;
    /// Verify a current exact target before any edit; example: `access.verify(files, document, key, TopologyEntityKind::Face)?`.
    fn verify(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        key: &[u8],
        kind: TopologyEntityKind,
    ) -> Result<(), String>;
}

/// Native body queries share the exact reference owner used by assignments.
pub const BODY_QUERY_METHODS: [&str; 1] = [METHOD_BODY_LIST];
pub(crate) fn list<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let _: EmptyObject = crate::decode(METHOD_BODY_LIST, encoded)?;
    let document = crate::part_owner::active_part(files)?;
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "body query document {document}: expected an injected exact topology owner"
        ))
    })?;
    let bodies = owner.list_bodies(files, document).map_err(Error)?;
    crate::encode(&BodyListResult {
        bodies: Some(bodies),
    })
}
