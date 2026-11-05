// SPDX-License-Identifier: GPL-2.0-only
mod reference;
mod resolve;
pub use reference::AssignmentReference;
use std::collections::BTreeMap;
/// Assignments belong to one document; asset definitions remain in their own scopes.
#[derive(Clone, Debug, Default, PartialEq, Eq)]
pub struct AssetAssignments {
    part_material: Option<String>,
    part_appearance: Option<String>,
    body_material: BTreeMap<AssignmentReference, String>,
    body_appearance: BTreeMap<AssignmentReference, String>,
    face_appearance: BTreeMap<AssignmentReference, String>,
}
impl AssetAssignments {
    /// Set or clear physical part intent; example: `assignments.set_part_material("steel")?`.
    pub fn set_part_material(&mut self, id: &str) -> Result<(), String> {
        self.part_material = assignment_asset_id(id)?;
        Ok(())
    }
    /// Set or clear explicit part visual intent; example: `assignments.set_part_appearance("oak")?`.
    pub fn set_part_appearance(&mut self, id: &str) -> Result<(), String> {
        self.part_appearance = assignment_asset_id(id)?;
        Ok(())
    }
    /// Set or clear one occurrence's physical override; example: `assignments.set_body_material(&body, "steel")?`.
    pub fn set_body_material(
        &mut self,
        target: &AssignmentReference,
        id: &str,
    ) -> Result<(), String> {
        set_entity_asset(&mut self.body_material, target, id)
    }
    /// Set or clear one occurrence's visual override; example: `assignments.set_body_appearance(&body, "oak")?`.
    pub fn set_body_appearance(
        &mut self,
        target: &AssignmentReference,
        id: &str,
    ) -> Result<(), String> {
        set_entity_asset(&mut self.body_appearance, target, id)
    }
    /// Set or clear one occurrence's face override; example: `assignments.set_face_appearance(&face, "abs-black")?`.
    pub fn set_face_appearance(
        &mut self,
        target: &AssignmentReference,
        id: &str,
    ) -> Result<(), String> {
        set_entity_asset(&mut self.face_appearance, target, id)
    }
    /// Borrow raw part physical intent; example: `assignments.part_material_id()` for storage.
    pub fn part_material_id(&self) -> Option<&str> {
        self.part_material.as_deref()
    }
    /// Borrow raw part visual intent; example: `assignments.part_appearance_id()` for storage.
    pub fn part_appearance_id(&self) -> Option<&str> {
        self.part_appearance.as_deref()
    }
    /// Read body physical overrides in reference order; example: `assignments.body_materials()` for storage.
    pub fn body_materials(&self) -> impl Iterator<Item = (&AssignmentReference, &str)> {
        self.body_material
            .iter()
            .map(|(key, id)| (key, id.as_str()))
    }
    /// Read body visual overrides in reference order; example: `assignments.body_appearances()` for storage.
    pub fn body_appearances(&self) -> impl Iterator<Item = (&AssignmentReference, &str)> {
        self.body_appearance
            .iter()
            .map(|(key, id)| (key, id.as_str()))
    }
    /// Read face visual overrides in reference order; example: `assignments.face_appearances()` for storage.
    pub fn face_appearances(&self) -> impl Iterator<Item = (&AssignmentReference, &str)> {
        self.face_appearance
            .iter()
            .map(|(key, id)| (key, id.as_str()))
    }
    /// Read effective physical intent without resolving properties; example: `assignments.effective_material_id(&body)`.
    pub fn effective_material_id(&self, body: &AssignmentReference) -> Option<&str> {
        self.body_material
            .get(body)
            .or(self.part_material.as_ref())
            .map(String::as_str)
    }
}
fn assignment_asset_id(id: &str) -> Result<Option<String>, String> {
    if id.contains('\0') {
        return Err(format!(
            "asset assignment id {id:?}: expected text without NUL, or empty text to clear"
        ));
    }
    Ok((!id.is_empty()).then(|| id.into()))
}
fn set_entity_asset(
    assignments: &mut BTreeMap<AssignmentReference, String>,
    target: &AssignmentReference,
    id: &str,
) -> Result<(), String> {
    match assignment_asset_id(id)? {
        Some(id) => {
            assignments.insert(target.clone(), id);
        }
        None => {
            assignments.remove(target);
        }
    }
    Ok(())
}
