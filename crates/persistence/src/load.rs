// SPDX-License-Identifier: GPL-2.0-only

use crate::package::{DocumentPackage, PackageCodecError};
use oblikovati_model::identity::{
    DocumentIdentityIndex, FileIdentity, GuidSource, IdentityRecordError, IdentityRegistration,
    IdentityRegistrationError, IdentityTransitionError,
};
use std::{
    error::Error,
    fmt, io,
    path::{Path, PathBuf},
};

mod parameters;
pub use parameters::{
    LoadedParameterPackage, PreparedParameterPackage, load_parameter_package,
    read_parameter_package,
};

/// Injected whole-file input; the package codec owns validation after reading.
pub trait PackageFileReader {
    /// Read a snapshot without retaining a file handle. Example: `reader.read(path)?`.
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>>;
}

/// Native buffered package reads.
pub struct NativePackageReader;

impl PackageFileReader for NativePackageReader {
    fn read(&mut self, path: &Path) -> io::Result<Vec<u8>> {
        std::fs::read(path)
    }
}

/// A loaded snapshot and the identity outcome the host must process.
#[must_use]
#[derive(Debug)]
pub struct LoadedPackage {
    pub package: DocumentPackage,
    pub registration: IdentityRegistration,
}

/// A load failure with the requested file path and its specific cause.
#[derive(Debug)]
pub struct PackageLoadError {
    pub path: PathBuf,
    cause: LoadFailure,
}

#[derive(Debug)]
enum LoadFailure {
    Read(io::Error),
    Codec(PackageCodecError),
    ScalarOwnership(oblikovati_model::feature::FeatureScalarOwnershipError),
    Generation(IdentityTransitionError),
    Identity(IdentityRecordError),
    Registration(IdentityRegistrationError),
}

/// Load and validate a file before claiming its persistent identity in the session.
/// Recipe interpretation remains the host's next step; this does not execute it.
///
/// ```no_run
/// use oblikovati_model::identity::{DocumentIdentityIndex, SystemGuidSource};
/// use oblikovati_persistence::load::{load_registered, NativePackageReader};
/// let mut index = DocumentIdentityIndex::default();
/// let loaded = load_registered(std::path::Path::new("part.obk"), 1, "alpha",
///     &mut index, &mut SystemGuidSource, &mut NativePackageReader)?;
/// # Ok::<(), oblikovati_persistence::load::PackageLoadError>(())
/// ```
pub fn load_registered(
    path: &Path,
    session_id: u64,
    version: &str,
    index: &mut DocumentIdentityIndex,
    source: &mut dyn GuidSource,
    reader: &mut dyn PackageFileReader,
) -> Result<LoadedPackage, PackageLoadError> {
    let package = read_package(path, reader)?;
    register_package(path, session_id, version, index, source, package)
}

/// Read and validate a package envelope without registering or restoring its document identity.
/// Example: `let template = read_package(path, &mut reader)?;`.
pub fn read_package(
    path: &Path,
    reader: &mut dyn PackageFileReader,
) -> Result<DocumentPackage, PackageLoadError> {
    let encoded = reader
        .read(path)
        .map_err(|error| load_error(path, LoadFailure::Read(error)))?;
    DocumentPackage::decode(&encoded).map_err(|error| load_error(path, LoadFailure::Codec(error)))
}

fn register_package(
    path: &Path,
    session_id: u64,
    version: &str,
    index: &mut DocumentIdentityIndex,
    source: &mut dyn GuidSource,
    package: DocumentPackage,
) -> Result<LoadedPackage, PackageLoadError> {
    let identity =
        restore_identity(&package, version, source).map_err(|cause| load_error(path, cause))?;
    let registration = index
        .register(session_id, identity, source)
        .map_err(|error| load_error(path, LoadFailure::Registration(error)))?;
    Ok(LoadedPackage {
        package,
        registration,
    })
}

fn load_error(path: &Path, cause: LoadFailure) -> PackageLoadError {
    PackageLoadError {
        path: path.to_owned(),
        cause,
    }
}

fn restore_identity(
    package: &DocumentPackage,
    version: &str,
    source: &mut dyn GuidSource,
) -> Result<FileIdentity, LoadFailure> {
    let record = package.identity_record().map_err(LoadFailure::Codec)?;
    let fresh = FileIdentity::new(version, source).map_err(LoadFailure::Generation)?;
    match record {
        Some(record) => record.restore(&fresh).map_err(LoadFailure::Identity),
        None => Ok(fresh),
    }
}

impl fmt::Display for PackageLoadError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            formatter,
            "load {:?}: expected a readable valid schema-2 package: {}",
            self.path, self.cause
        )
    }
}

impl Error for PackageLoadError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        Some(&self.cause)
    }
}

impl fmt::Display for LoadFailure {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Read(error) => error.fmt(formatter),
            Self::Codec(error) => error.fmt(formatter),
            Self::ScalarOwnership(error) => error.fmt(formatter),
            Self::Generation(error) => error.fmt(formatter),
            Self::Identity(error) => error.fmt(formatter),
            Self::Registration(error) => error.fmt(formatter),
        }
    }
}

impl Error for LoadFailure {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Read(error) => Some(error),
            Self::Codec(error) => Some(error),
            Self::ScalarOwnership(error) => Some(error),
            Self::Generation(error) => Some(error),
            Self::Identity(error) => Some(error),
            Self::Registration(error) => Some(error),
        }
    }
}
