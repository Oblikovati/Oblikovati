// SPDX-License-Identifier: GPL-2.0-only

use super::{
    DocumentPackage, PackageCodecError, PackageSaveError, PackageSaveTarget, SavedAssemblyView,
    SavedDrawingViews,
};
use crate::atomic::AtomicFiles;
use oblikovati_model::{
    history::{DocumentSnapshot, DocumentSnapshotTarget},
    identity::{DocumentSaveTarget, FileIdentity, RecipeDigest},
};
use std::collections::BTreeSet;
use std::{error::Error, fmt, path::Path};

/// Prepare a live parameter/property recipe while retaining all other package sections.
pub struct ParameterPackageTarget<'a, Files> {
    original: &'a mut DocumentPackage,
    destination: &'a Path,
    files: &'a mut Files,
    prepared: Option<DocumentPackage>,
    display_name: Option<String>,
    drawing_views: Option<SavedDrawingViews>,
    assembly_view: Option<SavedAssemblyView>,
    assembly_references: Option<BTreeSet<String>>,
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
            display_name: None,
            drawing_views: None,
            assembly_view: None,
            assembly_references: None,
        }
    }
    /// Supply references verified by the document owner; example: `target.with_assembly_references(names)`.
    pub fn with_assembly_references(mut self, names: BTreeSet<String>) -> Self {
        self.assembly_references = Some(names);
        self.prepared = None;
        self
    }
    /// Add saved drawing presentation to the same atomic candidate; example: `target.with_drawing_views(views)`.
    pub fn with_drawing_views(mut self, views: Option<SavedDrawingViews>) -> Self {
        self.drawing_views = views;
        self.prepared = None;
        self
    }
    /// Add a validated assembly presentation to the same atomic save candidate.
    /// Example: `target.with_assembly_view(Some(view))`.
    pub fn with_assembly_view(mut self, view: Option<SavedAssemblyView>) -> Self {
        self.assembly_view = view;
        self.prepared = None;
        self
    }
    /// Stage the current title without mutating the retained package; example: `target.with_display_name("Bracket".into())` before preparation.
    pub fn with_display_name(mut self, name: String) -> Self {
        self.display_name = Some(name);
        self.prepared = None;
        self
    }
    fn named_candidate(&self, candidate: DocumentPackage) -> DocumentPackage {
        match &self.display_name {
            Some(name) => candidate.with_display_name(name),
            None => candidate,
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
        let candidate = capture_candidate(
            self.original,
            snapshot,
            derived_high_water,
            parameter_high_water,
            sketch_marks,
            feature_high_water,
            self.assembly_references.as_ref(),
        )?;
        let candidate = self.named_candidate(candidate);
        let candidate = match &self.drawing_views {
            Some(views) => candidate
                .with_saved_drawing_views(views)
                .map_err(SnapshotPackageError::Codec)?,
            None => candidate,
        };
        let candidate = if snapshot.assembly.is_some() {
            let clean = candidate.without_saved_assembly_view();
            match &self.assembly_view {
                Some(view) => clean
                    .with_saved_assembly_view(view)
                    .map_err(SnapshotPackageError::Codec)?,
                None => clean,
            }
        } else {
            candidate
        };
        require_current_properties(&candidate, snapshot)?;
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

fn capture_candidate(
    original: &DocumentPackage,
    snapshot: &DocumentSnapshot,
    derived_high_water: i64,
    parameter_high_water: u64,
    sketch_marks: oblikovati_model::history::SketchReservationMarks,
    feature_high_water: u64,
    assembly_references: Option<&BTreeSet<String>>,
) -> Result<DocumentPackage, SnapshotPackageError> {
    snapshot
        .validate_scalar_ownership()
        .map_err(|error| SnapshotPackageError::Codec(PackageCodecError(error.to_string())))?;
    capture_nonparameter_sections(
        original,
        snapshot,
        sketch_marks,
        feature_high_water,
        assembly_references,
    )?
    .with_derived_reservations(derived_high_water)
    .map_err(SnapshotPackageError::Codec)?
    .with_parameter_properties(
        &snapshot.parameters,
        &snapshot.properties,
        snapshot.parameters.units().clone(),
    )
    .map_err(SnapshotPackageError::Codec)?
    .with_parameter_reservations(&snapshot.parameters, parameter_high_water)
    .map_err(SnapshotPackageError::Codec)
}
fn capture_nonparameter_sections(
    original: &DocumentPackage,
    snapshot: &DocumentSnapshot,
    sketch_marks: oblikovati_model::history::SketchReservationMarks,
    feature_high_water: u64,
    assembly_references: Option<&BTreeSet<String>>,
) -> Result<DocumentPackage, SnapshotPackageError> {
    if snapshot.part_design_views.is_some() {
        let mut ids = oblikovati_model::identity::RecipeIdAllocator::default();
        original
            .restore_part_design_views(&mut ids)
            .map_err(SnapshotPackageError::Codec)?;
        if !snapshot.design_views.is_empty() {
            return Err(SnapshotPackageError::Codec(PackageCodecError(format!(
                "legacy designViews {:?}: expected one part view source",
                snapshot.design_views
            ))));
        }
    }
    let candidate = match &snapshot.assembly {
        Some(occurrences) => match assembly_references {
            Some(names) => original.with_owned_assembly_occurrences(occurrences, names),
            None => original.with_assembly_occurrences(occurrences),
        }
        .map_err(SnapshotPackageError::Codec)?,
        None => original.clone(),
    };
    candidate
        .with_model_states(snapshot.model_states.as_ref())
        .map_err(SnapshotPackageError::Codec)?
        .with_drawing(snapshot.drawing.as_ref())
        .map_err(SnapshotPackageError::Codec)?
        .with_work_geometry(&snapshot.work_geometry, feature_high_water)
        .map_err(SnapshotPackageError::Codec)?
        .with_sketch_reservations(&snapshot.sketches, sketch_marks)
        .map_err(SnapshotPackageError::Codec)?
        .with_body_names(&snapshot.body_names)
        .map_err(SnapshotPackageError::Codec)?
        .with_design_views(&snapshot.design_views)
        .map_err(SnapshotPackageError::Codec)?
        .with_part_design_views(snapshot.part_design_views.as_ref())
        .map_err(SnapshotPackageError::Codec)?
        .with_assets(&snapshot.assets)
        .map_err(SnapshotPackageError::Codec)?
        .with_sketch_settings(snapshot.sketch_settings.as_ref())
        .map_err(SnapshotPackageError::Codec)?
        .with_attachments(&snapshot.attachments)
        .map_err(SnapshotPackageError::Codec)?
        .with_resources(&snapshot.resources)
        .with_interests(&snapshot.interests)
        .map_err(SnapshotPackageError::Codec)?
        .with_attributes(&snapshot.attributes)
        .map_err(SnapshotPackageError::Codec)
}
fn require_current_properties(
    candidate: &DocumentPackage,
    snapshot: &DocumentSnapshot,
) -> Result<(), SnapshotPackageError> {
    let restored = candidate
        .restore_parameter_properties(&snapshot.parameters)
        .map_err(SnapshotPackageError::Codec)?;
    if !restored.same_recipe(&snapshot.properties) {
        return Err(SnapshotPackageError::StaleProperties);
    }
    Ok(())
}
