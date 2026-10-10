// SPDX-License-Identifier: GPL-2.0-only

use super::{DocumentPackage, PackageCodecError, Value, codec_error};
use crate::metadata::{decode_attributes, encode_attributes};
use oblikovati_model::attribute::DocumentAttributes;

impl DocumentPackage {
    /// Reconstruct add-in attributes without binding their reference keys to geometry.
    /// Example: `let attributes = package.restore_attributes()?;`.
    pub fn restore_attributes(&self) -> Result<DocumentAttributes, PackageCodecError> {
        let bytes: Vec<u8> = match self.root.get("attributes") {
            None | Some(Value::Null) => Vec::new(),
            Some(section) => serde_yaml_ng::from_value(section.clone()).map_err(codec_error)?,
        };
        if bytes.is_empty() {
            return Ok(DocumentAttributes::default());
        }
        decode_attributes(&bytes).map_err(attribute_error)
    }

    /// Replace only the attribute section after validating retained and replacement state.
    /// Example: `let saved = package.with_attributes(&attributes)?;`.
    pub fn with_attributes(
        &self,
        attributes: &DocumentAttributes,
    ) -> Result<Self, PackageCodecError> {
        self.restore_attributes()?;
        let mut candidate = self.clone();
        candidate.root.remove("attributes");
        if !attributes.anchors().is_empty() {
            let bytes = encode_attributes(attributes).map_err(attribute_error)?;
            candidate.root.insert(
                "attributes".into(),
                serde_yaml_ng::to_value(bytes).map_err(codec_error)?,
            );
        }
        if !candidate.restore_attributes()?.same_recipe(attributes) {
            return Err(PackageCodecError(
                "attribute export: expected an exact collection round trip".into(),
            ));
        }
        Ok(candidate)
    }
}

fn attribute_error(error: impl std::fmt::Display) -> PackageCodecError {
    PackageCodecError(format!("document attributes: {error}"))
}
