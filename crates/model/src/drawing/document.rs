// SPDX-License-Identifier: GPL-2.0-only
use super::{
    DrawingAnnotationDefinition, DrawingBaseViewDefinition, DrawingError, DrawingRecipe,
    DrawingSheetRecipe,
    validation::{validate_drawing, validate_sheet},
};
/// One owner of the sheet and view definitions; workspace resolvers never enter history snapshots.
#[derive(Clone, Debug, PartialEq)]
pub struct DrawingDocument {
    recipe: DrawingRecipe,
}
impl Default for DrawingDocument {
    fn default() -> Self {
        Self::new()
    }
}
impl DrawingDocument {
    /// Create the default drawing content; example: `DrawingDocument::new().sheets().len() == 1`.
    pub fn new() -> Self {
        Self {
            recipe: DrawingRecipe {
                model_reference: String::new(),
                standard: oblikovati_api::types::DRAFTING_ISO,
                active_sheet: 0,
                sheets: vec![DrawingSheetRecipe::a3_landscape("Sheet:1")],
            },
        }
    }
    /// Admit an owned recipe only after checking every sheet and view definition.
    /// Example: `DrawingDocument::from_recipe(saved_recipe)?` on reopen.
    pub fn from_recipe(recipe: DrawingRecipe) -> Result<Self, DrawingError> {
        validate_drawing(&recipe)?;
        Ok(Self { recipe })
    }
    /// Read the complete immutable recipe; example: `drawing.recipe()` before persistence.
    pub fn recipe(&self) -> &DrawingRecipe {
        &self.recipe
    }
    /// Read ordered sheets; example: `drawing.sheets()[0].name`.
    pub fn sheets(&self) -> &[DrawingSheetRecipe] {
        &self.recipe.sheets
    }
    /// Read the zero-based active sheet; example: `drawing.active_sheet()`.
    pub fn active_sheet(&self) -> usize {
        self.recipe.active_sheet
    }
    /// Read the unresolved document reference; example: `drawing.model_reference()` for workspace resolution.
    pub fn model_reference(&self) -> &str {
        &self.recipe.model_reference
    }
    /// Change the model reference without resolving or evaluating it.
    /// Example: `drawing.set_model_reference("/project/bracket.opd")?`.
    pub fn set_model_reference(&mut self, name: &str) -> Result<(), DrawingError> {
        super::validation::require_model_reference(name)?;
        if name.is_empty()
            && self
                .recipe
                .sheets
                .iter()
                .any(|sheet| !sheet.annotations.is_empty())
        {
            return Err(DrawingError(format!(
                "model reference {name:?}: expected a referenced model while annotations exist"
            )));
        }
        self.recipe.model_reference = name.into();
        Ok(())
    }
    /// Make an existing sheet active; example: `drawing.set_active_sheet(0)?`.
    pub fn set_active_sheet(&mut self, index: usize) -> Result<(), DrawingError> {
        if index >= self.recipe.sheets.len() {
            return Err(DrawingError(format!(
                "sheet index {index}: expected an index below {}",
                self.recipe.sheets.len()
            )));
        }
        self.recipe.active_sheet = index;
        Ok(())
    }
    /// Append a checked sheet and activate it; example: `drawing.add_sheet(DrawingSheetRecipe::a3_landscape("Second"))?`.
    pub fn add_sheet(&mut self, sheet: DrawingSheetRecipe) -> Result<(), DrawingError> {
        validate_sheet(&sheet)?;
        if !sheet.annotations.is_empty() && self.recipe.model_reference.is_empty() {
            return Err(DrawingError(format!(
                "sheet {:?} annotations: expected a referenced model",
                sheet.name
            )));
        }
        if self
            .recipe
            .sheets
            .iter()
            .any(|existing| existing.name == sheet.name)
        {
            return Err(DrawingError(format!(
                "sheet {:?}: expected a unique sheet name",
                sheet.name
            )));
        }
        self.recipe.sheets.push(sheet);
        self.recipe.active_sheet = self.recipe.sheets.len() - 1;
        Ok(())
    }
    /// Append an authored base-view definition; the host must project real geometry before presenting it.
    /// Example: `drawing.add_base_view_definition(0, view)?`.
    pub fn add_base_view_definition(
        &mut self,
        sheet: usize,
        view: DrawingBaseViewDefinition,
    ) -> Result<(), DrawingError> {
        view.validate()?;
        let target = self.referenced_sheet(sheet)?;
        if target
            .views
            .iter()
            .any(|existing| existing.name == view.name)
        {
            return Err(DrawingError(format!(
                "view {:?}: expected a unique name within sheet {sheet}",
                view.name
            )));
        }
        target.views.push(view);
        Ok(())
    }
    /// Delete a named view from one sheet; example: `drawing.remove_view(0, "Front")?`.
    pub fn remove_view(&mut self, sheet: usize, name: &str) -> Result<(), DrawingError> {
        let count = self.recipe.sheets.len();
        let target = self.recipe.sheets.get_mut(sheet).ok_or_else(|| {
            DrawingError(format!(
                "sheet index {sheet}: expected an index below {count}"
            ))
        })?;
        let index = target
            .views
            .iter()
            .position(|view| view.name == name)
            .ok_or_else(|| {
                DrawingError(format!(
                    "drawing view {name:?}: expected a view on sheet {sheet}"
                ))
            })?;
        target.views.remove(index);
        Ok(())
    }
    /// Append an authored parts list; example: `drawing.add_annotation_definition(0, annotation)?`.
    pub fn add_annotation_definition(
        &mut self,
        sheet: usize,
        annotation: DrawingAnnotationDefinition,
    ) -> Result<(), DrawingError> {
        annotation.validate()?;
        let target = self.referenced_sheet(sheet)?;
        if target
            .annotations
            .iter()
            .any(|existing| existing.name() == annotation.name())
        {
            return Err(DrawingError(format!(
                "annotation {:?}: expected a unique name within sheet {sheet}",
                annotation.name()
            )));
        }
        target.annotations.push(annotation);
        Ok(())
    }
    fn referenced_sheet(&mut self, sheet: usize) -> Result<&mut DrawingSheetRecipe, DrawingError> {
        if self.recipe.model_reference.is_empty() {
            return Err(DrawingError("model reference \"\": expected a referenced document before adding model-dependent drawing content".into()));
        }
        let count = self.recipe.sheets.len();
        self.recipe.sheets.get_mut(sheet).ok_or_else(|| {
            DrawingError(format!(
                "sheet index {sheet}: expected an index below {count}"
            ))
        })
    }
}
