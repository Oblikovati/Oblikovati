// SPDX-License-Identifier: GPL-2.0-only
//! Part member recipes are retained independently from assembly representation tuples.
use super::*;
use oblikovati_api::types::DOCUMENT_PART;
use oblikovati_model::model_state::ModelStateMembers;
const SECTION: &str = "partModelStates";
impl DocumentPackage {
    /// Restore validated part members, supplying Primary for older part files.
    /// Example: `package.restore_model_states()?` before history construction.
    pub fn restore_model_states(&self) -> Result<Option<ModelStateMembers>, PackageCodecError> {
        if self.manifest.document_type != DOCUMENT_PART.0 {
            if self.root.contains_key(SECTION) {
                return Err(PackageCodecError(format!(
                    "document type {} with {SECTION}: expected a part document",
                    self.manifest.document_type
                )));
            }
            return Ok(None);
        }
        let Some(section) = self.root.get(SECTION) else {
            return Ok(Some(ModelStateMembers::default()));
        };
        let members: ModelStateMembers =
            serde_yaml_ng::from_value(section.clone()).map_err(|cause| {
                PackageCodecError(format!(
                    "{SECTION} {section:?}: expected a model-state member recipe: {cause}"
                ))
            })?;
        members.validate().map_err(PackageCodecError)?;
        Ok(Some(members))
    }
    /// Capture part members without changing unrelated sections.
    /// Example: `package.with_model_states(snapshot.model_states.as_ref())?` during save.
    pub fn with_model_states(
        &self,
        members: Option<&ModelStateMembers>,
    ) -> Result<Self, PackageCodecError> {
        if self.manifest.document_type != DOCUMENT_PART.0 && members.is_some() {
            return Err(PackageCodecError(format!(
                "document type {}, members present {}: expected model-state members only for part documents",
                self.manifest.document_type,
                members.is_some()
            )));
        }
        let Some(members) = members else {
            if self.root.contains_key(SECTION) {
                return Err(PackageCodecError(format!(
                    "{SECTION} content None: expected retained owned part members"
                )));
            }
            return Ok(self.clone());
        };
        members.validate().map_err(PackageCodecError)?;
        let encoded = serde_yaml_ng::to_value(members).map_err(|cause| {
            PackageCodecError(format!(
                "members {members:?}: expected serializable native content: {cause}"
            ))
        })?;
        let mut candidate = self.clone();
        candidate.root.insert(SECTION.into(), encoded);
        Ok(candidate)
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::types::DOCUMENT_DRAWING;
    #[test]
    fn named_members_and_active_selection_round_trip() {
        let package = DocumentPackage::empty(DOCUMENT_PART, "Part");
        let mut members = package.restore_model_states().unwrap().unwrap();
        members.insert(2, "Large").unwrap();
        members.activate(2).unwrap();
        let stored = package.with_model_states(Some(&members)).unwrap();
        assert_eq!(stored.restore_model_states().unwrap(), Some(members));
        assert!(stored.with_model_states(None).is_err());
        super::super::validation::validate_root(&stored.root).unwrap();
    }
    #[test]
    fn malformed_and_wrong_document_members_are_refused() {
        let mut package = DocumentPackage::empty(DOCUMENT_PART, "Part");
        package.root.insert(SECTION.into(), Value::from(3));
        assert!(package.restore_model_states().is_err());
        let drawing = DocumentPackage::empty(DOCUMENT_DRAWING, "Drawing");
        assert!(drawing.restore_model_states().unwrap().is_none());
        assert!(
            drawing
                .with_model_states(Some(&ModelStateMembers::default()))
                .is_err()
        );
    }
}
