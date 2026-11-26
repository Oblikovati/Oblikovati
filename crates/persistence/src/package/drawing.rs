// SPDX-License-Identifier: GPL-2.0-only
//! The drawing's Go-compatible model section is captured by the same snapshot save as other content.
use super::{DocumentPackage, Mapping, PackageCodecError, Value};
use oblikovati_api::types::DOCUMENT_DRAWING;
use oblikovati_model::drawing::{DrawingDocument, DrawingRecipe};
const RECIPE_FIELDS: &[&str] = &[
    "modelReference",
    "standard",
    "activeSheet",
    "sheets",
    "sheetFormats",
];
impl DocumentPackage {
    /// Restore owned drawing content by document kind, without resolving or projecting a model.
    /// Example: `let drawing = package.restore_drawing()?;` before workspace registration.
    pub fn restore_drawing(&self) -> Result<Option<DrawingDocument>, PackageCodecError> {
        if self.manifest.document_type != DOCUMENT_DRAWING.0 {
            return Ok(None);
        }
        let model = self.root.get("model").and_then(Value::as_mapping);
        let drawing = model.map(drawing_fields).unwrap_or_default();
        let recipe: DrawingRecipe = serde_yaml_ng::from_value(Value::Mapping(drawing.clone()))
            .map_err(|cause| {
                PackageCodecError(format!(
                    "drawing model {drawing:?}: expected a supported drawing recipe: {cause}"
                ))
            })?;
        DrawingDocument::from_recipe(recipe)
            .map(Some)
            .map_err(|cause| PackageCodecError(cause.to_string()))
    }
    /// Capture real drawing definitions while retaining other model and package sections.
    /// Example: `package.with_drawing(Some(&drawing))?` during snapshot save.
    pub fn with_drawing(
        &self,
        drawing: Option<&DrawingDocument>,
    ) -> Result<Self, PackageCodecError> {
        require_drawing_kind(self.manifest.document_type, drawing)?;
        let Some(drawing) = drawing else {
            return Ok(self.clone());
        };
        if self.drawing_is_stored(drawing)? {
            return Ok(self.clone());
        }
        self.capture_drawing(drawing)
    }
    fn drawing_is_stored(&self, drawing: &DrawingDocument) -> Result<bool, PackageCodecError> {
        let has_sheet_recipe = self
            .root
            .get("model")
            .and_then(Value::as_mapping)
            .is_some_and(|model| model.contains_key("sheets"));
        Ok(has_sheet_recipe && self.restore_drawing()?.as_ref() == Some(drawing))
    }
    fn capture_drawing(&self, drawing: &DrawingDocument) -> Result<Self, PackageCodecError> {
        let mut candidate = self.without_stream(super::drawing_view_snapshot::SECTION);
        let mut model = candidate
            .root
            .get("model")
            .and_then(Value::as_mapping)
            .cloned()
            .unwrap_or_default();
        let updated = drawing_mapping(drawing)?;
        for field in RECIPE_FIELDS {
            if !updated.contains_key(*field) {
                model.remove(*field);
            }
        }
        model.extend(updated);
        candidate.root.insert("model".into(), Value::Mapping(model));
        Ok(candidate)
    }
}
fn drawing_fields(model: &Mapping) -> Mapping {
    model
        .iter()
        .filter(|(key, _)| {
            key.as_str()
                .is_some_and(|name| RECIPE_FIELDS.contains(&name))
        })
        .map(|(key, value)| (key.clone(), value.clone()))
        .collect()
}
fn require_drawing_kind(
    kind: u32,
    drawing: Option<&DrawingDocument>,
) -> Result<(), PackageCodecError> {
    if (kind == DOCUMENT_DRAWING.0) != drawing.is_some() {
        return Err(PackageCodecError(format!(
            "document type {kind}, drawing content present {}: expected drawing content exactly for drawing documents",
            drawing.is_some()
        )));
    }
    Ok(())
}

fn drawing_mapping(drawing: &DrawingDocument) -> Result<Mapping, PackageCodecError> {
    let encoded = serde_yaml_ng::to_value(drawing.recipe()).map_err(|cause| {
        PackageCodecError(format!(
            "drawing recipe {:?}: expected serializable native content: {cause}",
            drawing.recipe()
        ))
    })?;
    encoded
        .as_mapping()
        .cloned()
        .ok_or_else(|| PackageCodecError(format!("drawing recipe {encoded:?}: expected a mapping")))
}
