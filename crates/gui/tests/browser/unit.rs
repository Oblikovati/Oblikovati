// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn sample_row() -> GuiBrowserSketch {
    GuiBrowserSketch {
        id: DocumentSketchId(7),
        name: "Section##7".into(),
        consumed_by: None,
        inspection: Ok(oblikovati_api::wire::SketchInfo {
            dof: 0,
            healthy: true,
            plane: "XY".into(),
            entity_count: 5,
            ..Default::default()
        }),
    }
}
fn sample_state() -> GuiCommandState {
    GuiCommandState {
        document: Some(1),
        authoring: crate::GuiAuthoringContext::Part,
        browser: Some(GuiModelBrowser {
            name: "Part".into(),
            sketches: vec![sample_row()],
            ..Default::default()
        }),
        ..Default::default()
    }
}
#[test]
fn sketch_activation_requires_current_document_identity_and_no_open_edit_group() {
    let mut state = sample_state();
    let action = GuiShellAction::EditSketch {
        document: 1,
        sketch: DocumentSketchId(7),
    };
    assert!(action.dispatchable(&state));
    state.transaction_open = true;
    assert!(action.enabled(&state) && !action.dispatchable(&state));
    state.transaction_open = false;
    state.document = Some(2);
    assert!(!action.enabled(&state));
    state.document = Some(1);
    state.authoring = crate::GuiAuthoringContext::Sketch;
    assert!(!action.enabled(&state));
    state.authoring = crate::GuiAuthoringContext::Part;
    assert!(
        !GuiShellAction::EditSketch {
            document: 1,
            sketch: DocumentSketchId(8)
        }
        .enabled(&state)
    );
}
#[test]
fn captions_and_tooltips_keep_literal_names_and_show_local_failures() {
    let mut row = sample_row();
    assert_eq!(sketch_caption(&row), "Section##7  [DOF 0]");
    assert!(sketch_tooltip(&row).contains("XY plane, 5 entities"));
    row.inspection.as_mut().unwrap().healthy = false;
    assert!(sketch_caption(&row).contains("sick"));
    row.inspection = Err("missing plane 19".into());
    assert_eq!(sketch_caption(&row), "Section##7  [unavailable]");
    assert!(sketch_tooltip(&row).contains("missing plane 19"));
}
#[test]
fn browser_validation_refuses_ambiguous_or_dropped_identity_and_unsafe_text() {
    let mut state = sample_state();
    state.validate().unwrap();
    state.document = None;
    assert!(state.validate().unwrap_err().contains("active document"));
    let browser = state.browser.as_mut().unwrap();
    browser.sketches.push(sample_row());
    assert!(browser.validate().unwrap_err().contains("unique nonzero"));
    browser.sketches.pop();
    browser.sketches[0].consumed_by = Some(FeatureId(42));
    assert!(
        browser
            .validate()
            .unwrap_err()
            .contains("current feature program")
    );
    browser.features.push(GuiBrowserFeature {
        id: FeatureId(42),
        name: "Extrude".into(),
        suppressed: false,
    });
    browser.validate().unwrap();
    browser.sketches[0].inspection = Err("bad\0cause".into());
    assert!(browser.validate().unwrap_err().contains("NUL"));
}
#[test]
fn labels_are_checked_before_imgui_receives_a_nul_terminated_name() {
    let mut browser = sample_state().browser.unwrap();
    browser.name = "bad\0document".into();
    assert!(browser.validate().unwrap_err().contains("NUL"));
    browser.name = "Part".into();
    browser.sketches[0].name = "bad\0sketch".into();
    assert!(browser.validate().unwrap_err().contains("NUL"));
    browser.sketches[0].name = "Valid".into();
    browser.sketches[0].id = DocumentSketchId(0);
    assert!(browser.validate().unwrap_err().contains("unique nonzero"));
}
#[test]
fn canonical_plane_text_is_checked_before_it_reaches_the_tooltip() {
    let mut browser = sample_state().browser.unwrap();
    browser.sketches[0].inspection.as_mut().unwrap().plane = "XY\0plane".into();
    assert!(browser.validate().unwrap_err().contains("NUL"));
}

#[test]
fn selection_has_its_own_enablement_and_requires_a_current_row() {
    let mut state = sample_state();
    let select = GuiShellAction::SelectBrowserSketch {
        document: 1,
        sketch: DocumentSketchId(7),
        toggle: false,
    };
    assert!(select.dispatchable(&state));
    state.authoring = crate::GuiAuthoringContext::Sketch;
    assert!(select.dispatchable(&state));
    state.transaction_open = true;
    assert!(!select.dispatchable(&state));
    state.transaction_open = false;
    state.document = Some(2);
    assert!(!select.dispatchable(&state));
    state.document = Some(1);
    state
        .browser
        .as_mut()
        .unwrap()
        .selected_sketches
        .push(DocumentSketchId(99));
    assert!(state.validate().unwrap_err().contains("current sketch row"));
}
