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
pub trait PartTopologyAccess<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>:
    crate::PartUpdateAccess<O, W, F>
{
    /// Identify the code and dependencies that produce persisted assembly views.
    /// Example: `access.assembly_view_producer()?` before staging a cache.
    fn assembly_view_producer(&mut self) -> Result<String, String>;

    /// Present an assembly without transient selection or visibility.
    /// Example: `access.assembly_view(files, document)?` before qualifying its cache.
    fn assembly_view(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<
        (
            Vec<oblikovati_persistence::package::SavedAssemblyTriangle>,
            f64,
            f64,
        ),
        String,
    >;
    /// Enumerate current native bodies; example: `access.list_bodies(files, document)?`.
    fn list_bodies(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<Vec<BodyInfo>, String>;
    /// List exact named topology; example: `access.reference_keys(files, document)?`.
    fn reference_keys(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<ReferenceKeysResult, String>;
    /// Query an indexed body or the complete current part at an explicit density; example: `access.physical_properties(files, document, Some(0), 7.85)?`.
    fn physical_properties(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        index: Option<i64>,
        density: f64,
    ) -> Result<MassPropertiesResult, String>;
    /// Integrate a named face of the selected body; example: `access.face_area(files, document, 0, &key)?`.
    fn face_area(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        index: i64,
        key: &[u8],
    ) -> Result<f64, String>;
    /// Integrate a named edge of the selected body; example: `access.edge_length(files, document, 0, &key)?`.
    fn edge_length(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
        index: i64,
        key: &[u8],
    ) -> Result<f64, String>;
    /// Read current replay health; example: `access.model_health(files, document)?`.
    fn model_health(
        &mut self,
        files: &mut FileWorkspace<O, W, F>,
        document: u64,
    ) -> Result<ModelHealthResult, String>;
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
pub const BODY_QUERY_METHODS: [&str; 7] = [
    METHOD_MODEL_PHYSICAL_PROPERTIES,
    METHOD_ANALYSIS_MODEL_HEALTH,
    METHOD_MODEL_REFERENCE_KEYS,
    METHOD_BODY_LIST,
    METHOD_BODY_PHYSICAL_PROPS,
    METHOD_ANALYSIS_MASS_PROPERTIES,
    METHOD_ANALYSIS_MEASURE,
];
mod measure;
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
        METHOD_MODEL_PHYSICAL_PROPERTIES => {
            let density = physical::model_density(files, encoded)?;
            let result = owner
                .physical_properties(files, document, None, density)
                .map_err(Error)?;
            crate::encode(&physical::model_properties(result)?)
        }
        METHOD_ANALYSIS_MODEL_HEALTH => {
            let _: ModelHealthArgs = crate::decode(method, encoded)?;
            crate::encode(&owner.model_health(files, document).map_err(Error)?)
        }
        METHOD_MODEL_REFERENCE_KEYS => {
            if !encoded.is_empty() {
                let _: EmptyObject = crate::decode(method, encoded)?;
            }
            crate::encode(&owner.reference_keys(files, document).map_err(Error)?)
        }
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
        METHOD_ANALYSIS_MEASURE => measure::query(files, owner, document, encoded),
        _ => Err(Error(format!(
            "body query {method:?}: expected {BODY_QUERY_METHODS:?}"
        ))),
    }
}

pub(super) fn no_arguments(method: &str, encoded: &[u8]) -> Result<(), Error> {
    if !encoded.is_empty() {
        let _: EmptyObject = crate::decode(method, encoded)?;
    }
    Ok(())
}
