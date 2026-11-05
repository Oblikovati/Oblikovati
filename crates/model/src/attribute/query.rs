// SPDX-License-Identifier: GPL-2.0-only

use super::*;

impl DocumentAttributes {
    /// Inspect targets in creation order; example: `attributes.anchors()`.
    pub fn anchors(&self) -> &[AttributeAnchor] {
        &self.anchors
    }

    /// Look up a namespace without creating it; example: `attributes.set(&target, "AddIn")`.
    pub fn set(&self, target: &AttributeTarget, set: &str) -> Option<&ObjectAttributeSet> {
        self.anchors
            .iter()
            .find(|anchor| &anchor.target == target)?
            .sets
            .iter()
            .find(|candidate| candidate.name == set)
    }

    /// Read a typed attribute; example: `attributes.get(&target, "AddIn", "tag")`.
    pub fn get(&self, target: &AttributeTarget, set: &str, name: &str) -> Option<&ObjectAttribute> {
        self.set(target, set)?
            .attributes
            .iter()
            .find(|attribute| attribute.name == name)
    }

    /// Find matching attributes in target and attribute insertion order.
    /// Example: `attributes.find("AddIn", "")` selects all attributes in that namespace.
    pub fn find(&self, set: &str, name: &str) -> Vec<AttributeMatch<'_>> {
        let owners = self.anchors.iter().filter_map(|anchor| {
            anchor
                .sets
                .iter()
                .find(|candidate| candidate.name == set)
                .map(|set| (&anchor.target, set))
        });
        owners
            .flat_map(|(target, set)| {
                set.attributes
                    .iter()
                    .filter(move |attribute| name.is_empty() || attribute.name == name)
                    .map(move |attribute| AttributeMatch {
                        target,
                        set,
                        attribute,
                    })
            })
            .collect()
    }

    /// Compare persisted state, retaining the distinction between signed zeros.
    /// Example: `before.same_recipe(&after)` before recording history.
    pub fn same_recipe(&self, other: &Self) -> bool {
        self == other
            && self
                .values()
                .zip(other.values())
                .all(|(left, right)| crate::metadata_value::same(left, right))
    }

    fn values(&self) -> impl Iterator<Item = &VariantValue> {
        self.anchors
            .iter()
            .flat_map(|anchor| &anchor.sets)
            .flat_map(|set| &set.attributes)
            .map(|attribute| &attribute.value)
    }
}
