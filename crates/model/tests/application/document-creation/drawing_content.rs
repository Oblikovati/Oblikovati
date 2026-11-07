// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::{BASE_VIEW_FRONT, SHEET_LANDSCAPE, SHEET_SIZE_A3};
use oblikovati_model::drawing::{DrawingBaseViewDefinition, DrawingDocument};
#[test]
fn a_new_drawing_owns_one_bordered_a3_landscape_sheet() {
    let drawing = DrawingDocument::new();
    let sheet = &drawing.sheets()[0];
    assert_eq!(drawing.active_sheet(), 0);
    assert_eq!(drawing.model_reference(), "");
    assert_eq!(sheet.name, "Sheet:1");
    assert_eq!(
        (sheet.size, sheet.orientation),
        (SHEET_SIZE_A3, SHEET_LANDSCAPE)
    );
    assert_eq!(sheet.dimensions_mm().unwrap(), [420., 297.]);
    assert!(sheet.border);
    assert_eq!(sheet.title_block, "Default");
    assert!(sheet.views.is_empty());
}
#[test]
fn a_view_definition_requires_a_source_and_retains_its_requested_placement() {
    let mut drawing = DrawingDocument::new();
    let view = DrawingBaseViewDefinition::new("Front", BASE_VIEW_FRONT, 0.5, [90., 120.]).unwrap();
    assert!(drawing.add_base_view_definition(0, view.clone()).is_err());
    drawing.set_model_reference("/project/bracket.opd").unwrap();
    drawing.add_base_view_definition(0, view.clone()).unwrap();
    assert_eq!(
        drawing.sheets()[0].views.as_slice(),
        std::slice::from_ref(&view)
    );
    assert!(
        drawing
            .add_base_view_definition(0, view)
            .unwrap_err()
            .to_string()
            .contains("Front")
    );
    assert!(drawing.set_active_sheet(1).is_err());
    assert!(drawing.set_model_reference("bad\0path").is_err());
    assert_eq!(drawing.model_reference(), "/project/bracket.opd");
}
#[test]
fn invalid_view_dimensions_are_refused_before_a_definition_exists() {
    for scale in [0., -1., f64::NAN, f64::INFINITY] {
        assert!(DrawingBaseViewDefinition::new("Front", BASE_VIEW_FRONT, scale, [0.; 2]).is_err());
    }
    assert!(DrawingBaseViewDefinition::new("", BASE_VIEW_FRONT, 1., [0.; 2]).is_err());
    assert!(DrawingBaseViewDefinition::new("Front", BASE_VIEW_FRONT, 1., [f64::NAN, 0.]).is_err());
}

#[test]
fn sheets_own_border_and_title_property_bindings_instead_of_gui_placeholders() {
    use oblikovati_model::property::SUMMARY_INFORMATION;
    let mut sheet = oblikovati_model::drawing::DrawingSheetRecipe::a3_landscape("Second");
    assert_eq!(sheet.border_margins_mm(), Some([20., 10., 10., 10.]));
    let fields = sheet.title_block_fields().unwrap().unwrap();
    assert_eq!(fields.len(), 6);
    assert_eq!(
        (
            fields[0].label,
            fields[0].property_set,
            fields[0].property_name
        ),
        ("Title", SUMMARY_INFORMATION, "Title")
    );
    assert_eq!(fields[1].property_name, "Part Number");
    sheet.border = false;
    sheet.title_block.clear();
    assert!(sheet.border_margins_mm().is_none() && sheet.title_block_fields().unwrap().is_none());
}
#[test]
fn sheet_edits_validate_custom_dimensions_names_and_enum_values_atomically() {
    use oblikovati_api::types::{SHEET_SIZE_CUSTOM, SheetOrientation};
    use oblikovati_model::drawing::DrawingSheetRecipe;
    let mut drawing = DrawingDocument::default();
    drawing
        .add_sheet(DrawingSheetRecipe::a3_landscape("Second"))
        .unwrap();
    drawing.set_active_sheet(0).unwrap();
    assert!(
        drawing
            .add_sheet(DrawingSheetRecipe::a3_landscape("Second"))
            .is_err()
    );
    let mut custom = DrawingSheetRecipe::a3_landscape("Custom");
    custom.size = SHEET_SIZE_CUSTOM;
    assert!(drawing.add_sheet(custom.clone()).is_err());
    custom.width_mm = 100.;
    custom.height_mm = 80.;
    assert_eq!(custom.dimensions_mm().unwrap(), [100., 80.]);
    custom.orientation = SheetOrientation(99);
    assert!(custom.dimensions_mm().is_err());
    assert_eq!(drawing.sheets().len(), 2);
}
#[test]
fn invalid_restored_recipes_are_refused_without_repair_or_data_loss() {
    let original = DrawingDocument::new();
    let mut recipe = original.recipe().clone();
    recipe.sheets.clear();
    assert!(
        DrawingDocument::from_recipe(recipe)
            .unwrap_err()
            .to_string()
            .contains("sheet count 0")
    );
    let mut recipe = original.recipe().clone();
    recipe.sheets[0].title_block = "Unknown".into();
    assert!(
        DrawingDocument::from_recipe(recipe)
            .unwrap_err()
            .to_string()
            .contains("Unknown")
    );
    let mut recipe = original.recipe().clone();
    recipe.sheets.push(recipe.sheets[0].clone());
    assert!(DrawingDocument::from_recipe(recipe).is_err());
    assert!(
        DrawingBaseViewDefinition::new(
            "Front",
            oblikovati_api::types::BaseViewOrientation(99),
            1.,
            [0.; 2]
        )
        .is_err()
    );
}

#[test]
fn a_standard_sheet_cannot_retain_nonfinite_custom_dimension_fields() {
    let mut recipe = DrawingDocument::new().recipe().clone();
    recipe.sheets[0].width_mm = f64::NAN;
    assert!(DrawingDocument::from_recipe(recipe).is_err());
}
