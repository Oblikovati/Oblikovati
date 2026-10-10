// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::NewFileMetadata;

impl DocumentPackage {
    /// Apply nonempty copy display-name/subtype overrides without changing the recipe.
    /// Explicit copy destinations take precedence over metadata file/template fields, as in Go.
    /// Example: `let copy = package.with_copy_metadata(&metadata);`.
    pub fn with_copy_metadata(&self, metadata: &NewFileMetadata) -> Self {
        let mut copy = if metadata.display_name.is_empty() {
            self.clone()
        } else {
            self.with_display_name(&metadata.display_name)
        };
        if !metadata.sub_type.is_empty() {
            copy.manifest.sub_type = metadata.sub_type.clone();
            copy.root
                .insert("subType".into(), metadata.sub_type.clone().into());
        }
        copy
    }
}

impl DocumentPackage {
    /// Replace the title override; an empty name permits a file-derived display name.
    /// Example: `let derived = package.with_display_name("");`.
    pub fn with_display_name(&self, name: &str) -> Self {
        let mut copy = self.clone();
        copy.manifest.display_name = name.into();
        copy.root.insert("displayName".into(), name.into());
        copy
    }
}

impl PackageManifest {
    /// Read the canonical document kind; example: `manifest.type_name() == "part"`.
    pub fn type_name(&self) -> &'static str {
        use oblikovati_api::types::*;
        match DocumentType(self.document_type) {
            DOCUMENT_PART => "part",
            DOCUMENT_ASSEMBLY => "assembly",
            DOCUMENT_DRAWING => "drawing",
            DOCUMENT_PRESENTATION => "presentation",
            _ => "unknown",
        }
    }
}
