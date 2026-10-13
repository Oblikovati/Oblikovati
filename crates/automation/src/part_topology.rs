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
    /// Query an indexed body or the complete current part at an explicit density; example: `access.physical_properties(files, document, Some(0), 7.85)?`.
    fn physical_properties(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        index: Option<i64>,
        density: f64,
    ) -> Result<MassPropertiesResult, String>;
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
pub const BODY_QUERY_METHODS: [&str; 3] = [
    METHOD_BODY_LIST,
    METHOD_BODY_PHYSICAL_PROPS,
    METHOD_ANALYSIS_MASS_PROPERTIES,
];
mod physical;
pub(crate) fn query<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &mut FileWorkspace<O, W, F>,
    access: &mut Option<&mut dyn PartTopologyAccess<O, W, F>>,
    method: &str,
    encoded: &[u8],
) -> Result<Vec<u8>, Error> {
    let document = crate::part_owner::active_part(files)?;
    let owner = access.as_deref_mut().ok_or_else(|| {
        Error(format!(
            "body query document {document}: expected an injected exact topology owner"
        ))
    })?;
    match method {
        METHOD_BODY_LIST => {
            let _: EmptyObject = crate::decode(method, encoded)?;
            crate::encode(&BodyListResult {
                bodies: Some(owner.list_bodies(files, document).map_err(Error)?),
            })
        }
        METHOD_BODY_PHYSICAL_PROPS => {
            let request = physical::request(files, encoded)?;
            crate::encode(
                &owner
                    .physical_properties(
                        files,
                        document,
                        Some(request.body_index),
                        request.density_g_cm3,
                    )
                    .map_err(Error)?,
            )
        }
        METHOD_ANALYSIS_MASS_PROPERTIES => {
            let request = physical::mass_request(files, encoded)?;
            crate::encode(
                &owner
                    .physical_properties(files, document, None, request.density_g_cm3)
                    .map_err(Error)?,
            )
        }
        _ => Err(Error(format!(
            "body query {method:?}: expected {BODY_QUERY_METHODS:?}"
        ))),
    }
}
