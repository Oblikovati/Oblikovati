// SPDX-License-Identifier: GPL-2.0-only
//! Authored sheet annotations and the placement of their labels.
use super::{DrawingError, DrawingRevisionRow, validation::require_label};
use serde::{Deserialize, Serialize};

/// A supported annotation definition in the Go drawing-recipe shape.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(tag = "kind", deny_unknown_fields)]
pub enum DrawingAnnotationDefinition {
    #[serde(rename = "datumFeature")]
    DatumFeature {
        name: String,
        #[serde(rename = "xmm", default)]
        x_mm: f64,
        #[serde(rename = "ymm", default)]
        y_mm: f64,
        #[serde(rename = "tag")]
        letter: String,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        characteristic: Option<String>,
        #[serde(
            rename = "materialRemoval",
            default,
            skip_serializing_if = "Option::is_none"
        )]
        material_removal: Option<String>,
    },
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
    #[serde(rename = "drawingNote")]
    Note {
        name: String,
        #[serde(rename = "xmm", default)]
        x_mm: f64,
        #[serde(rename = "ymm", default)]
        y_mm: f64,
        #[serde(rename = "widthMm", default)]
        leader_x_mm: f64,
        #[serde(rename = "heightMm", default)]
        leader_y_mm: f64,
        #[serde(rename = "tag")]
        text: String,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        characteristic: Option<String>,
        #[serde(
            rename = "materialRemoval",
            default,
            skip_serializing_if = "Option::is_none"
        )]
        material_removal: Option<String>,
    },
    #[serde(rename = "customTable")]
    CustomTable {
        name: String,
        #[serde(rename = "xmm", default)]
        x_mm: f64,
        #[serde(rename = "ymm", default)]
        y_mm: f64,
        headers: Vec<String>,
        #[serde(default, skip_serializing_if = "Vec::is_empty")]
        rows: Vec<Option<Vec<String>>>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        characteristic: Option<String>,
        #[serde(
            rename = "materialRemoval",
            default,
            skip_serializing_if = "Option::is_none"
        )]
        material_removal: Option<String>,
    },
    #[serde(rename = "revisionTable")]
    RevisionTable {
        name: String,
        #[serde(rename = "xmm", default)]
        x_mm: f64,
        #[serde(rename = "ymm", default)]
        y_mm: f64,
        revisions: Vec<DrawingRevisionRow>,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        characteristic: Option<String>,
        #[serde(
            rename = "materialRemoval",
            default,
            skip_serializing_if = "Option::is_none"
        )]
        material_removal: Option<String>,
    },
    #[serde(rename = "revisionTag")]
    RevisionTag {
        name: String,
        #[serde(rename = "xmm", default)]
        x_mm: f64,
        #[serde(rename = "ymm", default)]
        y_mm: f64,
        #[serde(rename = "tag")]
        revision: String,
        #[serde(default, skip_serializing_if = "Option::is_none")]
        characteristic: Option<String>,
        #[serde(
            rename = "materialRemoval",
            default,
            skip_serializing_if = "Option::is_none"
        )]
        material_removal: Option<String>,
    },
    #[serde(rename = "revisionCloud")]
    RevisionCloud {
        name: String,
        #[serde(rename = "xmm", default)]
        x_mm: f64,
        #[serde(rename = "ymm", default)]
        y_mm: f64,
        #[serde(rename = "widthMm")]
        width_mm: f64,
        #[serde(rename = "heightMm")]
        height_mm: f64,
        #[serde(default, skip_serializing_if = "String::is_empty")]
        tag: String,
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
            Self::PartsList { name, .. }
            | Self::Note { name, .. }
            | Self::CustomTable { name, .. }
            | Self::RevisionTable { name, .. }
            | Self::RevisionTag { name, .. }
            | Self::DatumFeature { name, .. }
            | Self::RevisionCloud { name, .. } => name,
        }
    }
    /// Check whether a definition requires a model, as a parts list does.
    pub fn requires_model(&self) -> bool {
        matches!(self, Self::PartsList { .. })
    }
    pub(super) fn validate(&self) -> Result<(), DrawingError> {
        self.validate_content()?;
        require_label("annotation name", self.name())?;
        let (point, leader, characteristic, material_removal) = match self {
            Self::CustomTable {
                x_mm,
                y_mm,
                characteristic,
                material_removal,
                ..
            }
            | Self::PartsList {
                x_mm,
                y_mm,
                characteristic,
                material_removal,
                ..
            }
            | Self::RevisionTable {
                x_mm,
                y_mm,
                characteristic,
                material_removal,
                ..
            }
            | Self::RevisionTag {
                x_mm,
                y_mm,
                characteristic,
                material_removal,
                ..
            }
            | Self::RevisionCloud {
                x_mm,
                y_mm,
                characteristic,
                material_removal,
                ..
            }
            | Self::DatumFeature {
                x_mm,
                y_mm,
                characteristic,
                material_removal,
                ..
            } => ([*x_mm, *y_mm], [0., 0.], characteristic, material_removal),
            Self::Note {
                x_mm,
                y_mm,
                leader_x_mm,
                leader_y_mm,
                characteristic,
                material_removal,
                ..
            } => (
                [*x_mm, *y_mm],
                [*leader_x_mm, *leader_y_mm],
                characteristic,
                material_removal,
            ),
        };
        validate_recipe_defaults(characteristic.as_deref(), material_removal.as_deref())?;
        if point
            .into_iter()
            .chain(leader)
            .any(|value| !value.is_finite())
        {
            return Err(DrawingError(format!(
                "annotation {:?} point {point:?}, leader {leader:?}: expected finite sheet coordinates",
                self.name()
            )));
        }
        Ok(())
    }

    fn validate_content(&self) -> Result<(), DrawingError> {
        match self {
            Self::DatumFeature { letter, .. } => require_label("datum letter", letter),
            Self::PartsList { .. } => Ok(()),
            Self::Note { text, .. } => require_label("note text", text),
            Self::CustomTable { headers, rows, .. } => super::table::validate_cells(headers, rows),
            Self::RevisionTable { revisions, .. } => super::revision::validate_rows(revisions),
            Self::RevisionTag { revision, .. } => require_label("revision tag", revision),
            Self::RevisionCloud {
                x_mm,
                y_mm,
                width_mm,
                height_mm,
                tag,
                ..
            } => {
                super::revision_cloud::validate_cloud([*x_mm, *y_mm], [*width_mm, *height_mm], tag)
            }
        }
    }
}

pub(super) fn unique_name(requested: &str, annotations: &[DrawingAnnotationDefinition]) -> String {
    let exists = |name: &str| {
        annotations
            .iter()
            .any(|annotation| annotation.name() == name)
    };
    if !requested.is_empty() && !exists(requested) {
        return requested.into();
    }
    let mut number = annotations.len() + 1;
    loop {
        let name = format!("NOTE:{number}");
        if !exists(&name) {
            return name;
        }
        number += 1;
    }
}

fn validate_recipe_defaults(
    characteristic: Option<&str>,
    material_removal: Option<&str>,
) -> Result<(), DrawingError> {
    if characteristic.is_some_and(|value| value != "enum(?)")
        || material_removal.is_some_and(|value| value != "any")
    {
        return Err(DrawingError(format!(
            "annotation characteristic {characteristic:?}, material removal {material_removal:?}: expected the Go default values"
        )));
    }
    Ok(())
}
