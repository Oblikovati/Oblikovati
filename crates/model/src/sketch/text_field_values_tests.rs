// SPDX-License-Identifier: GPL-2.0-only
//! Exercise typed field values against the real document parameter collection.
use super::*;
use crate::units::UnitsOfMeasure;
use oblikovati_api::types::{MODEL_PARAM, SKETCH_TEXT_FIELD_PLACEHOLDER};

#[test]
fn field_resolution_tracks_value_and_identity_through_a_rename() {
    let mut parameters = model_dimension();
    let style = linked_document_style(None);
    let bindings = bind_sketch_text_parameters(&style, &parameters).unwrap();
    assert_eq!(
        bindings,
        [SketchTextParameterBinding {
            at: 0,
            parameter_id: 42
        }]
    );
    assert_eq!(
        document_text(&style, &bindings, &parameters, "Part6.opd"),
        "10.000\nPart6.opd"
    );
    parameters.set_expression(42, "12 mm").unwrap();
    parameters.rename(42, "width").unwrap();
    parameters
        .insert_expression(43, "d0", MODEL_PARAM, "99 mm")
        .unwrap();
    assert_eq!(
        document_text(&style, &bindings, &parameters, "Rename.opd"),
        "12.000\nRename.opd"
    );
    parameters.remove_for_owner(42).unwrap();
    let error =
        resolve_document_sketch_text(&style, &bindings, &parameters, "Rename.opd").unwrap_err();
    assert!(error.to_string().contains("42"));
}

#[test]
fn display_uses_explicit_precision_in_document_units() {
    let parameters = model_dimension();
    for (precision, expected) in [(0, "10"), (2, "10.00"), (8, "10.00000000")] {
        let style = linked_document_style(Some(precision));
        let bindings = bind_sketch_text_parameters(&style, &parameters).unwrap();
        let result = document_text(&style, &bindings, &parameters, "Part6.opd");
        assert_eq!(result, format!("{expected}\nPart6.opd"));
    }
}

#[test]
fn document_editor_projects_property_labels_and_live_numeric_values() {
    let mut parameters = model_dimension();
    let authored = linked_document_style(Some(2));
    let editor = resolve_document_sketch_text_editor(&authored, None, &parameters).unwrap();
    assert_eq!(editor.content, "10.00\nFILENAME");
    assert_eq!(editor.fields[0].authored, 0..3);
    assert_eq!(editor.fields[0].display, 0..5);
    assert_eq!(editor.fields[1].authored, 4..7);
    assert_eq!(editor.fields[1].display, 6..14);
    parameters.set_expression(42, "123 mm").unwrap();
    let updated = resolve_document_sketch_text_editor(&authored, None, &parameters).unwrap();
    assert_eq!(updated.content, "123.00\nFILENAME");
    assert_eq!(updated.fields[1].display, 7..15);
    assert_eq!(updated.fields[1].authored, 4..7);
}

#[test]
fn missing_field_identity_is_a_diagnostic() {
    let parameters = model_dimension();
    let mut style = linked_document_style(None);
    let error = resolve_document_sketch_text(&style, &[], &parameters, "Part6.opd").unwrap_err();
    assert!(
        error
            .to_string()
            .contains("expected an authored parameter identity")
    );
    style.linked_fields.as_mut().unwrap()[0].parameter = "absent".into();
    let error = bind_sketch_text_parameters(&style, &parameters).unwrap_err();
    assert!(
        error
            .to_string()
            .contains("expected a current document parameter")
    );
}

fn document_text(
    style: &SketchTextStyle,
    bindings: &[SketchTextParameterBinding],
    parameters: &ParameterCollection<u64>,
    file_name: &str,
) -> String {
    resolve_document_sketch_text(style, bindings, parameters, file_name)
        .unwrap()
        .into_owned()
        .content
}

fn model_dimension() -> ParameterCollection<u64> {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(42, "d0", MODEL_PARAM, "10 mm")
        .unwrap();
    parameters
}

fn linked_document_style(precision: Option<u8>) -> SketchTextStyle {
    SketchTextStyle {
        content: format!("{SKETCH_TEXT_FIELD_PLACEHOLDER}\n{SKETCH_TEXT_FIELD_PLACEHOLDER}"),
        height: 0.305,
        linked_fields: Some(vec![
            SketchTextField {
                at: 0,
                kind: SKETCH_TEXT_FIELD_MODEL_PARAMETER,
                parameter: "d0".into(),
                precision: precision.map(Box::new),
            },
            SketchTextField {
                at: 2,
                kind: SKETCH_TEXT_FIELD_FILE_NAME,
                ..Default::default()
            },
        ]),
        ..Default::default()
    }
}
