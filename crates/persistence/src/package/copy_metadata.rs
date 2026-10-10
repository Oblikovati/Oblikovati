// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::NewFileMetadata;

impl DocumentPackage {
    /// Apply nonempty copy display-name/subtype overrides without changing the recipe.
    /// Explicit copy destinations take precedence over metadata file/template fields, as in Go.
    /// Example: `let copy = package.with_copy_metadata(&metadata);`.
    pub fn with_copy_metadata(&self, metadata: &NewFileMetadata) -> Self {
        let mut copy = self.clone();
        if !metadata.display_name.is_empty() {
            copy.manifest.display_name = metadata.display_name.clone();
            copy.root
                .insert("displayName".into(), metadata.display_name.clone().into());
        }
        if !metadata.sub_type.is_empty() {
            copy.manifest.sub_type = metadata.sub_type.clone();
            copy.root
                .insert("subType".into(), metadata.sub_type.clone().into());
        }
        copy
    }
}
