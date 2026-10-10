// SPDX-License-Identifier: GPL-2.0-only

//! Ordered document metadata used by drawings, bills of materials, and file properties.
use oblikovati_api::{
    types::{Variant, VariantValue},
    wire::PropertyInfo,
};
mod comparison;
mod exposure;
mod restore;
use std::{error::Error, fmt};

pub const SUMMARY_INFORMATION: &str = "Summary Information";
pub const DOCUMENT_SUMMARY: &str = "Document Summary Information";
pub const DESIGN_TRACKING: &str = "Design Tracking Properties";
pub const USER_DEFINED: &str = "User Defined Properties";

/// A named typed value; edits go through its document-owned collection.
#[derive(Clone, Debug, PartialEq)]
pub struct DocumentProperty {
    pub name: String,
    pub value: VariantValue,
    pub from_parameter: String,
}

/// Properties retain insertion order within a set.
#[derive(Clone, Debug, PartialEq)]
pub struct DocumentPropertySet {
    pub name: String,
    properties: Vec<DocumentProperty>,
}

impl DocumentPropertySet {
    /// Inspect properties in insertion order; for example `set.properties()`.
    pub fn properties(&self) -> &[DocumentProperty] {
        &self.properties
    }
}

/// The four standard sets precede custom sets in creation order.
#[derive(Clone, Debug, PartialEq)]
pub struct DocumentProperties {
    sets: Vec<DocumentPropertySet>,
}

impl Default for DocumentProperties {
    fn default() -> Self {
        let sets = [
            SUMMARY_INFORMATION,
            DOCUMENT_SUMMARY,
            DESIGN_TRACKING,
            USER_DEFINED,
        ]
        .into_iter()
        .map(|name| DocumentPropertySet {
            name: name.into(),
            properties: Vec::new(),
        })
        .collect();
        Self { sets }
    }
}

/// A refused property edit with the offending property or value named.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct PropertyError(pub(crate) String);
impl fmt::Display for PropertyError {
    fn fmt(&self, output: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(output)
    }
}
impl Error for PropertyError {}

impl DocumentProperties {
    /// Inspect all sets in creation order; for example `properties.sets()`.
    pub fn sets(&self) -> &[DocumentPropertySet] {
        &self.sets
    }

    /// Create a named empty set, retaining the position of any existing set.
    /// Example: `properties.ensure_set("Manufacturing")?;` before defining its properties.
    pub fn ensure_set(&mut self, name: &str) -> Result<bool, PropertyError> {
        if name.is_empty() {
            return Err(PropertyError(format!(
                "property set {name:?}: expected a nonempty name"
            )));
        }
        if self.set(name).is_some() {
            return Ok(false);
        }
        self.set_index_or_insert(name);
        Ok(true)
    }

    /// Find a set without creating it; for example `properties.set(USER_DEFINED)`.
    pub fn set(&self, name: &str) -> Option<&DocumentPropertySet> {
        self.sets.iter().find(|set| set.name == name)
    }

    /// Read one property; for example `properties.get(SUMMARY_INFORMATION, "Title")`.
    pub fn get(&self, set: &str, name: &str) -> Option<&DocumentProperty> {
        self.set(set)?
            .properties
            .iter()
            .find(|property| property.name == name)
    }

    /// Insert or replace a typed property while retaining its position and provenance.
    /// Example: `properties.put(USER_DEFINED, "Approved", VariantValue::Boolean(true))?;`.
    pub fn put(&mut self, set: &str, name: &str, value: VariantValue) -> Result<(), PropertyError> {
        validate_property(set, name, &value)?;
        let value = crate::metadata_value::normalize(value);
        if let Some(property) = self.find_mut(set, name) {
            property.value = value;
            return Ok(());
        }
        let index = self.set_index_or_insert(set);
        self.sets[index].properties.push(DocumentProperty {
            name: name.into(),
            value,
            from_parameter: String::new(),
        });
        Ok(())
    }

    /// Remove a property without removing its set; for example `properties.remove(USER_DEFINED, "Cost")`.
    pub fn remove(&mut self, set: &str, name: &str) -> bool {
        let Some(set) = self.sets.iter_mut().find(|candidate| candidate.name == set) else {
            return false;
        };
        let Some(index) = set
            .properties
            .iter()
            .position(|property| property.name == name)
        else {
            return false;
        };
        set.properties.remove(index);
        true
    }

    /// Project a property through the canonical API DTO; for example `properties.info(USER_DEFINED, "Cost")`.
    pub fn info(&self, set: &str, name: &str) -> Option<PropertyInfo> {
        let property = self.get(set, name)?;
        Some(PropertyInfo {
            set: set.into(),
            name: property.name.clone(),
            value: Variant {
                value: property.value.clone(),
                unit: String::new(),
            },
            from_parameter: property.from_parameter.clone(),
        })
    }

    fn find_mut(&mut self, set: &str, name: &str) -> Option<&mut DocumentProperty> {
        self.sets
            .iter_mut()
            .find(|candidate| candidate.name == set)?
            .properties
            .iter_mut()
            .find(|property| property.name == name)
    }

    fn set_index_or_insert(&mut self, name: &str) -> usize {
        if let Some(index) = self.sets.iter().position(|set| set.name == name) {
            return index;
        }
        self.sets.push(DocumentPropertySet {
            name: name.into(),
            properties: Vec::new(),
        });
        self.sets.len() - 1
    }
}

fn validate_property(set: &str, name: &str, value: &VariantValue) -> Result<(), PropertyError> {
    if set.is_empty() || name.is_empty() {
        return Err(PropertyError(format!(
            "property {set:?}/{name:?}: expected nonempty set and property names"
        )));
    }
    crate::metadata_value::validate(value).map_err(|expected| {
        PropertyError(format!(
            "property {set:?}/{name:?} value {value:?}: expected {expected}"
        ))
    })
}
