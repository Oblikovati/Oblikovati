// SPDX-License-Identifier: GPL-2.0-only

//! Ordered add-in metadata anchored by persistent reference identity.
mod edit;
mod query;
mod target;
use oblikovati_api::types::VariantValue;
use std::{error::Error, fmt};
pub use target::AttributeTarget;

/// An owned attribute with a canonical API value.
#[derive(Clone, Debug, PartialEq)]
pub struct ObjectAttribute {
    pub name: String,
    pub value: VariantValue,
}

/// A namespace of attributes, retaining creation order.
#[derive(Clone, Debug, PartialEq)]
pub struct ObjectAttributeSet {
    pub name: String,
    attributes: Vec<ObjectAttribute>,
}
impl ObjectAttributeSet {
    /// Inspect attributes without granting mutation; example: `set.attributes()`.
    pub fn attributes(&self) -> &[ObjectAttribute] {
        &self.attributes
    }
}

/// All attribute sets for one persistent target, with no live entity pointer.
#[derive(Clone, Debug, PartialEq)]
pub struct AttributeAnchor {
    pub target: AttributeTarget,
    sets: Vec<ObjectAttributeSet>,
}
impl AttributeAnchor {
    /// Inspect namespaces in creation order; example: `anchor.sets()`.
    pub fn sets(&self) -> &[ObjectAttributeSet] {
        &self.sets
    }
}

/// Document-owned metadata; geometry resolution belongs to the host.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct DocumentAttributes {
    anchors: Vec<AttributeAnchor>,
}

/// One matching attribute and its owning target and namespace.
#[derive(Clone, Copy, Debug)]
pub struct AttributeMatch<'collection> {
    pub target: &'collection AttributeTarget,
    pub set: &'collection ObjectAttributeSet,
    pub attribute: &'collection ObjectAttribute,
}

/// A rejected metadata edit; the original collection is unchanged.
#[derive(Clone, Debug, Eq, PartialEq)]
pub struct AttributeError(String);
impl fmt::Display for AttributeError {
    fn fmt(&self, formatter: &mut fmt::Formatter<'_>) -> fmt::Result {
        self.0.fmt(formatter)
    }
}
impl Error for AttributeError {}
