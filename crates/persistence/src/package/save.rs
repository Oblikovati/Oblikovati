// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, PackageCodecError, Value, codec_error};
use crate::atomic::{AtomicFiles, AtomicWriteError, replace_file};
use oblikovati_model::identity::{
    DocumentSaveTarget, FileIdentity, FileIdentityRecord, RecipeDigest,
};
use std::{error::Error, fmt, path::Path};

/// A complete package snapshot and destination, bound to an injected filesystem.
pub struct PackageSaveTarget<'snapshot, Files> {
    package: &'snapshot mut DocumentPackage,
    destination: &'snapshot Path,
    files: &'snapshot mut Files,
}

/// Encoding, digest agreement, or atomic replacement failed before adoption.
#[derive(Debug)]
pub enum PackageSaveError {
    Codec(PackageCodecError),
    DigestMismatch {
        supplied: Option<RecipeDigest>,
        expected: Option<RecipeDigest>,
    },
    Write(AtomicWriteError),
}

impl DocumentPackage {
    /// Read the persisted identity record for validation/restoration by the model.
    /// Example: `let record = package.identity_record()?;`.
    pub fn identity_record(&self) -> Result<Option<FileIdentityRecord>, PackageCodecError> {
        match self.root.get(Value::String("identity".into())) {
            None | Some(Value::Null) => Ok(None),
            Some(record) => serde_yaml_ng::from_value(record.clone())
                .map(Some)
                .map_err(codec_error),
        }
    }

    /// Fingerprint the exact recipe serialization used by this package encoder.
    /// Example: `let digest = package.recipe_digest()?;`.
    pub fn recipe_digest(&self) -> Result<Option<RecipeDigest>, PackageCodecError> {
        if matches!(
            self.root.get(Value::String("model".into())),
            None | Some(Value::Null)
        ) {
            return Ok(None);
        }
        self.section_yaml("model")
            .map(|recipe| recipe.map(|bytes| RecipeDigest::of_recipe(&bytes)))
    }

    fn encode_with_identity(
        &self,
        identity: &FileIdentity,
    ) -> Result<(Value, Vec<u8>), PackageCodecError> {
        // Refuse unknown identity fields before replacing the record, so none are lost.
        if let Some(record) = self.identity_record()? {
            record
                .restore(identity)
                .map_err(|error| PackageCodecError(error.to_string()))?;
        }
        let record =
            serde_yaml_ng::to_value(FileIdentityRecord::from(identity)).map_err(codec_error)?;
        let mut root = self.root.clone();
        root.insert(Value::String("identity".into()), record.clone());
        let encoded = serde_yaml_ng::to_string(&root)
            .map(String::into_bytes)
            .map_err(codec_error)?;
        Ok((record, encoded))
    }
}

impl<'snapshot, Files: AtomicFiles> PackageSaveTarget<'snapshot, Files> {
    /// Bind a snapshot to its destination; all other sections remain unchanged.
    /// Example: `let target = PackageSaveTarget::new(&mut package, path, &mut files);`.
    pub fn new(
        package: &'snapshot mut DocumentPackage,
        destination: &'snapshot Path,
        files: &'snapshot mut Files,
    ) -> Self {
        Self {
            package,
            destination,
            files,
        }
    }
}

impl<Files: AtomicFiles> DocumentSaveTarget for PackageSaveTarget<'_, Files> {
    type Failure = PackageSaveError;

    fn persist(&mut self, candidate: &FileIdentity) -> Result<(), Self::Failure> {
        let expected = self
            .package
            .recipe_digest()
            .map_err(PackageSaveError::Codec)?;
        if candidate.model_digest != expected {
            return Err(PackageSaveError::DigestMismatch {
                supplied: candidate.model_digest,
                expected,
            });
        }
        let (record, encoded) = self
            .package
            .encode_with_identity(candidate)
            .map_err(PackageSaveError::Codec)?;
        replace_file(self.destination, &encoded, self.files).map_err(PackageSaveError::Write)?;
        self.package
            .root
            .insert(Value::String("identity".into()), record);
        Ok(())
    }
}

impl fmt::Display for PackageSaveError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Codec(error) => error.fmt(formatter),
            Self::DigestMismatch { supplied, expected } => write!(
                formatter,
                "candidate recipe digest {supplied:?}: expected snapshot digest {expected:?}"
            ),
            Self::Write(error) => error.fmt(formatter),
        }
    }
}

impl Error for PackageSaveError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Codec(error) => Some(error),
            Self::Write(error) => Some(error),
            Self::DigestMismatch { .. } => None,
        }
    }
}
