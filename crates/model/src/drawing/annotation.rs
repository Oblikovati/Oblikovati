// SPDX-License-Identifier: GPL-2.0-only
//! Authored drawing annotations; their evaluated content comes from the referenced model.
use super::{DrawingError, validation::require_label};
use serde::{Deserialize, Serialize};

/// A supported annotation definition in the Go drawing-recipe shape.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(tag = "kind", deny_unknown_fields)]
pub enum DrawingAnnotationDefinition {
    /// A parts list whose rows must be resolved from the referenced model's BOM.
    #[serde(rename = "partsList")]
    PartsList {
        name: String,
        #[serde(rename = "xmm", default)]
        x_mm: f64,
        #[serde(rename = "ymm", default)]
        y_mm: f64,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        characteristic: Option<String>,
        #[serde(
            rename = "materialRemoval",
            default,
            skip_serializing_if = "Option::is_none"
        )]
        material_removal: Option<String>,
    },
}
impl DrawingAnnotationDefinition {
    /// Define a parts list without inventing BOM rows; example: `DrawingAnnotationDefinition::parts_list("Parts", [40., 200.])?`.
    pub fn parts_list(name: &str, position: [f64; 2]) -> Result<Self, DrawingError> {
        let value = Self::PartsList {
            name: name.into(),
            x_mm: position[0],
            y_mm: position[1],
            characteristic: None,
            material_removal: None,
        };
        value.validate()?;
        Ok(value)
    }
    /// Read the annotation name; example: `definition.name()` returns `"Parts"`.
    pub fn name(&self) -> &str {
        match self {
            Self::PartsList { name, .. } => name,
        }
    }
    pub(super) fn validate(&self) -> Result<(), DrawingError> {
        match self {
            Self::PartsList {
                name,
                x_mm,
                y_mm,
                characteristic,
                material_removal,
            } => {
                require_label("annotation name", name)?;
                if characteristic
                    .as_deref()
                    .is_some_and(|value| value != "enum(?)")
                    || material_removal
                        .as_deref()
                        .is_some_and(|value| value != "any")
                {
                    return Err(DrawingError(format!(
                        "parts list {name:?} characteristic {characteristic:?}, material removal {material_removal:?}: expected the Go default values"
                    )));
                }
                if !x_mm.is_finite() || !y_mm.is_finite() {
                    return Err(DrawingError(format!(
                        "parts list {name:?} position [{x_mm}, {y_mm}]: expected finite sheet coordinates"
                    )));
                }
            }
        }
        Ok(())
    }
}
