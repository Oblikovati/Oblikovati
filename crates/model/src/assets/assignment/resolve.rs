// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::assets::{AppearanceAsset, DEFAULT_APPEARANCE_ID, MaterialAsset, ScopedAssets};
use oblikovati_api::contract::Material;
impl AssetAssignments {
    /// Resolve the selected physical intent or report its missing identity; example: `assignments.resolve_material(&materials, &body)?`.
    pub fn resolve_material<'a>(
        &self,
        materials: &'a ScopedAssets<'_, MaterialAsset>,
        body: &AssignmentReference,
    ) -> Result<Option<&'a MaterialAsset>, String> {
        self.effective_material_id(body)
            .map(|id| materials.resolve(id))
            .transpose()
    }
    /// Resolve one chosen visual intent; example: `assignments.resolve_appearance(&visuals, &materials, &body, Some(&face))?`.
    pub fn resolve_appearance<'a>(
        &self,
        appearances: &'a ScopedAssets<'_, AppearanceAsset>,
        materials: &ScopedAssets<'_, MaterialAsset>,
        body: &AssignmentReference,
        face: Option<&AssignmentReference>,
    ) -> Result<&'a AppearanceAsset, String> {
        require_same_occurrence(body, face)?;
        // Go #1103 / ADR-0019: explicit part appearance wins over even an overridden body material.
        if let Some(id) = self.explicit_appearance_id(body, face) {
            return appearances.resolve(id);
        }
        let id = self.material_appearance_id(materials, body)?;
        appearances.resolve(&id)
    }
    fn explicit_appearance_id(
        &self,
        body: &AssignmentReference,
        face: Option<&AssignmentReference>,
    ) -> Option<&str> {
        [
            face.and_then(|target| self.face_appearance.get(target)),
            self.body_appearance.get(body),
            self.part_appearance.as_ref(),
        ]
        .into_iter()
        .flatten()
        .next()
        .map(String::as_str)
    }
    fn material_appearance_id(
        &self,
        materials: &ScopedAssets<'_, MaterialAsset>,
        body: &AssignmentReference,
    ) -> Result<String, String> {
        let id = self
            .resolve_material(materials, body)?
            .map(|asset| asset.appearance_id())
            .unwrap_or_default();
        Ok(if id.is_empty() {
            DEFAULT_APPEARANCE_ID.into()
        } else {
            id
        })
    }
}
fn require_same_occurrence(
    body: &AssignmentReference,
    face: Option<&AssignmentReference>,
) -> Result<(), String> {
    if let Some(face) = face
        && body.occurrence_path() != face.occurrence_path()
    {
        return Err(format!(
            "face occurrence {:?}, body occurrence {:?}: expected the same occurrence context",
            face.occurrence_path(),
            body.occurrence_path()
        ));
    }
    Ok(())
}
