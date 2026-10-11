// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, PackageCodecError, PackageSaveError, PackageSaveTarget};
use crate::atomic::AtomicFiles;
use oblikovati_model::{
    history::{DocumentSnapshot, DocumentSnapshotTarget},
    identity::{DocumentSaveTarget, FileIdentity, RecipeDigest},
};
use std::{error::Error, fmt, path::Path};

/// Prepare a live parameter/property recipe while retaining all other package sections.
pub struct ParameterPackageTarget<'a, Files> {
    original: &'a mut DocumentPackage,
    destination: &'a Path,
    files: &'a mut Files,
    prepared: Option<DocumentPackage>,
}

/// A snapshot could not be captured or atomically persisted.
#[derive(Debug)]
pub enum SnapshotPackageError {
    Codec(PackageCodecError),
    Save(PackageSaveError),
    Unprepared,
    StaleProperties,
}

impl<'a, Files: AtomicFiles> ParameterPackageTarget<'a, Files> {
    /// Bind the retained package to a save destination; example: `ParameterPackageTarget::new(package, path, files)`.
    pub fn new(
        original: &'a mut DocumentPackage,
        destination: &'a Path,
        files: &'a mut Files,
    ) -> Self {
        Self {
            original,
            destination,
            files,
            prepared: None,
        }
    }
}

impl<Files: AtomicFiles> DocumentSnapshotTarget for ParameterPackageTarget<'_, Files> {
    fn prepare(
        &mut self,
        snapshot: &DocumentSnapshot,
        derived_high_water: i64,
        parameter_high_water: u64,
        sketch_marks: oblikovati_model::history::SketchReservationMarks,
        feature_high_water: u64,
    ) -> Result<Option<RecipeDigest>, Self::Failure> {
        self.prepared = None;
        snapshot
            .validate_scalar_ownership()
            .map_err(|error| SnapshotPackageError::Codec(PackageCodecError(error.to_string())))?;
        let candidate = self
            .original
            .with_work_geometry(&snapshot.work_geometry, feature_high_water)
            .map_err(SnapshotPackageError::Codec)?
            .with_sketches(&snapshot.sketches, sketch_marks.sketch, sketch_marks.entity)
            .map_err(SnapshotPackageError::Codec)?
            .with_attachments(&snapshot.attachments)
            .map_err(SnapshotPackageError::Codec)?
            .with_resources(&snapshot.resources)
            .with_interests(&snapshot.interests)
            .map_err(SnapshotPackageError::Codec)?
            .with_attributes(&snapshot.attributes)
            .map_err(SnapshotPackageError::Codec)?
            .with_derived_reservations(derived_high_water)
            .map_err(SnapshotPackageError::Codec)?
            .with_parameter_properties(
                &snapshot.parameters,
                &snapshot.properties,
                snapshot.parameters.units().clone(),
            )
            .map_err(SnapshotPackageError::Codec)?
            .with_parameter_reservations(&snapshot.parameters, parameter_high_water)
            .map_err(SnapshotPackageError::Codec)?;
        let restored = candidate
            .restore_parameter_properties(&snapshot.parameters)
            .map_err(SnapshotPackageError::Codec)?;
        if !restored.same_recipe(&snapshot.properties) {
            return Err(SnapshotPackageError::StaleProperties);
        }
        let digest = candidate
            .recipe_digest()
            .map_err(SnapshotPackageError::Codec)?;
        self.prepared = Some(candidate);
        Ok(digest)
    }
}

impl<Files: AtomicFiles> DocumentSaveTarget for ParameterPackageTarget<'_, Files> {
    type Failure = SnapshotPackageError;

    fn persist(&mut self, identity: &FileIdentity) -> Result<(), Self::Failure> {
        let mut candidate = self
            .prepared
            .take()
            .ok_or(SnapshotPackageError::Unprepared)?;
        PackageSaveTarget::new(&mut candidate, self.destination, self.files)
            .persist(identity)
            .map_err(SnapshotPackageError::Save)?;
        *self.original = candidate;
        Ok(())
    }
}

impl fmt::Display for SnapshotPackageError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Codec(error) => error.fmt(output),
            Self::Save(error) => error.fmt(output),
            Self::Unprepared => write!(
                output,
                "save candidate is absent: expected successful snapshot preparation"
            ),
            Self::StaleProperties => write!(
                output,
                "snapshot properties differ from parameter projections: expected synchronized properties before save"
            ),
        }
    }
}
impl Error for SnapshotPackageError {
    fn source(&self) -> Option<&(dyn Error + 'static)> {
        match self {
            Self::Codec(error) => Some(error),
            Self::Save(error) => Some(error),
            _ => None,
        }
    }
}
