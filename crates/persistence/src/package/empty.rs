// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::DocumentType;

impl DocumentPackage {
    /// Build a schema-2 envelope for a new recipe without persisted identity.
    /// Example: `DocumentPackage::empty(DOCUMENT_PART, "Bracket")` before host registration.
    pub fn empty(kind: DocumentType, name: &str) -> Self {
        let manifest = PackageManifest {
            schema_version: 2,
            document_type: kind.0,
            sub_type: String::new(),
            display_name: name.into(),
        };
        let root = Mapping::from_iter([
            ("schemaVersion".into(), 2u32.into()),
            ("documentType".into(), kind.0.into()),
            ("displayName".into(), name.into()),
        ]);
        Self {
            manifest,
            root,
            streams: BTreeMap::new(),
            resources: BTreeMap::new(),
        }
    }
}

impl DocumentPackage {
    /// Mark a new package as a registered flavor of its document type.
    /// Example: `DocumentPackage::empty(DOCUMENT_PART, "Bracket").with_sub_type(id)`.
    pub fn with_sub_type(mut self, sub_type: &str) -> Self {
        self.manifest.sub_type = sub_type.into();
        self.root.insert("subType".into(), sub_type.into());
        self
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::types::*;

    #[test]
    fn a_new_package_flavor_survives_encoding() {
        let package = DocumentPackage::empty(DOCUMENT_PART, "Bracket")
            .with_sub_type(SUB_TYPE_SHEET_METAL_PART.0.as_ref());
        let decoded = DocumentPackage::decode(&package.encode().unwrap()).unwrap();
        assert_eq!(decoded.manifest().sub_type, SUB_TYPE_SHEET_METAL_PART.0);
    }

    #[test]
    fn empty_envelopes_round_trip_titles_kinds_and_have_no_persisted_identity() {
        for kind in [
            DOCUMENT_PART,
            DOCUMENT_ASSEMBLY,
            DOCUMENT_DRAWING,
            DOCUMENT_PRESENTATION,
        ] {
            let package = DocumentPackage::empty(kind, "Bracket: 日本語\n{not YAML}");
            assert_eq!(
                DocumentPackage::decode(&package.encode().unwrap()).unwrap(),
                package
            );
            assert!(package.identity_record().unwrap().is_none());
            assert!(package.streams().is_empty());
            let restored = package.restore_document_parameters().unwrap();
            assert!(
                package
                    .restore_parameter_properties(&restored.parameters)
                    .is_ok()
            );
            assert_eq!(package.manifest().document_type, kind.0);
        }
    }
}
