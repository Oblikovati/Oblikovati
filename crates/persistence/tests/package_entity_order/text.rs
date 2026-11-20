// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::types::{SketchTextRun, SketchTextStyle, TEXT_ALIGN_CENTER, TEXT_ALIGN_UPPER};
use oblikovati_model::sketch::{SKETCH_RECIPE_VERSION, SketchText2D};

const TEXT: SketchEntityId = SketchEntityId(10);
const ANCHOR: SketchEntityId = SketchEntityId(11);
const PLACED: DocumentSketchId = DocumentSketchId(7);

#[test]
fn native_yaml_round_trips_authored_text_style_and_its_hidden_anchor() {
    check_text_round_trip(None);
}

#[test]
fn native_yaml_round_trips_formatting_in_unicode_scalar_ranges() {
    check_text_round_trip(Some(vec![SketchTextRun {
        start: 6,
        end: 7,
        bold: true,
        width_percent: Some(Box::new(50.)),
        ..Default::default()
    }]));
}

fn check_text_round_trip(formatting: Option<Vec<SketchTextRun>>) {
    let mut owner = sketches();
    let mut geometry = geometry();
    let mut text = add_text(&mut geometry);
    text.style.formatting_runs = formatting;
    geometry.set_text_style(TEXT, text.style.clone()).unwrap();
    geometry.add_point(SketchEntityId(12), [1., 2.]).unwrap();
    owner.replace_geometry(PLACED, geometry).unwrap();
    let bytes = DocumentPackage::empty(DOCUMENT_PART, "Text")
        .with_sketches(&owner, PLACED.0, 12)
        .unwrap()
        .encode()
        .unwrap();
    let decoded = DocumentPackage::decode(&bytes).unwrap();
    let mut restored = decoded.restore_sketches().unwrap();
    assert!(restored.sketches.same_recipe(&owner));
    assert_saved_text(restored.sketches.get(PLACED).unwrap().geometry(), &text);
    assert_eq!(restored.entity_ids.allocate().unwrap(), 13);
    assert_eq!(decoded.encode().unwrap(), bytes);
}

fn add_text(geometry: &mut Sketch2D) -> SketchText2D {
    geometry.add_defining_point(ANCHOR, [2., 3.]).unwrap();
    let text = SketchText2D {
        id: TEXT,
        anchor: ANCHOR,
        style: SketchTextStyle {
            content: "Plate Ø12\nTop".into(),
            family: "Liberation Sans".into(),
            font_size: 0.45,
            height: 0.3,
            rotation: 0.125,
            h_align: TEXT_ALIGN_CENTER,
            v_align: TEXT_ALIGN_UPPER,
            formatting_runs: None,
        },
    };
    geometry.add_text(text.clone()).unwrap();
    text
}

fn assert_saved_text(geometry: &Sketch2D, text: &SketchText2D) {
    assert_eq!(geometry.recipe().version, SKETCH_RECIPE_VERSION);
    assert_eq!(geometry.text(TEXT).unwrap(), text);
    assert_eq!(geometry.point(ANCHOR).unwrap().position, [2., 3.]);
    assert!(!geometry.entity_ids().contains(&ANCHOR));
    assert_eq!(geometry.entity_ids()[4..], [TEXT, SketchEntityId(12)]);
}

#[test]
fn text_style_edit_preserves_identity_and_rejects_invalid_changes() {
    let mut geometry = geometry();
    let text = add_text(&mut geometry);
    let mut style = text.style.clone();
    style.height = 0.;
    assert!(geometry.set_text_style(TEXT, style.clone()).is_err());
    assert_eq!(geometry.text(TEXT).unwrap(), &text);
    style.height = 0.7;
    style.content = "REV B".into();
    geometry.set_text_style(TEXT, style.clone()).unwrap();
    assert_eq!(geometry.text(TEXT).unwrap().style, style);
    assert_eq!(geometry.text(TEXT).unwrap().anchor, ANCHOR);
}

#[test]
fn deleting_text_retires_its_unused_anchor_and_restores_old_recipe_version() {
    let mut geometry = geometry();
    let original = geometry.clone();
    add_text(&mut geometry);
    let deletion = geometry.delete_entities(&[TEXT]).unwrap();
    assert_eq!(deletion.entities, [TEXT, ANCHOR]);
    assert!(geometry.same_recipe(&original));
    assert_eq!(geometry.recipe().version, original.recipe().version);
}

#[test]
fn text_recipe_rejects_a_duplicate_id_or_missing_anchor() {
    let mut geometry = geometry();
    add_text(&mut geometry);
    let mut recipe = geometry.recipe();
    let text = &mut recipe.texts.as_mut().unwrap()[0];
    text.id = SketchEntityId(5);
    assert!(Sketch2D::from_recipe(&recipe).is_err());
    let text = &mut recipe.texts.as_mut().unwrap()[0];
    text.id = TEXT;
    text.anchor = SketchEntityId(99);
    assert!(Sketch2D::from_recipe(&recipe).is_err());
}

#[test]
fn capturing_text_keeps_local_ids_and_copies_only_the_selected_text_and_anchor() {
    let mut geometry = geometry();
    let text = add_text(&mut geometry);
    let captured = geometry.capture_selection(&[TEXT]).unwrap();
    let copied = &captured.texts()[0];
    assert_eq!(copied.id, TEXT);
    assert_eq!(copied.anchor, ANCHOR);
    assert_eq!(captured.entity_ids(), [copied.id]);
    assert_eq!(captured.point(copied.anchor).unwrap().position, [2., 3.]);
    assert_eq!(copied.style, text.style);
}
