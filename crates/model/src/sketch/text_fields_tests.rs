// SPDX-License-Identifier: GPL-2.0-only
//! Check source and display positions for live Sketch Text fields.
use super::*;
use oblikovati_api::types::{
    SKETCH_TEXT_FIELD_FILE_NAME, SKETCH_TEXT_FIELD_MODEL_PARAMETER, SKETCH_TEXT_FIELD_PLACEHOLDER,
};

#[test]
fn display_expands_unicode_fields_and_moves_formatting_and_fraction_ranges() {
    let authored = linked_style();
    let resolved = resolve_sketch_text_display(&authored, |field| match field.at {
        1 => Ok("10.000".into()),
        7 => Ok("部品.opd".into()),
        at => panic!("unexpected test field at {at}"),
    })
    .unwrap();
    assert_eq!(resolved.content, "界10.000 1/2\n部品.opd");
    assert_eq!(
        resolved.stacked_fractions.as_deref(),
        Some(&[SketchTextFraction { start: 8, end: 11 }][..])
    );
    let [parameter, filename] = resolved.formatting_runs.as_deref().unwrap() else {
        panic!("expected the two formatting runs");
    };
    assert_eq!(
        (parameter.start, parameter.end, parameter.bold),
        (1, 7, true)
    );
    assert_eq!(
        (filename.start, filename.end, filename.italic),
        (12, 18, true)
    );
    assert!(resolved.linked_fields.is_none());
    assert_eq!(authored, linked_style());
}

#[test]
fn display_expands_the_same_source_again_when_the_value_changes() {
    let authored = linked_style();
    let display = |value: &str| {
        resolve_sketch_text_display(&authored, |field| {
            Ok(if field.at == 1 {
                value.into()
            } else {
                "CurrentPart.opd".into()
            })
        })
        .unwrap()
        .into_owned()
    };
    let old = display("10.000");
    let edited = display("123.000");
    assert_eq!(old.content, "界10.000 1/2\nCurrentPart.opd");
    assert_eq!(edited.content, "界123.000 1/2\nCurrentPart.opd");
    assert_eq!(edited.stacked_fractions.unwrap()[0].start, 9);
    assert_eq!(display("10.000"), old);
}

#[test]
fn literal_text_stays_borrowed_and_never_calls_the_field_resolver() {
    let style = SketchTextStyle {
        content: "literal".into(),
        ..Default::default()
    };
    let display = resolve_sketch_text_display(&style, |_| panic!("literal content has no fields"));
    assert!(matches!(display.unwrap(), Cow::Borrowed(value) if std::ptr::eq(value, &style)));
}

#[test]
fn editor_projection_uses_utf8_bytes_for_literal_and_field_text() {
    let authored = linked_style();
    let editor = project_sketch_text_fields(&authored, |field| {
        Ok(if field.at == 1 {
            "10.000"
        } else {
            "部品.opd"
        }
        .into())
    })
    .unwrap();
    let [first, second] = editor.fields.as_slice() else {
        panic!("expected two editable field ranges");
    };
    assert_eq!(
        &authored.content[first.authored.clone()],
        SKETCH_TEXT_FIELD_PLACEHOLDER
    );
    assert_eq!(
        &authored.content[second.authored.clone()],
        SKETCH_TEXT_FIELD_PLACEHOLDER
    );
    assert_eq!(first.display, 3..9);
    assert_eq!(&editor.content[second.display.clone()], "部品.opd");
}

#[test]
fn invalid_field_content_or_location_cannot_publish_a_display_style() {
    let mut style = linked_style();
    let reason = resolve_sketch_text_display(&style, |_| Ok("\0".into())).unwrap_err();
    assert!(reason.to_string().contains("without NUL"));
    let reason = resolve_sketch_text_display(&style, |_| {
        Err(SketchError::Definition("parameter d0 is missing".into()))
    })
    .unwrap_err();
    assert!(reason.to_string().contains("parameter d0 is missing"));
    style.linked_fields.as_mut().unwrap()[0].at = 0;
    let reason = resolve_sketch_text_display(&style, |_| panic!("misplaced field must fail first"))
        .unwrap_err();
    assert!(reason.to_string().contains("U+FFFC"));
}

fn linked_style() -> SketchTextStyle {
    SketchTextStyle {
        content: format!("界{SKETCH_TEXT_FIELD_PLACEHOLDER} 1/2\n{SKETCH_TEXT_FIELD_PLACEHOLDER}"),
        height: 0.305,
        linked_fields: Some(vec![
            SketchTextField {
                at: 1,
                kind: SKETCH_TEXT_FIELD_MODEL_PARAMETER,
                parameter: "d0".into(),
                ..Default::default()
            },
            SketchTextField {
                at: 7,
                kind: SKETCH_TEXT_FIELD_FILE_NAME,
                ..Default::default()
            },
        ]),
        formatting_runs: Some(vec![
            SketchTextRun {
                start: 1,
                end: 2,
                bold: true,
                ..Default::default()
            },
            SketchTextRun {
                start: 7,
                end: 8,
                italic: true,
                ..Default::default()
            },
        ]),
        stacked_fractions: Some(vec![SketchTextFraction { start: 3, end: 6 }]),
        ..Default::default()
    }
}
