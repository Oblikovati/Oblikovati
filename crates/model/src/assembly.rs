// SPDX-License-Identifier: GPL-2.0-only
//! Persisted assembly occurrence definitions without resolved component ownership.
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

/// One Go-compatible occurrence recipe; example: `AssemblyOccurrenceRecipe::default()`.
#[derive(Clone, Debug, Default, PartialEq, Serialize, Deserialize)]
#[serde(default, rename_all = "camelCase", deny_unknown_fields)]
pub struct AssemblyOccurrenceRecipe {
    pub name: String,
    pub component: String,
    pub transform: Vec<f64>,
    pub suppressed: bool,
    pub grounded: bool,
    pub adaptive: bool,
    pub flexible: bool,
    pub child_transforms: BTreeMap<String, Vec<f64>>,
    pub substitute: bool,
    pub bom_structure: String,
    #[serde(rename = "virtual")]
    pub virtual_: bool,
    pub virtual_part_number: String,
    pub virtual_description: String,
    pub virtual_structure: String,
}

impl AssemblyOccurrenceRecipe {
    /// Read the external component reference, if any; example: `occurrence.component_reference()`.
    pub fn component_reference(&self) -> Option<&str> {
        (!self.virtual_ && !self.component.is_empty()).then_some(self.component.as_str())
    }
    /// Validate saved matrices before resolving the component; example: `occurrence.validate()?`.
    pub fn validate(&self) -> Result<(), String> {
        validate_cells(&self.name, "transform", &self.transform, true)?;
        for (child, cells) in &self.child_transforms {
            validate_cells(&self.name, &format!("child {child:?}"), cells, false)?;
        }
        Ok(())
    }
    /// Resolve an omitted transform to identity; example: `let matrix = occurrence.placement()?`.
    pub fn placement(&self) -> Result<oblikovati_api::types::Matrix, String> {
        self.validate()?;
        let cells = if self.transform.is_empty() {
            [
                1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1., 0., 0., 0., 0., 1.,
            ]
        } else {
            self.transform
                .as_slice()
                .try_into()
                .expect("validated matrix length")
        };
        Ok(oblikovati_api::types::Matrix { cells })
    }
}

/// Return each external component once, in placement order; example: `component_references(&occurrences)`.
pub fn component_references(records: &[AssemblyOccurrenceRecipe]) -> Vec<&str> {
    let mut seen = std::collections::BTreeSet::new();
    records
        .iter()
        .filter_map(AssemblyOccurrenceRecipe::component_reference)
        .filter(|name| seen.insert(*name))
        .collect()
}
fn validate_cells(
    name: &str,
    field: &str,
    cells: &[f64],
    allow_identity: bool,
) -> Result<(), String> {
    if (cells.len() == 16 || allow_identity && cells.is_empty())
        && cells.iter().all(|v| v.is_finite())
    {
        return Ok(());
    }
    Err(format!(
        "occurrence {name:?} {field} {cells:?}: expected {}finite row-major cells",
        if allow_identity { "zero or 16 " } else { "16 " }
    ))
}
