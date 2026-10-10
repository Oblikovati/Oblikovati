// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::package::RestoredParameters;

/// A registered package with its parameter domain reconstructed in stored working units.
/// Other retained model domains are not executed by this loader.
#[must_use]
#[derive(Debug)]
pub struct LoadedParameterPackage {
    pub loaded: LoadedPackage,
    pub restored: RestoredParameters,
}

/// Load the implemented parameter domain before claiming an open-document identity.
/// Example: `load_parameter_package(path, 1, "alpha", &mut index, &mut source, &mut reader)?;`.
pub fn load_parameter_package(
    path: &Path,
    session_id: u64,
    version: &str,
    index: &mut DocumentIdentityIndex,
    source: &mut dyn GuidSource,
    reader: &mut dyn PackageFileReader,
) -> Result<LoadedParameterPackage, PackageLoadError> {
    let package = read_snapshot(path, reader)?;
    let restored = package
        .restore_document_parameters()
        .map_err(|error| load_error(path, LoadFailure::Codec(error)))?;
    let loaded = register_package(path, session_id, version, index, source, package)?;
    Ok(LoadedParameterPackage { loaded, restored })
}
