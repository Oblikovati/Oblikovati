// SPDX-License-Identifier: GPL-2.0-only
use super::{DrawingError, DrawingRecipe, DrawingSheetRecipe};
use std::collections::BTreeSet;
pub(super) fn require_label(role: &str, label: &str) -> Result<(), DrawingError> {
    if label.trim().is_empty() || label.contains('\0') {
        return Err(DrawingError(format!(
            "{role} {label:?}: expected nonempty text without NUL"
        )));
    }
    Ok(())
}
pub(super) fn require_model_reference(name: &str) -> Result<(), DrawingError> {
    if name.contains('\0') {
        return Err(DrawingError(format!(
            "model reference {name:?}: expected a document name without NUL"
        )));
    }
    Ok(())
}
pub(super) fn validate_drawing(recipe: &DrawingRecipe) -> Result<(), DrawingError> {
    require_model_reference(&recipe.model_reference)?;
    validate_header(recipe)?;
    let mut names = BTreeSet::new();
    for sheet in &recipe.sheets {
        validate_sheet(sheet)?;
        if !sheet.annotations.is_empty() && recipe.model_reference.is_empty() {
            return Err(DrawingError(format!(
                "sheet {:?} annotations: expected a referenced model",
                sheet.name
            )));
        }
        if !names.insert(&sheet.name) {
            return Err(DrawingError(format!(
                "sheet {:?}: expected a unique sheet name",
                sheet.name
            )));
        }
    }
    Ok(())
}
fn validate_header(recipe: &DrawingRecipe) -> Result<(), DrawingError> {
    if recipe.sheets.is_empty() || recipe.active_sheet >= recipe.sheets.len() {
        return Err(DrawingError(format!(
            "active sheet {}, sheet count {}: expected at least one sheet and an in-range zero-based index",
            recipe.active_sheet,
            recipe.sheets.len()
        )));
    }
    if super::spelling::drafting_standard::name(&recipe.standard).is_none() {
        return Err(DrawingError(format!(
            "drafting standard {:?}: expected a canonical drafting standard",
            recipe.standard
        )));
    }
    Ok(())
}
pub(super) fn validate_sheet(sheet: &DrawingSheetRecipe) -> Result<(), DrawingError> {
    require_label("sheet name", &sheet.name)?;
    sheet.dimensions_mm()?;
    validate_sheet_format(sheet)?;
    let mut names = BTreeSet::new();
    for view in &sheet.views {
        view.validate()?;
        if !names.insert(&view.name) {
            return Err(DrawingError(format!(
                "view {:?}: expected a unique name within its sheet",
                view.name
            )));
        }
    }
    let mut annotation_names = BTreeSet::new();
    for annotation in &sheet.annotations {
        annotation.validate()?;
        if !annotation_names.insert(annotation.name()) {
            return Err(DrawingError(format!(
                "annotation {:?}: expected a unique name within its sheet",
                annotation.name()
            )));
        }
    }
    Ok(())
}
fn validate_sheet_format(sheet: &DrawingSheetRecipe) -> Result<(), DrawingError> {
    sheet.title_block_fields()?;
    if super::spelling::title_block_location::name(&sheet.title_block_at).is_none() {
        return Err(DrawingError(format!(
            "sheet {:?} title-block location {:?}: expected a canonical sheet corner",
            sheet.name, sheet.title_block_at
        )));
    }
    if [&sheet.title_block, &sheet.revision]
        .iter()
        .any(|text| text.contains('\0'))
    {
        return Err(DrawingError(format!(
            "sheet text {:?}, revision {:?}: expected text without NUL",
            sheet.title_block, sheet.revision
        )));
    }
    Ok(())
}
