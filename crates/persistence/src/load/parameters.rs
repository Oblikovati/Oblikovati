// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::package::RestoredParameters;

/// A registered package with typed parameters, placed sketches and document properties.
/// Other retained model domains are not executed by this loader.
#[must_use]
#[derive(Debug)]
pub struct LoadedParameterPackage {
    pub loaded: LoadedPackage,
    pub restored: RestoredParameters,
    pub sketches: crate::package::RestoredDocumentSketches,
    pub attributes: oblikovati_model::attribute::DocumentAttributes,
    pub attachments: oblikovati_model::attachment::DocumentAttachments,
    pub interests: oblikovati_model::interest::DocumentInterests,
    pub resources: std::collections::BTreeMap<String, oblikovati_model::resource::DocumentResource>,
    pub properties: oblikovati_model::property::DocumentProperties,
}

/// Reconstruct typed recipe state before claiming an open-document identity.
/// Example: `load_parameter_package(path, 1, "alpha", &mut index, &mut source, &mut reader)?;`.
pub fn load_parameter_package(
    path: &Path,
    session_id: u64,
    version: &str,
    index: &mut DocumentIdentityIndex,
    source: &mut dyn GuidSource,
    reader: &mut dyn PackageFileReader,
) -> Result<LoadedParameterPackage, PackageLoadError> {
    let prepared = read_parameter_package(path, version, source, reader)?;
    let registration = index
        .register(session_id, prepared.identity, source)
        .map_err(|error| load_error(path, LoadFailure::Registration(error)))?;
    Ok(LoadedParameterPackage {
        loaded: LoadedPackage {
            package: prepared.package,
            registration,
        },
        restored: prepared.restored,
        sketches: prepared.sketches,
        properties: prepared.properties,
        attributes: prepared.attributes,
        attachments: prepared.attachments,
        interests: prepared.interests,
        resources: prepared.resources,
    })
}

/// Reconstructed state without an open-document identity claim.
#[must_use]
#[derive(Debug)]
pub struct PreparedParameterPackage {
    pub package: DocumentPackage,
    pub restored: RestoredParameters,
    pub sketches: crate::package::RestoredDocumentSketches,
    pub attributes: oblikovati_model::attribute::DocumentAttributes,
    pub attachments: oblikovati_model::attachment::DocumentAttachments,
    pub interests: oblikovati_model::interest::DocumentInterests,
    pub resources: std::collections::BTreeMap<String, oblikovati_model::resource::DocumentResource>,
    pub properties: oblikovati_model::property::DocumentProperties,
    pub identity: FileIdentity,
}

/// Validate typed recipe state before a workspace takes ownership.
/// Example: `let prepared = read_parameter_package(path, "alpha", source, reader)?;`.
pub fn read_parameter_package(
    path: &Path,
    version: &str,
    source: &mut dyn GuidSource,
    reader: &mut dyn PackageFileReader,
) -> Result<PreparedParameterPackage, PackageLoadError> {
    let package = read_package(path, reader)?;
    let sketches = package
        .restore_sketches()
        .map_err(|error| load_error(path, LoadFailure::Codec(error)))?;
    let attachments = package
        .restore_attachments()
        .map_err(|error| load_error(path, LoadFailure::Codec(error)))?;
    let interests = package
        .restore_interests()
        .map_err(|error| load_error(path, LoadFailure::Codec(error)))?;
    let attributes = package
        .restore_attributes()
        .map_err(|error| load_error(path, LoadFailure::Codec(error)))?;
    let restored = package
        .restore_document_parameters()
        .map_err(|error| load_error(path, LoadFailure::Codec(error)))?;
    let properties = package
        .restore_parameter_properties(&restored.parameters)
        .map_err(|error| load_error(path, LoadFailure::Codec(error)))?;
    let identity =
        restore_identity(&package, version, source).map_err(|cause| load_error(path, cause))?;
    let resources = package.resources().clone();
    Ok(PreparedParameterPackage {
        package,
        restored,
        sketches,
        attributes,
        attachments,
        interests,
        resources,
        properties,
        identity,
    })
}
