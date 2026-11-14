// SPDX-License-Identifier: GPL-2.0-only
use super::records::null_record;
use oblikovati_model::assets::{AssetAssignments, AssignmentReference};
use serde::{Deserialize, Serialize};
use serde_yaml_ng::{Mapping, Value};
use std::collections::BTreeSet;
#[derive(Default, Serialize, Deserialize)]
#[serde(default, rename_all = "camelCase")]
pub(super) struct AssignmentRecipe {
    #[serde(deserialize_with = "null_record")]
    pub part_material: String,
    #[serde(deserialize_with = "null_record")]
    pub part_appearance: String,
    #[serde(deserialize_with = "null_record")]
    pub body_material: Mapping,
    #[serde(deserialize_with = "null_record")]
    pub body_appearance: Mapping,
    #[serde(deserialize_with = "null_record")]
    pub face_appearance: Mapping,
}
#[derive(Serialize, Deserialize)]
#[serde(deny_unknown_fields, rename_all = "camelCase")]
struct OccurrenceReference {
    reference_key: String,
    occurrence_path: Vec<String>,
}
impl AssignmentRecipe {
    pub fn restore(&self, schema: u32) -> Result<AssetAssignments, String> {
        let mut assignments = AssetAssignments::default();
        assignments.set_part_material(&self.part_material)?;
        assignments.set_part_appearance(&self.part_appearance)?;
        restore_overrides(
            &self.body_material,
            schema,
            &mut assignments,
            AssetAssignments::set_body_material,
        )?;
        restore_overrides(
            &self.body_appearance,
            schema,
            &mut assignments,
            AssetAssignments::set_body_appearance,
        )?;
        restore_overrides(
            &self.face_appearance,
            schema,
            &mut assignments,
            AssetAssignments::set_face_appearance,
        )?;
        Ok(assignments)
    }
    pub fn capture(assignments: &AssetAssignments) -> Result<Self, String> {
        Ok(Self {
            part_material: assignments.part_material_id().unwrap_or_default().into(),
            part_appearance: assignments.part_appearance_id().unwrap_or_default().into(),
            body_material: capture_overrides(assignments.body_materials())?,
            body_appearance: capture_overrides(assignments.body_appearances())?,
            face_appearance: capture_overrides(assignments.face_appearances())?,
        })
    }
    pub fn has_occurrences(&self) -> bool {
        [
            &self.body_material,
            &self.body_appearance,
            &self.face_appearance,
        ]
        .iter()
        .any(|entries| entries.keys().any(Value::is_mapping))
    }
}
fn restore_overrides(
    entries: &Mapping,
    schema: u32,
    assignments: &mut AssetAssignments,
    apply: fn(&mut AssetAssignments, &AssignmentReference, &str) -> Result<(), String>,
) -> Result<(), String> {
    let mut seen = BTreeSet::new();
    for (key, value) in entries {
        let target = restore_reference(key, schema)?;
        if !seen.insert(target.clone()) {
            return Err(format!(
                "assignment target {target:?}: expected a unique reference in its assignment family"
            ));
        }
        let id = value
            .as_str()
            .ok_or_else(|| format!("assignment asset {value:?}: expected an asset ID string"))?;
        apply(assignments, &target, id)?;
    }
    Ok(())
}
fn restore_reference(key: &Value, schema: u32) -> Result<AssignmentReference, String> {
    if let Some(key) = key.as_str() {
        return AssignmentReference::new(key, vec![]);
    }
    if schema < 3 {
        return Err(format!(
            "assignment reference {key:?}, schema {schema}: expected schema 3 for an occurrence reference"
        ));
    }
    let record: OccurrenceReference = serde_yaml_ng::from_value(key.clone()).map_err(|cause| {
        format!("assignment reference {key:?}: expected referenceKey and occurrencePath: {cause}")
    })?;
    AssignmentReference::new(&record.reference_key, record.occurrence_path)
}
fn capture_overrides<'a>(
    entries: impl Iterator<Item = (&'a AssignmentReference, &'a str)>,
) -> Result<Mapping, String> {
    let mut encoded = Mapping::new();
    for (target, id) in entries {
        encoded.insert(capture_reference(target)?, Value::String(id.into()));
    }
    Ok(encoded)
}
fn capture_reference(target: &AssignmentReference) -> Result<Value, String> {
    if target.occurrence_path().is_empty() {
        return Ok(Value::String(target.reference_key().into()));
    }
    serde_yaml_ng::to_value(OccurrenceReference {
        reference_key: target.reference_key().into(),
        occurrence_path: target.occurrence_path().to_vec(),
    })
    .map_err(|cause| cause.to_string())
}
