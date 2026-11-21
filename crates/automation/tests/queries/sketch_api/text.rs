// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::sketch::SketchText2D;
use std::boxed::Box;

#[test]
fn typed_text_preserves_spacing_and_identity_through_history_and_reopen() {
    let (mut files, document, mut source) = entities::configured_scaled(0.125);
    let entity = add_multiline(&mut files, &mut source);
    let initial = read_text(&files, entity);
    assert_initial_style(&initial);
    let anchor = text_anchor(&files, document, entity);
    assert_eq!(anchor.1, [16., 24.]);
    let edited = edit_content(&mut files, &mut source, entity);
    assert_eq!(edited.style.content, "c\nd");
    assert!(edited.style.formatting_runs.is_none());
    assert_eq!(edited.style.line_spacing, initial.style.line_spacing);
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(read_text(&files, entity), initial);
    files.commands(document).unwrap().redo().unwrap();
    assert_eq!(read_text(&files, entity), edited);
    let document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(read_text(&files, entity), edited);
    assert_eq!(text_anchor(&files, document, entity), anchor);
}

#[test]
fn typed_text_rejects_invalid_edits_and_resets_spacing_explicitly() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let entity = add_multiline(&mut files, &mut source);
    let current = read_text(&files, entity);
    let revision = owner_revision(&files, document);
    check_rejected_range(&mut files, &mut source, entity);
    assert_eq!(read_text(&files, entity), current);
    assert_eq!(owner_revision(&files, document), revision);
    let client = Client::new(FileHost::new(&mut files, &mut source, "test"));
    let reset = client
        .sketch()
        .edit_text(&EditTextArgs {
            entity_id: entity,
            line_spacing: spacing(TEXT_LINE_SPACING_SINGLE, 0.),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(
        reset.style.line_spacing,
        spacing(TEXT_LINE_SPACING_SINGLE, 0.)
    );
    assert_eq!(reset.style.formatting_runs, current.style.formatting_runs);
}

fn add_multiline(files: &mut DiscoveryWorkspace, source: &mut QueryGuids) -> u64 {
    let client = Client::new(FileHost::new(files, source, "test"));
    client.sketch().create(&Default::default()).unwrap();
    client
        .sketch()
        .add_text(&AddTextArgs {
            anchor: Some(vec![2., 3.]),
            text: "a界\nb".into(),
            height: "width / 2".into(),
            rotation: "90 deg".into(),
            justify: "center".into(),
            v_justify: "middle".into(),
            font: "DejaVu Sans".into(),
            font_size: "2 mm".into(),
            formatting_runs: Some(vec![SketchTextRun {
                start: 1,
                end: 2,
                bold: true,
                ..Default::default()
            }]),
            line_spacing: spacing(TEXT_LINE_SPACING_EXACTLY, 0.4),
            ..Default::default()
        })
        .unwrap()
        .entity_id
}

fn read_text(files: &DiscoveryWorkspace, entity: u64) -> SketchTextResult {
    Client::new(FileQueryHost::new(files))
        .sketch()
        .get_text(&GetTextArgs {
            sketch_index: 0,
            entity_id: entity,
        })
        .unwrap()
}

fn assert_initial_style(result: &SketchTextResult) {
    assert_eq!(result.style.height, 0.5);
    assert_eq!(result.style.font_size, 0.2);
    assert_eq!(result.style.rotation, std::f64::consts::FRAC_PI_2);
    assert_eq!(result.style.h_align, TEXT_ALIGN_CENTER);
    assert_eq!(result.style.v_align, TEXT_ALIGN_MIDDLE);
    assert_eq!(
        result.style.line_spacing,
        spacing(TEXT_LINE_SPACING_EXACTLY, 0.4)
    );
}

fn text_anchor(
    files: &DiscoveryWorkspace,
    document: u64,
    entity: u64,
) -> (SketchEntityId, [f64; 2]) {
    let geometry = owned_geometry(files, document).sketches()[0].geometry();
    let SketchText2D { anchor, .. } = geometry.text(SketchEntityId(entity)).unwrap();
    (*anchor, geometry.point(*anchor).unwrap().position)
}

fn edit_content(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    entity: u64,
) -> SketchTextResult {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .edit_text(&EditTextArgs {
            entity_id: entity,
            text: Some(Box::new("c\nd".into())),
            ..Default::default()
        })
        .unwrap()
}

fn check_rejected_range(files: &mut DiscoveryWorkspace, source: &mut QueryGuids, entity: u64) {
    let result = Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .edit_text(&EditTextArgs {
            entity_id: entity,
            height: "9 mm".into(),
            formatting_runs: Some(Box::new(Some(vec![SketchTextRun {
                start: 0,
                end: 5,
                italic: true,
                ..Default::default()
            }]))),
            ..Default::default()
        });
    expect_api_decline(result, "scalar");
}

fn spacing(mode: TextLineSpacingMode, value: f64) -> Option<Box<SketchTextLineSpacing>> {
    Some(Box::new(SketchTextLineSpacing { mode, value }))
}

#[test]
fn typed_fraction_is_preserved_cleared_and_restored_with_the_original_text() {
    use oblikovati_api::types::SketchTextFraction;
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let entity = add_multiline(&mut files, &mut source);
    let range = SketchTextFraction { start: 1, end: 4 };
    let stacked = edit_fraction(
        &mut files,
        &mut source,
        EditTextArgs {
            entity_id: entity,
            text: Some(Box::new("⌀1/2".into())),
            stacked_fractions: Some(Box::new(Some(vec![range]))),
            ..Default::default()
        },
    );
    let taller = edit_fraction(
        &mut files,
        &mut source,
        EditTextArgs {
            entity_id: entity,
            height: "6 mm".into(),
            ..Default::default()
        },
    );
    assert_eq!(
        taller.style.stacked_fractions,
        stacked.style.stacked_fractions
    );
    let cleared = edit_fraction(
        &mut files,
        &mut source,
        EditTextArgs {
            entity_id: entity,
            stacked_fractions: Some(Box::new(Some(Vec::new()))),
            ..Default::default()
        },
    );
    assert_eq!(cleared.style.content, "⌀1/2");
    assert!(cleared.style.stacked_fractions.is_none());
    files.commands(document).unwrap().undo().unwrap();
    assert_eq!(read_text(&files, entity), taller);
    let _document = reopen_with_observer(&mut files, document, &mut source, Default::default());
    assert_eq!(read_text(&files, entity), taller);
}

fn edit_fraction(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    request: EditTextArgs,
) -> SketchTextResult {
    Client::new(FileHost::new(files, source, "test"))
        .sketch()
        .edit_text(&request)
        .unwrap()
}
