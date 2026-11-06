// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::assets::{AssignmentReference, BuiltinAssetCatalog, ProjectAssetLibrary, ScopedAssets};

/// Authored physical target; the host must verify body identity before applying it.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum MaterialAssignmentTarget {
    Part,
    Body(AssignmentReference),
}
/// Authored visual target; the host must verify entity identity and occurrence context.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum AppearanceAssignmentTarget {
    Part,
    Body(AssignmentReference),
    Face(AssignmentReference),
}
impl DocumentAssets {
    /// Prepare portable physical intent without changing the source; example: `assets.with_material_assignment(project, catalog, &MaterialAssignmentTarget::Part, "steel")?`.
    pub fn with_material_assignment(
        &self,
        project: &ProjectAssetLibrary,
        catalog: &BuiltinAssetCatalog,
        target: &MaterialAssignmentTarget,
        id: &str,
    ) -> Result<Self, String> {
        let mut candidate = self.clone();
        candidate.retain_assignment_material(self, project, catalog, id)?;
        target.apply(&mut candidate.assignments, id)?;
        Ok(candidate)
    }
    /// Prepare portable visual intent without changing physical intent; example: `assets.with_appearance_assignment(project, catalog, &AppearanceAssignmentTarget::Part, "steel")?`.
    pub fn with_appearance_assignment(
        &self,
        project: &ProjectAssetLibrary,
        catalog: &BuiltinAssetCatalog,
        target: &AppearanceAssignmentTarget,
        id: &str,
    ) -> Result<Self, String> {
        let mut candidate = self.clone();
        candidate.retain_assignment_appearance(self, project, catalog, id)?;
        target.apply(&mut candidate.assignments, id)?;
        Ok(candidate)
    }
    fn retain_assignment_material(
        &mut self,
        original: &DocumentAssets,
        project: &ProjectAssetLibrary,
        catalog: &BuiltinAssetCatalog,
        id: &str,
    ) -> Result<(), String> {
        if id.is_empty() {
            return Ok(());
        }
        let materials = assignment_scope(
            original.materials(),
            project.materials(),
            catalog.materials(),
        );
        let material = materials.resolve(id)?;
        self.retain_assignment_appearance(original, project, catalog, &material.appearance_id())?;
        self.embed_material(material)?;
        Ok(())
    }
    fn retain_assignment_appearance(
        &mut self,
        original: &DocumentAssets,
        project: &ProjectAssetLibrary,
        catalog: &BuiltinAssetCatalog,
        id: &str,
    ) -> Result<(), String> {
        if id.is_empty() {
            return Ok(());
        }
        let appearances = assignment_scope(
            original.appearances(),
            project.appearances(),
            catalog.appearances(),
        );
        self.embed_appearance(appearances.resolve(id)?)?;
        Ok(())
    }
}
impl MaterialAssignmentTarget {
    fn apply(
        &self,
        assignments: &mut super::super::AssetAssignments,
        id: &str,
    ) -> Result<(), String> {
        match self {
            Self::Part => assignments.set_part_material(id),
            Self::Body(reference) => assignments.set_body_material(reference, id),
        }
    }
}
impl AppearanceAssignmentTarget {
    fn apply(
        &self,
        assignments: &mut super::super::AssetAssignments,
        id: &str,
    ) -> Result<(), String> {
        match self {
            Self::Part => assignments.set_part_appearance(id),
            Self::Body(reference) => assignments.set_body_appearance(reference, id),
            Self::Face(reference) => assignments.set_face_appearance(reference, id),
        }
    }
}
fn assignment_scope<'a, T>(
    document: &'a AssetCollection<T>,
    project: &'a AssetCollection<T>,
    builtin: &'a AssetCollection<T>,
) -> ScopedAssets<'a, T> {
    ScopedAssets {
        document,
        project,
        builtin,
    }
}
