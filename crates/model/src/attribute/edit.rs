// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use crate::metadata_value;

impl DocumentAttributes {
    /// Create a namespace without attributes; example: `attributes.ensure_set(&target, "AddIn")?;`.
    pub fn ensure_set(
        &mut self,
        target: &AttributeTarget,
        set: &str,
    ) -> Result<bool, AttributeError> {
        require_name(set)?;
        if self.set(target, set).is_some() {
            return Ok(false);
        }
        self.anchor_mut(target).sets.push(ObjectAttributeSet {
            name: set.into(),
            attributes: Vec::new(),
        });
        Ok(true)
    }

    /// Insert or replace a value without changing its order; example: `attributes.put(&target, "AddIn", "tag", value)?;`.
    pub fn put(
        &mut self,
        target: &AttributeTarget,
        set: &str,
        name: &str,
        value: VariantValue,
    ) -> Result<(), AttributeError> {
        require_name(set)?;
        require_name(name)?;
        metadata_value::validate(&value).map_err(|expected| {
            AttributeError(format!(
                "attribute {set:?}/{name:?} value {value:?}: expected {expected}"
            ))
        })?;
        self.ensure_set(target, set)?;
        let attributes = &mut self.set_mut(target, set).expect("created set").attributes;
        let value = metadata_value::normalize(value);
        if let Some(attribute) = attributes
            .iter_mut()
            .find(|attribute| attribute.name == name)
        {
            attribute.value = value;
            return Ok(());
        }
        attributes.push(ObjectAttribute {
            name: name.into(),
            value,
        });
        Ok(())
    }

    /// Remove one attribute, retaining its namespace; example: `attributes.remove(&target, "AddIn", "tag")`.
    pub fn remove(&mut self, target: &AttributeTarget, set: &str, name: &str) -> bool {
        let Some(set) = self.set_mut(target, set) else {
            return false;
        };
        let previous = set.attributes.len();
        set.attributes.retain(|attribute| attribute.name != name);
        previous != set.attributes.len()
    }

    /// Remove one namespace, retaining its anchor; example: `attributes.remove_set(&target, "AddIn")`.
    pub fn remove_set(&mut self, target: &AttributeTarget, set: &str) -> bool {
        let Some(anchor) = self
            .anchors
            .iter_mut()
            .find(|anchor| &anchor.target == target)
        else {
            return false;
        };
        let previous = anchor.sets.len();
        anchor.sets.retain(|candidate| candidate.name != set);
        previous != anchor.sets.len()
    }

    /// Remove all metadata at one target; example: `attributes.remove_target(&target)`.
    pub fn remove_target(&mut self, target: &AttributeTarget) -> bool {
        let previous = self.anchors.len();
        self.anchors.retain(|anchor| &anchor.target != target);
        previous != self.anchors.len()
    }

    fn anchor_mut(&mut self, target: &AttributeTarget) -> &mut AttributeAnchor {
        let index = self
            .anchors
            .iter()
            .position(|anchor| &anchor.target == target)
            .unwrap_or_else(|| {
                self.anchors.push(AttributeAnchor {
                    target: target.clone(),
                    sets: Vec::new(),
                });
                self.anchors.len() - 1
            });
        &mut self.anchors[index]
    }

    fn set_mut(&mut self, target: &AttributeTarget, set: &str) -> Option<&mut ObjectAttributeSet> {
        self.anchors
            .iter_mut()
            .find(|anchor| &anchor.target == target)?
            .sets
            .iter_mut()
            .find(|candidate| candidate.name == set)
    }
}

fn require_name(name: &str) -> Result<(), AttributeError> {
    if name.is_empty() {
        return Err(AttributeError(format!(
            "attribute name {name:?}: expected a nonempty name"
        )));
    }
    Ok(())
}
