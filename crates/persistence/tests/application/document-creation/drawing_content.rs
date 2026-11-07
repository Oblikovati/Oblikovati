// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_api::types::BASE_VIEW_FRONT;
use oblikovati_model::drawing::{DrawingBaseViewDefinition, DrawingDocument};
use oblikovati_persistence::package::DocumentPackage;
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
