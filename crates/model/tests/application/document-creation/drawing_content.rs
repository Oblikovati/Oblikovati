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
fn border_zone_labels_use_spreadsheet_columns_and_independent_row_modes() {
    use oblikovati_api::types::{ALPHABETICAL_BORDER_LABEL, NO_BORDER_LABEL, NUMERIC_BORDER_LABEL};
    use oblikovati_model::drawing::border_zone_label;
    for (index, label) in [(0, "A"), (25, "Z"), (26, "AA"), (701, "ZZ"), (702, "AAA")] {
        assert_eq!(border_zone_label(index, ALPHABETICAL_BORDER_LABEL), label);
    }
    assert_eq!(border_zone_label(26, NUMERIC_BORDER_LABEL), "27");
    assert!(border_zone_label(26, NO_BORDER_LABEL).is_empty());
}

#[test]
fn malformed_zoned_border_recipes_are_rejected() {
    let mut drawing = DrawingDocument::new();
    drawing
        .set_zoned_border(0, [4, 3], ["numeric", ""])
        .unwrap();
    let mut incomplete = drawing.recipe().clone();
    incomplete.sheets[0].border_v_zones = 0;
    assert!(DrawingDocument::from_recipe(incomplete).is_err());
    let mut missing_border = drawing.recipe().clone();
    missing_border.sheets[0].border = false;
    assert!(DrawingDocument::from_recipe(missing_border).is_err());
}

#[test]
fn zoned_border_uses_the_go_sheet_recipe_keys() {
    use oblikovati_model::drawing::DrawingRecipe;
    let recipe = serde_json::json!({
        "sheets": [{
            "name": "Sheet:1", "size": "a3", "orientation": "landscape", "border": true,
            "borderHZones": 4, "borderVZones": 3, "borderHLabel": "numeric", "borderVLabel": "none"
        }]
    });
    let decoded: DrawingRecipe = serde_json::from_value(recipe.clone()).unwrap();
    let drawing = DrawingDocument::from_recipe(decoded).unwrap();
    let written = serde_json::to_value(drawing.recipe()).unwrap();
    for key in [
        "borderHZones",
        "borderVZones",
        "borderHLabel",
        "borderVLabel",
    ] {
        assert_eq!(written["sheets"][0][key], recipe["sheets"][0][key]);
    }
}

#[test]
fn reusable_formats_require_unique_blank_prototypes() {
    use oblikovati_model::drawing::{DrawingRecipe, DrawingSheetFormat, DrawingSheetRecipe};
    let mut drawing = DrawingDocument::new();
    let paper = DrawingSheetRecipe::a3_landscape("Plate");
    assert!(DrawingSheetFormat::new(paper.clone(), [-1, 0], ["", ""], "").is_err());
    let plain = DrawingSheetFormat::new(paper, [4, 0], ["numeric", ""], "").unwrap();
    drawing.define_sheet_format(plain.clone()).unwrap();
    drawing.add_sheet_using_format("Plain", "Plate").unwrap();
    assert_eq!(drawing.sheets()[1].border_h_zones, 0);
    let mut duplicate = drawing.recipe().clone();
    duplicate.sheet_formats.push(plain);
    assert!(DrawingDocument::from_recipe(duplicate).is_err());

    let mut stored = serde_json::to_value(drawing.recipe()).unwrap();
    stored["sheetFormats"][0]["revision"] = "B".into();
    let invalid: DrawingRecipe = serde_json::from_value(stored).unwrap();
    assert!(DrawingDocument::from_recipe(invalid).is_err());
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

#[test]
fn parts_list_definitions_require_a_model_and_unique_finite_names() {
    use oblikovati_model::drawing::DrawingAnnotationDefinition;
    let mut drawing = DrawingDocument::new();
    let list = DrawingAnnotationDefinition::parts_list("Parts", [40., 200.]).unwrap();
    assert_eq!(list.name(), "Parts");
    assert!(drawing.add_annotation_definition(0, list.clone()).is_err());
    let mut annotated = oblikovati_model::drawing::DrawingSheetRecipe::a3_landscape("Extra");
    annotated.annotations.push(list.clone());
    assert!(drawing.add_sheet(annotated).is_err());
    assert_eq!(drawing.sheets().len(), 1);
    drawing.set_model_reference("/project/bracket.opd").unwrap();
    drawing.add_annotation_definition(0, list.clone()).unwrap();
    assert_eq!(
        drawing.sheets()[0].annotations.as_slice(),
        std::slice::from_ref(&list)
    );
    assert!(drawing.add_annotation_definition(0, list).is_err());
    assert!(drawing.set_model_reference("").is_err());
    assert!(DrawingAnnotationDefinition::parts_list("", [1., 2.]).is_err());
    assert!(DrawingAnnotationDefinition::parts_list("Parts", [f64::NAN, 2.]).is_err());
    let mut bad = drawing.recipe().clone();
    bad.model_reference.clear();
    assert!(DrawingDocument::from_recipe(bad).is_err());
    let mut bad = drawing.recipe().clone();
    let duplicate = bad.sheets[0].annotations[0].clone();
    bad.sheets[0].annotations.push(duplicate);
    assert!(DrawingDocument::from_recipe(bad).is_err());
}

#[test]
fn parts_list_rebuilds_go_table_geometry_from_current_source_rows() {
    use oblikovati_model::drawing::{
        DrawingAnnotationDefinition, DrawingPartsListRow, DrawingPartsListSource,
    };
    use std::cell::{Cell, RefCell};
    struct FakePartsSource {
        rows: RefCell<Vec<DrawingPartsListRow>>,
        calls: Cell<usize>,
    }
    impl DrawingPartsListSource for FakePartsSource {
        fn parts_list_rows(&self, reference: &str) -> Result<Vec<DrawingPartsListRow>, String> {
            self.calls.set(self.calls.get() + 1);
            if reference != "assembly.oba" {
                return Err(format!("model {reference:?}: expected assembly.oba"));
            }
            Ok(self.rows.borrow().clone())
        }
    }
    let source = FakePartsSource {
        calls: Cell::new(0),
        rows: RefCell::new(vec![DrawingPartsListRow {
            item: 1,
            part_number: "BRKT-01".into(),
            description: "Bracket".into(),
            quantity: 2,
        }]),
    };
    let list = DrawingAnnotationDefinition::parts_list("Parts", [40., 200.]).unwrap();
    let first = list.evaluate_parts_list("assembly.oba", &source).unwrap();
    assert_eq!(first.row_count, 1);
    assert_eq!(first.lines.len(), 8);
    assert_eq!(first.labels.len(), 8);
    assert_eq!(first.lines[0].start, [40., 200.]);
    assert_eq!(first.lines[0].end, [154., 200.]);
    assert_eq!(first.labels[1].text, "PART NUMBER");
    assert_eq!(first.labels[5].text, "BRKT-01");
    assert_eq!(first.labels[5].center, [72., 188.]);
    assert_eq!(first.labels[5].cell_size, [40., 8.]);
    source.rows.borrow_mut().push(DrawingPartsListRow {
        item: 2,
        part_number: "BOLT".into(),
        description: "Bolt".into(),
        quantity: 4,
    });
    let updated = list.evaluate_parts_list("assembly.oba", &source).unwrap();
    assert_eq!(
        (updated.row_count, updated.lines.len(), updated.labels.len()),
        (2, 9, 12)
    );
    assert_eq!(updated.labels[9].text, "BOLT");
    let mut drawing = DrawingDocument::new();
    drawing
        .add_note("Finish", [40., 90.], "Deburr", [0., 0.])
        .unwrap();
    let before = source.calls.get();
    assert!(drawing.evaluate_parts_lists(0, &source).unwrap().is_empty());
    assert_eq!(source.calls.get(), before);
    drawing.set_model_reference("assembly.oba").unwrap();
    drawing.add_annotation_definition(0, list.clone()).unwrap();
    drawing
        .add_annotation_definition(
            0,
            DrawingAnnotationDefinition::parts_list("Other", [10., 20.]).unwrap(),
        )
        .unwrap();
    let before = source.calls.get();
    let tables = drawing.evaluate_parts_lists(0, &source).unwrap();
    assert_eq!(source.calls.get() - before, 1);
    assert_eq!(tables.len(), 2);
    assert_eq!(tables[0].0, "Parts");
    assert_eq!(tables[0].1, updated);
    assert!(drawing.evaluate_parts_lists(1, &source).is_err());
    drawing.set_model_reference("missing.oba").unwrap();
    assert!(drawing.evaluate_parts_lists(0, &source).is_err());
    assert!(list.evaluate_parts_list("missing.oba", &source).is_err());
    assert!(list.evaluate_parts_list("", &source).is_err());
    let invalid = DrawingAnnotationDefinition::PartsList {
        name: "bad".into(),
        x_mm: f64::NAN,
        y_mm: 0.,
        characteristic: None,
        material_removal: None,
    };
    let before = source.calls.get();
    assert!(
        invalid
            .evaluate_parts_list("assembly.oba", &source)
            .is_err()
    );
    assert_eq!(
        source.calls.get(),
        before,
        "invalid input must not query the source"
    );
}
