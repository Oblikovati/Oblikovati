// SPDX-License-Identifier: GPL-2.0-only
//! Drawing browser branches derived from the authored sheets and views.
use oblikovati_model::drawing::{DrawingDocument, DrawingSheetRecipe};

pub(crate) type DrawingBranch = (String, Vec<String>);

pub(crate) fn drawing_tree(drawing: Option<&DrawingDocument>) -> Vec<DrawingBranch> {
    let resources = ["Sheet Formats", "Default Border", "Default Title Block"];
    let mut branches = vec![(
        "Drawing Resources".into(),
        resources.map(String::from).into(),
    )];
    if let Some(drawing) = drawing {
        branches.extend(
            drawing
                .sheets()
                .iter()
                .map(|sheet| sheet_branch(sheet, drawing.model_reference())),
        );
    }
    branches
}
fn sheet_branch(sheet: &DrawingSheetRecipe, reference: &str) -> DrawingBranch {
    let mut rows = Vec::new();
    if sheet.border {
        rows.push("Default Border".into());
    }
    if !sheet.title_block.is_empty() {
        rows.push("Title Block".into());
    }
    let model = std::path::Path::new(reference)
        .file_name()
        .and_then(std::ffi::OsStr::to_str);
    rows.extend(sheet.views.iter().map(|view| match model {
        Some(model) if !sheet.annotations.is_empty() => format!("{} ({model})", view.name),
        _ => view.name.clone(),
    }));
    rows.extend(
        sheet
            .annotations
            .iter()
            .map(|annotation| annotation.name().to_owned()),
    );
    (sheet.name.clone(), rows)
}

pub(crate) const SHEET_TITLE_BLOCK: &str = "Title Block";
/// What a title block holds, one level under its row on the sheet.
pub(crate) const TITLE_BLOCK_FIELD: &str = "Field Text";

pub(crate) fn drawing_row_glyph(name: &str) -> crate::icons::IconGlyph {
    use crate::icons::IconGlyph;
    if name.contains("Border") || name.contains("Sheet") {
        IconGlyph::DrawingSheet
    } else if name.contains("Title") || name.contains("Note") || name.contains("Text") {
        IconGlyph::Annotation
    } else if name.contains("Weld") {
        IconGlyph::DrawingWeld
    } else if name.contains("Surface") {
        IconGlyph::DrawingSurface
    } else {
        IconGlyph::DrawingBase
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::drawing::DrawingBaseViewDefinition;
    #[test]
    fn tree_follows_the_authored_sheets_and_views() {
        let mut drawing = DrawingDocument::new();
        drawing.set_model_reference("source.opd").unwrap();
        let mut second = DrawingSheetRecipe::a3_landscape("Detail");
        second.border = false;
        drawing.add_sheet(second).unwrap();
        drawing
            .add_base_view_definition(
                1,
                DrawingBaseViewDefinition::new(
                    "Front",
                    oblikovati_api::types::BASE_VIEW_FRONT,
                    1.,
                    [20., 30.],
                )
                .unwrap(),
            )
            .unwrap();
        let branches = drawing_tree(Some(&drawing));
        assert_eq!(branches[1].0, "Sheet:1");
        assert_eq!(
            branches[2],
            ("Detail".into(), vec!["Title Block".into(), "Front".into()])
        );
        assert_eq!(drawing_tree(None).len(), 1);
    }
}
