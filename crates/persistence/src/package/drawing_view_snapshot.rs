// SPDX-License-Identifier: GPL-2.0-only
//! Saved view geometry is a presentation cache and is never used as model geometry.
use super::{DocumentPackage, PackageCodecError};
use oblikovati_model::drawing::{DrawingDocument, DrawingRecipe};
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;

pub(super) const SECTION: &str = "presentation/drawing-views-v1";
/// A line sampled in sheet millimetres for deferred presentation.
/// Example: `SavedDrawingLine { points: vec![[0., 0.], [1., 0.]], hidden: false }`.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct SavedDrawingLine {
    pub points: Vec<[f64; 2]>,
    pub hidden: bool,
}
/// One saved view, identified by its authored sheet and view names.
/// Example: `SavedDrawingView { sheet: "Sheet:1".into(), view: "Front".into(), lines: vec![] }`.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct SavedDrawingView {
    pub sheet: String,
    pub view: String,
    pub lines: Vec<SavedDrawingLine>,
}
/// The display cache made for one saved drawing recipe.
/// Example: `SavedDrawingViews { version: SavedDrawingViews::VERSION, drawing_recipe: drawing.recipe().clone(), views: vec![] }`.
#[derive(Clone, Debug, PartialEq, Serialize, Deserialize)]
#[serde(rename_all = "camelCase", deny_unknown_fields)]
pub struct SavedDrawingViews {
    pub version: u32,
    pub drawing_recipe: DrawingRecipe,
    pub views: Vec<SavedDrawingView>,
}
impl SavedDrawingViews {
    /// The current stream format; example: `views.version == SavedDrawingViews::VERSION`.
    pub const VERSION: u32 = 1;
}
impl DocumentPackage {
    /// Store a complete display cache for this drawing; example: `package.with_saved_drawing_views(&views)?`.
    pub fn with_saved_drawing_views(
        &self,
        views: &SavedDrawingViews,
    ) -> Result<Self, PackageCodecError> {
        validate_for_package(self, views)?;
        let section = serde_yaml_ng::to_string(views).map_err(|cause| {
            PackageCodecError(format!(
                "saved drawing views {views:?}: expected serializable geometry: {cause}"
            ))
        })?;
        Ok(self.with_stream(SECTION, section.as_bytes()))
    }
    /// Read a validated saved display cache; example: `package.restore_saved_drawing_views()?`.
    pub fn restore_saved_drawing_views(
        &self,
    ) -> Result<Option<SavedDrawingViews>, PackageCodecError> {
        let Some(section) = self.streams().get(SECTION) else {
            return Ok(None);
        };
        let views: SavedDrawingViews =
            serde_yaml_ng::from_slice(section).map_err(|cause| {
                PackageCodecError(format!(
                    "saved drawing view stream {SECTION:?} ({} bytes): expected versioned view geometry: {cause}", section.len()
                ))
            })?;
        validate_for_package(self, &views)?;
        Ok(Some(views))
    }
}
fn validate_for_package(
    package: &DocumentPackage,
    views: &SavedDrawingViews,
) -> Result<(), PackageCodecError> {
    let drawing = package.restore_drawing()?.ok_or_else(|| {
        PackageCodecError(format!(
            "saved drawing views for document type {}: expected a drawing",
            package.manifest().document_type
        ))
    })?;
    validate_views(&drawing, views)
}
fn validate_views(
    drawing: &DrawingDocument,
    views: &SavedDrawingViews,
) -> Result<(), PackageCodecError> {
    if views.version != SavedDrawingViews::VERSION || &views.drawing_recipe != drawing.recipe() {
        return Err(PackageCodecError(format!(
            "saved drawing views version {} and recipe {:?}: expected version {} and recipe {:?}",
            views.version,
            views.drawing_recipe,
            SavedDrawingViews::VERSION,
            drawing.recipe()
        )));
    }
    let expected: BTreeSet<_> = drawing
        .sheets()
        .iter()
        .flat_map(|sheet| {
            sheet
                .views
                .iter()
                .map(|view| (sheet.name.as_str(), view.name.as_str()))
        })
        .collect();
    let mut actual = BTreeSet::new();
    for view in &views.views {
        if !actual.insert((view.sheet.as_str(), view.view.as_str())) {
            return Err(PackageCodecError(format!(
                "saved drawing view {:?}/{:?}: expected a unique view",
                view.sheet, view.view
            )));
        }
        validate_lines(view)?;
    }
    if actual != expected {
        return Err(PackageCodecError(format!(
            "saved drawing views {actual:?}: expected the authored views {expected:?}"
        )));
    }
    Ok(())
}
fn validate_lines(view: &SavedDrawingView) -> Result<(), PackageCodecError> {
    for line in &view.lines {
        if line.points.len() < 2 || line.points.iter().flatten().any(|point| !point.is_finite()) {
            return Err(PackageCodecError(format!(
                "saved drawing view {:?}/{:?} line {:?}: expected at least two finite points",
                view.sheet, view.view, line.points
            )));
        }
    }
    Ok(())
}
