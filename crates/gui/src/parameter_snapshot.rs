// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::wire::{ParameterDetail, ToleranceInfo};
use std::collections::BTreeSet;
/// Owned model identity with the canonical parameter presentation contract.
#[derive(Clone, Debug)]
pub struct GuiParameterRow {
    pub parameter: u64,
    pub detail: ParameterDetail,
    pub tolerance_text: String,
    pub has_groups: bool,
}
impl PartialEq for GuiParameterRow {
    fn eq(&self, other: &Self) -> bool {
        // #3614: a stored sign change must request a new frame even when numeric values compare equal.
        self.has_groups == other.has_groups
            && self.tolerance_text == other.tolerance_text
            && self.parameter == other.parameter
            && self.detail == other.detail
            && self.detail.model_value.to_bits() == other.detail.model_value.to_bits()
            && matching_tolerance_bits(
                self.detail.tolerance.as_deref(),
                other.detail.tolerance.as_deref(),
            )
    }
}
fn matching_tolerance_bits(left: Option<&ToleranceInfo>, right: Option<&ToleranceInfo>) -> bool {
    match (left, right) {
        (Some(left), Some(right)) => {
            left.upper.to_bits() == right.upper.to_bits()
                && left.lower.to_bits() == right.lower.to_bits()
        }
        (None, None) => true,
        _ => false,
    }
}
impl GuiParameterRow {
    /// Follow the canonical editable kinds; example: `row.is_editable()` before offering value controls.
    pub(crate) fn is_editable(&self) -> bool {
        matches!(
            self.detail.parameter_info.kind.as_str(),
            "user" | "model" | "table"
        )
    }
}
/// One document's immutable parameter view; edits retain document and parameter identities.
#[derive(Clone, Debug, PartialEq)]
pub struct GuiParameterSnapshot {
    pub document: u64,
    pub rows: Vec<GuiParameterRow>,
    pub linked_tables: Vec<oblikovati_api::wire::DerivedParameterTableInfo>,
    pub link_sources: Vec<GuiParameterLinkSource>,
    pub groups: Vec<GuiParameterGroup>,
}
/// A group retains its immutable API key and stable member identities for native controls.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiParameterGroup {
    pub key: String,
    pub display_name: String,
    pub client_id: String,
    pub members: Vec<u64>,
}
/// One open document that can supply numeric user parameters to a derived table.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiParameterLinkSource {
    pub document: u64,
    pub source_document: String,
    pub display_name: String,
    pub parameters: Vec<GuiParameterLinkParameter>,
}
/// Stable source parameter identity carried through the picker before a name is sent to the API.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct GuiParameterLinkParameter {
    pub parameter: u64,
    pub name: String,
}
impl GuiParameterSnapshot {
    /// Admit display-safe rows before any C widget receives a label or value.
    /// Example: `snapshot.validate()?` before opening the parameter table.
    pub fn validate(&self) -> Result<(), String> {
        if self.document == 0 {
            return Err("GUI parameter document 0: expected an active document identity".into());
        }
        let mut identities = BTreeSet::new();
        for row in &self.rows {
            if !identities.insert(row.parameter) {
                return Err(format!(
                    "GUI parameter ID {} in document {}: expected a unique row identity",
                    row.parameter, self.document
                ));
            }
            validate_parameter_row(row)?;
        }
        validate_linked_tables(&self.linked_tables)
            .and_then(|()| validate_link_sources(&self.link_sources, self.document))
            .and_then(|()| validate_groups(&self.groups, &identities))
    }
}
fn validate_groups(groups: &[GuiParameterGroup], parameters: &BTreeSet<u64>) -> Result<(), String> {
    let mut keys = BTreeSet::new();
    for group in groups {
        if group.key.is_empty() || group.key.contains('\0') || !keys.insert(&group.key) {
            return Err(format!(
                "GUI parameter group {:?}: expected a unique nonempty NUL-free key",
                group.key
            ));
        }
        if group.display_name.is_empty()
            || group.display_name.contains('\0')
            || group.client_id.contains('\0')
        {
            return Err(format!(
                "GUI parameter group {:?}: expected a nonempty NUL-free display name and NUL-free client ID",
                group.key
            ));
        }
        let mut members = BTreeSet::new();
        for member in &group.members {
            if !parameters.contains(member) || !members.insert(member) {
                return Err(format!(
                    "GUI parameter group {:?}, member {}: expected a unique current parameter identity",
                    group.key, member
                ));
            }
        }
    }
    Ok(())
}
fn validate_link_sources(sources: &[GuiParameterLinkSource], target: u64) -> Result<(), String> {
    let mut documents = BTreeSet::new();
    for source in sources {
        if source.document == 0 || source.document == target || !documents.insert(source.document) {
            return Err(format!(
                "GUI link source document {}: expected a unique non-target identity",
                source.document
            ));
        }
        if source.source_document.is_empty()
            || source.source_document.contains('\0')
            || source.display_name.contains('\0')
        {
            return Err(format!(
                "GUI link source document {}: expected nonempty NUL-free names",
                source.document
            ));
        }
        let mut parameters = BTreeSet::new();
        for parameter in &source.parameters {
            if !parameters.insert(parameter.parameter)
                || parameter.name.is_empty()
                || parameter.name.contains('\0')
            {
                return Err(format!(
                    "GUI link source parameter {} in document {}: expected a unique nonempty NUL-free name",
                    parameter.parameter, source.document
                ));
            }
        }
    }
    Ok(())
}
fn validate_parameter_row(row: &GuiParameterRow) -> Result<(), String> {
    let detail = &row.detail;
    for (field, text) in display_fields(row) {
        if let Some(offset) = text.as_bytes().iter().position(|byte| *byte == 0) {
            return Err(format!(
                "GUI parameter {}, field {field}, NUL at byte {offset}: expected display-safe text",
                row.parameter
            ));
        }
    }
    if !detail.model_value.is_finite() {
        return Err(format!(
            "GUI parameter {}, model value {}: expected a finite presentation value",
            row.parameter, detail.model_value
        ));
    }
    validate_parameter_choices(row)
}
fn display_fields(row: &GuiParameterRow) -> [(&'static str, &str); 8] {
    let detail = &row.detail;
    let information = &detail.parameter_info;
    [
        ("name", &information.name),
        ("equation", &information.expression),
        ("value", &information.value),
        ("health", &information.health),
        ("kind", &information.kind),
        ("units", &detail.units),
        ("comment", &detail.comment),
        ("tolerance", &row.tolerance_text),
    ]
}
fn validate_parameter_choices(row: &GuiParameterRow) -> Result<(), String> {
    let Some(choices) = &row.detail.expression_list else {
        return Ok(());
    };
    for text in choices.expressions.as_deref().unwrap_or_default() {
        if text.contains('\0') {
            return Err(format!(
                "GUI parameter {}, choice {text:?}: expected display-safe text",
                row.parameter
            ));
        }
    }
    Ok(())
}
#[cfg(test)]
#[path = "../tests/parameters/snapshot.rs"]
mod tests;

fn validate_linked_tables(
    tables: &[oblikovati_api::wire::DerivedParameterTableInfo],
) -> Result<(), String> {
    let mut identities = BTreeSet::new();
    for table in tables {
        if table.id <= 0 || !identities.insert(table.id) {
            return Err(format!(
                "GUI linked table ID {}: expected a unique positive identity",
                table.id
            ));
        }
        let strings = [&table.source_document, &table.health]
            .into_iter()
            .chain(table.linked.iter().flatten());
        for text in strings {
            if text.contains('\0') {
                return Err(format!(
                    "GUI linked table {}, text {text:?}: expected NUL-free display text",
                    table.id
                ));
            }
        }
    }
    Ok(())
}
