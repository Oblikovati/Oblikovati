// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{GuiAuthoringContext, GuiBrowserSketch, GuiModelBrowser};
fn part() -> GuiCommandState {
    GuiCommandState {
        document: Some(7),
        authoring: GuiAuthoringContext::Part,
        extrude_available: true,
        browser: Some(GuiModelBrowser {
            sketches: vec![GuiBrowserSketch {
                id: DocumentSketchId(4),
                inspection: Ok(Default::default()),
                name: "Sketch".into(),
                visible: true,
                consumed_by: None,
                line_style: Default::default(),
                dimensions_shown: false,
            }],
            ..Default::default()
        }),
        ..Default::default()
    }
}
#[test]
fn stale_extrusion_closes_before_docking_without_consuming_a_new_request() {
    let mut dialog = ExtrudeDialog::default();
    dialog.set_document(Some(7));
    let mut state = part();
    state.document = Some(8);
    dialog.sync_before_docking(&state);
    assert!(dialog.document.is_none());
    dialog.requested = true;
    dialog.sync_before_docking(&state);
    assert!(dialog.requested);
}
#[test]
fn draft_is_bound_to_the_open_document() {
    let mut state = part();
    let mut dialog = ExtrudeDialog {
        requested: true,
        ..Default::default()
    };
    dialog.synchronize(&state);
    assert_eq!(dialog.document, Some(7));
    assert_eq!(dialog.sketch, Some(DocumentSketchId(4)));
    dialog.distance = "size".into();
    state.document = Some(8);
    dialog.synchronize(&state);
    assert_eq!(dialog.document, None);
    assert_eq!(start_action(&state), GuiShellAction::OpenExtrude);
    state.browser.as_mut().unwrap().sketches.clear();
    assert_eq!(
        start_action(&state),
        GuiShellAction::ChooseSketchPlaneForExtrude
    );
}
#[test]
fn extrusion_requires_capability_current_sketch_and_nonempty_distance() {
    let mut state = part();
    let action = GuiShellAction::Extrude {
        document: 7,
        sketch: DocumentSketchId(4),
        profile: 0,
        distance: "size".into(),
        second_distance: None,
        direction: GuiExtrudeDirection::Positive,
    };
    assert!(action.dispatchable(&state));
    state.extrude_available = false;
    assert!(!action.dispatchable(&state));
    state.extrude_available = true;
    state.transaction_open = true;
    assert!(!action.dispatchable(&state));
    state.transaction_open = false;
    state.document = Some(8);
    assert!(!action.dispatchable(&state));
}
#[test]
fn edit_seed_keeps_its_source_and_new_extrusion_clears_edit_mode() {
    let mut state = part();
    let feature = oblikovati_model::feature::FeatureId(9);
    state.browser.as_mut().unwrap().extrusion_edits.insert(
        feature,
        crate::GuiExtrudeEditSeed {
            sketch: DocumentSketchId(4),
            distance: "size".into(),
            second_distance: None,
            direction: GuiExtrudeDirection::Symmetric,
        },
    );
    let mut dialog = ExtrudeDialog::default();
    dialog.open_edit(&state, feature);
    assert_eq!(dialog.edit_feature, Some(feature));
    assert_eq!(dialog.distance, "size");
    assert_eq!(dialog.direction, GuiExtrudeDirection::Symmetric);
    dialog.requested = true;
    dialog.synchronize(&state);
    assert_eq!(dialog.edit_feature, None);
    assert!(dialog.distance.is_empty());
    assert_eq!(dialog.direction, GuiExtrudeDirection::Positive);
}
