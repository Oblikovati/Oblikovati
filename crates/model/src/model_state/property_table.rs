// SPDX-License-Identifier: GPL-2.0-only
//! Ordered visible property columns and retained per-member values (#4119).
use oblikovati_api::types::{ModelStatePropertyKey, Variant};
use serde::{Deserialize, Serialize};

/// One property override belonging to a model-state member.
/// Example: `ModelStatePropertyValue { member: 1, property: key, value }`.
#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ModelStatePropertyValue {
    pub member: u64,
    pub property: ModelStatePropertyKey,
    pub value: Variant,
}

impl PartialEq for ModelStatePropertyValue {
    fn eq(&self, other: &Self) -> bool {
        self.member == other.member
            && self.property == other.property
            && self.value.unit == other.value.unit
            && crate::metadata_value::same(&self.value.value, &other.value.value)
    }
}
impl Eq for ModelStatePropertyValue {}

/// The inherited document value of a property when a member has no override.
/// Example: `ModelStatePropertyBase { property: key, value: None }`.
#[derive(Clone, Debug, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ModelStatePropertyBase {
    pub property: ModelStatePropertyKey,
    pub value: Option<Variant>,
}
impl PartialEq for ModelStatePropertyBase {
    fn eq(&self, other: &Self) -> bool {
        self.property == other.property && same_optional(&self.value, &other.value)
    }
}
impl Eq for ModelStatePropertyBase {}

fn same_optional(first: &Option<Variant>, second: &Option<Variant>) -> bool {
    match (first, second) {
        (None, None) => true,
        (Some(first), Some(second)) => {
            first.unit == second.unit && crate::metadata_value::same(&first.value, &second.value)
        }
        _ => false,
    }
}

/// Authored property table data. Hidden values survive removal of a column.
/// Example: `ModelStatePropertyTable::default()` creates an empty table.
#[derive(Clone, Debug, Default, PartialEq, Serialize, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct ModelStatePropertyTable {
    values: Vec<ModelStatePropertyValue>,
    #[serde(default)]
    bases: Vec<ModelStatePropertyBase>,
}
impl Eq for ModelStatePropertyTable {}
impl ModelStatePropertyTable {
    /// Inspect all retained overrides; example: `table.values()`.
    pub fn values(&self) -> &[ModelStatePropertyValue] {
        &self.values
    }
    /// Inspect inherited property values; example: `table.bases()`.
    pub fn bases(&self) -> &[ModelStatePropertyBase] {
        &self.bases
    }
    /// Find an inherited property; example: `table.base(&key)`.
    pub fn base(&self, property: &ModelStatePropertyKey) -> Option<&ModelStatePropertyBase> {
        self.bases.iter().find(|base| base.property == *property)
    }
    /// Find an authored value; example: `table.value(1, &key)`.
    pub fn value(&self, member: u64, property: &ModelStatePropertyKey) -> Option<&Variant> {
        self.values
            .iter()
            .find(|value| value.member == member && value.property == *property)
            .map(|value| &value.value)
    }
    /// Resolve an inherited value; example: `table.effective_value(1, &key)`.
    pub fn effective_value(
        &self,
        member: u64,
        property: &ModelStatePropertyKey,
    ) -> Option<&Variant> {
        self.value(member, property).or_else(|| {
            self.bases
                .iter()
                .find(|base| base.property == *property)?
                .value
                .as_ref()
        })
    }
    pub(super) fn validate(&self, members: &[super::ModelStateMember]) -> Result<(), String> {
        self.validate_bases()?;
        self.validate_values(members)?;
        self.validate_property_bases()
    }
    fn validate_property_bases(&self) -> Result<(), String> {
        for key in self.values.iter().map(|value| &value.property) {
            if self.base(key).is_none() {
                return Err(format!(
                    "property {key:?}: expected an inherited base entry"
                ));
            }
        }
        Ok(())
    }
    fn validate_bases(&self) -> Result<(), String> {
        for (index, base) in self.bases.iter().enumerate() {
            validate_key(&base.property)?;
            if self.bases[..index]
                .iter()
                .any(|item| item.property == base.property)
            {
                return Err(format!(
                    "property base {base:?}: expected a unique property"
                ));
            }
            if let Some(value) = &base.value {
                validate_variant(value)
                    .map_err(|expected| format!("property base {base:?}: expected {expected}"))?;
            }
        }
        Ok(())
    }
    fn validate_values(&self, members: &[super::ModelStateMember]) -> Result<(), String> {
        for (index, value) in self.values.iter().enumerate() {
            validate_value(value, members)?;
            if self.values[..index].iter().any(|previous| {
                previous.member == value.member && previous.property == value.property
            }) {
                return Err(format!(
                    "property value {value:?}: expected a unique member and property"
                ));
            }
        }
        Ok(())
    }
    pub(super) fn clone_member(&mut self, source: u64, target: u64) {
        self.forget_member(target);
        let copied: Vec<_> = self
            .values
            .iter()
            .filter(|value| value.member == source)
            .cloned()
            .map(|mut value| {
                value.member = target;
                value
            })
            .collect();
        self.values.extend(copied);
    }
    pub(super) fn forget_member(&mut self, member: u64) {
        self.values.retain(|value| value.member != member);
    }
    /// Construct a candidate for validation against its members.
    /// Example: `ModelStatePropertyTable::new(properties, values)` creates inherited bases.
    pub fn new(
        properties: Vec<ModelStatePropertyKey>,
        values: Vec<ModelStatePropertyValue>,
    ) -> Self {
        let mut bases = Vec::<ModelStatePropertyBase>::new();
        for key in properties
            .iter()
            .chain(values.iter().map(|value| &value.property))
        {
            if !bases.iter().any(|base| base.property == *key) {
                bases.push(ModelStatePropertyBase {
                    property: key.clone(),
                    value: None,
                });
            }
        }
        Self { values, bases }
    }
    /// Add inherited values to a table candidate; example: `table.with_bases(bases)`.
    pub fn with_bases(mut self, bases: Vec<ModelStatePropertyBase>) -> Self {
        self.bases = bases;
        self
    }
}
fn validate_key(key: &ModelStatePropertyKey) -> Result<(), String> {
    if key.set.trim().is_empty()
        || key.name.trim().is_empty()
        || key.set.contains('\0')
        || key.name.contains('\0')
    {
        return Err(format!(
            "property key {key:?}: expected nonempty names without NUL"
        ));
    }
    Ok(())
}
fn validate_value(
    value: &ModelStatePropertyValue,
    members: &[super::ModelStateMember],
) -> Result<(), String> {
    validate_key(&value.property)?;
    if !members.iter().any(|member| member.id == value.member) {
        return Err(format!(
            "property value {value:?}: expected an existing member"
        ));
    }
    validate_variant(&value.value)
        .map_err(|expected| format!("property value {value:?}: expected {expected}"))
}

fn validate_variant(value: &Variant) -> Result<(), &'static str> {
    if !value.unit.is_empty() {
        return Err("a document property value without a unit string");
    }
    crate::metadata_value::validate(&value.value)
}
