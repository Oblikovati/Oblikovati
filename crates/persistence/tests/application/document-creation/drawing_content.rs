// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::BASE_VIEW_FRONT;
use oblikovati_model::drawing::{DrawingBaseViewDefinition, DrawingDocument};
use oblikovati_persistence::package::{
    DocumentPackage, SavedDrawingLine, SavedDrawingView, SavedDrawingViews,
};

fn saved_view_fixture() -> (DocumentPackage, DrawingDocument, SavedDrawingViews) {
    let mut drawing = DrawingDocument::new();
    drawing.set_model_reference("/project/bracket.opd").unwrap();
    drawing
        .add_base_view_definition(
            0,
            DrawingBaseViewDefinition::new("Front", BASE_VIEW_FRONT, 1., [90., 120.]).unwrap(),
        )
        .unwrap();
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 3\n")
        .unwrap()
        .with_drawing(Some(&drawing))
        .unwrap();
    let views = SavedDrawingViews {
        version: SavedDrawingViews::VERSION,
        drawing_recipe: drawing.recipe().clone(),
        views: vec![SavedDrawingView {
            sheet: "Sheet:1".into(),
            view: "Front".into(),
            lines: vec![SavedDrawingLine {
                points: vec![[0., 1.], [2., 3.]],
                hidden: true,
            }],
        }],
    };
    (package, drawing, views)
}
#[test]
fn saved_view_geometry_round_trips_and_unchanged_recipe_keeps_it() {
    let (package, drawing, views) = saved_view_fixture();
    let saved = package
        .with_saved_drawing_views(&views)
        .unwrap()
        .with_drawing(Some(&drawing))
        .unwrap();
    assert_eq!(
        package.recipe_digest().unwrap(),
        saved.recipe_digest().unwrap()
    );
    let reopened = DocumentPackage::decode(&saved.encode().unwrap()).unwrap();
    assert_eq!(
        reopened.restore_saved_drawing_views().unwrap(),
        Some(views.clone())
    );
    let (compacted, _) = reopened.without_caches().unwrap();
    assert_eq!(
        compacted.restore_saved_drawing_views().unwrap(),
        Some(views)
    );
    assert_eq!(reopened.restore_drawing().unwrap(), Some(drawing));
}
#[test]
fn corrupted_saved_view_stream_is_reported_on_read() {
    let (package, _, _) = saved_view_fixture();
    let corrupted = package.with_stream("presentation/drawing-views-v1", b"version: [broken");
    let reopened = DocumentPackage::decode(&corrupted.encode().unwrap()).unwrap();
    let error = reopened
        .restore_saved_drawing_views()
        .unwrap_err()
        .to_string();
    assert!(error.contains("presentation/drawing-views-v1"), "{error}");
    let (_, _, mut views) = saved_view_fixture();
    views.drawing_recipe.sheets[0].views[0].scale = 2.;
    let bytes = serde_yaml_ng::to_string(&views).unwrap();
    let changed = package.with_stream("presentation/drawing-views-v1", bytes.as_bytes());
    assert!(changed.restore_saved_drawing_views().is_err());
}
#[test]
fn changed_drawing_recipe_discards_old_view_geometry() {
    let (package, mut drawing, views) = saved_view_fixture();
    let saved = package.with_saved_drawing_views(&views).unwrap();
    drawing.set_model_reference("/project/other.opd").unwrap();
    let changed = saved.with_drawing(Some(&drawing)).unwrap();
    assert_eq!(changed.restore_saved_drawing_views().unwrap(), None);
    assert!(changed.with_saved_drawing_views(&views).is_err());
}
#[test]
fn saved_view_geometry_refuses_incomplete_duplicate_and_nonfinite_data() {
    let (package, _, mut views) = saved_view_fixture();
    views.views.clear();
    assert!(package.with_saved_drawing_views(&views).is_err());
    let (_, _, mut views) = saved_view_fixture();
    views.views.push(views.views[0].clone());
    assert!(package.with_saved_drawing_views(&views).is_err());
    views.views.pop();
    views.views[0].lines[0].points[0][0] = f64::NAN;
    assert!(package.with_saved_drawing_views(&views).is_err());
    let (_, _, mut views) = saved_view_fixture();
    views.version += 1;
    assert!(package.with_saved_drawing_views(&views).is_err());
    views.version = SavedDrawingViews::VERSION;
    views.views[0].lines[0].points.truncate(1);
    assert!(package.with_saved_drawing_views(&views).is_err());
    let (_, _, mut views) = saved_view_fixture();
    views.drawing_recipe.sheets[0].views[0].scale = 2.;
    assert!(package.with_saved_drawing_views(&views).is_err());
}
#[test]
fn a_native_go_shaped_drawing_recipe_keeps_its_sheet_reference_and_base_view_definition() {
    let encoded = br#"schemaVersion: 2
documentType: 3
model:
  modelReference: /project/bracket.opd
  standard: ISO
  sheets:
    - name: 'Sheet:1'
      size: a3
      orientation: landscape
      border: true
      titleBlock: Default
      views:
        - name: Front
          type: base
          orientation: front
          scale: 0.5
          centerXmm: 90
          centerYmm: 120
"#;
    let package = DocumentPackage::decode(encoded).unwrap();
    let drawing = package.restore_drawing().unwrap().unwrap();
    assert_eq!(drawing.sheets()[0].dimensions_mm().unwrap(), [420., 297.]);
    assert_eq!(drawing.model_reference(), "/project/bracket.opd");
    assert_eq!(drawing.sheets()[0].views[0].scale, 0.5);
    assert_eq!(
        package
            .with_drawing(Some(&drawing))
            .unwrap()
            .encode()
            .unwrap(),
        package.encode().unwrap()
    );
}
#[test]
fn a_new_drawing_round_trips_without_claiming_a_projection() {
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 3\n").unwrap();
    let mut drawing = DrawingDocument::new();
    drawing.set_model_reference("/project/bracket.opd").unwrap();
    drawing
        .add_base_view_definition(
            0,
            DrawingBaseViewDefinition::new("Front", BASE_VIEW_FRONT, 1., [90., 120.]).unwrap(),
        )
        .unwrap();
    let saved = package
        .with_drawing(Some(&drawing))
        .unwrap()
        .encode()
        .unwrap();
    let reopened = DocumentPackage::decode(&saved).unwrap();
    assert_eq!(reopened.restore_drawing().unwrap(), Some(drawing));
    let yaml = String::from_utf8(saved).unwrap();
    assert!(yaml.contains("modelReference: /project/bracket.opd"));
    assert!(
        !yaml.contains("curves"),
        "definitions must not invent evaluated drawing curves"
    );
}

#[test]
fn saving_an_untouched_default_drawing_writes_its_real_sheet_for_go_readers() {
    let package = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 3\n").unwrap();
    let saved = package
        .with_drawing(Some(&DrawingDocument::new()))
        .unwrap()
        .encode()
        .unwrap();
    let yaml = String::from_utf8(saved).unwrap();
    assert!(
        yaml.contains("sheets:"),
        "default sheet must be stored, not supplied only by this reader: {yaml}"
    );
    assert!(yaml.contains("titleBlock: Default"));
}

#[test]
fn drawing_content_cannot_be_silently_saved_under_another_document_kind() {
    let part = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 1\n").unwrap();
    assert!(part.restore_drawing().unwrap().is_none());
    assert!(
        part.with_drawing(Some(&DrawingDocument::new()))
            .unwrap_err()
            .to_string()
            .contains("document type 1")
    );
    let drawing = DocumentPackage::decode(b"schemaVersion: 2\ndocumentType: 3\n").unwrap();
    assert!(drawing.with_drawing(None).is_err());
    assert_eq!(part.with_drawing(None).unwrap(), part);
}
#[test]
fn unsupported_drawing_recipe_content_is_refused_instead_of_dropped() {
    let bytes = b"schemaVersion: 2\ndocumentType: 3\nmodel:\n  sheets:\n    - name: Sheet:1\n      size: a3\n      annotations: [unimplemented]\n";
    let package = DocumentPackage::decode(bytes).unwrap();
    let cause = package.restore_drawing().unwrap_err().to_string();
    assert!(
        cause.contains("annotations") && cause.contains("expected"),
        "{cause}"
    );
    let bytes = b"schemaVersion: 2\ndocumentType: 3\nmodel:\n  sheets:\n    - name: Sheet:1\n      size: bad-size\n";
    let cause = DocumentPackage::decode(bytes)
        .unwrap()
        .restore_drawing()
        .unwrap_err()
        .to_string();
    assert!(
        cause.contains("bad-size") && cause.contains("expected"),
        "{cause}"
    );
}

#[test]
fn go_parts_list_definition_round_trips_without_cached_bom_rows() {
    use oblikovati_model::drawing::DrawingAnnotationDefinition;
    let encoded = br#"schemaVersion: 2
documentType: 3
model:
  modelReference: /project/bracket.opd
  sheets:
    - name: 'Sheet:1'
      size: a3
      orientation: landscape
      border: true
      annotations:
        - name: Parts
          kind: partsList
          xmm: 40
          ymm: 200
          characteristic: enum(?)
          materialRemoval: any
"#;
    let package = DocumentPackage::decode(encoded).unwrap();
    let drawing = package.restore_drawing().unwrap().unwrap();
    assert_eq!(drawing.sheets()[0].annotations[0].name(), "Parts");
    assert_eq!(
        DrawingAnnotationDefinition::parts_list("Parts", [40., 200.])
            .unwrap()
            .name(),
        "Parts"
    );
    let saved = package
        .with_drawing(Some(&drawing))
        .unwrap()
        .encode()
        .unwrap();
    assert_eq!(
        DocumentPackage::decode(&saved)
            .unwrap()
            .restore_drawing()
            .unwrap(),
        Some(drawing)
    );
    assert!(!String::from_utf8(saved).unwrap().contains("rows:"));
}
