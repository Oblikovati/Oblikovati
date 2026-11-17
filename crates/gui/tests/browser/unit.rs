// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_model::feature::FeatureId;
fn sample_row() -> GuiBrowserSketch {
    GuiBrowserSketch {
        id: DocumentSketchId(7),
        name: "Section##7".into(),
        visible: true,
        consumed_by: None,
        line_style: Default::default(),
        dimensions_shown: true,
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
        extended: None,
        kind: "Extrusion",
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
fn block_browser_entries_require_a_valid_owner_definition_and_identity() {
    let mut browser = sample_state().browser.unwrap();
    browser.block_definitions.push("Rib".into());
    browser
        .block_instances
        .push(crate::GuiBrowserBlockInstance {
            sketch: DocumentSketchId(7),
            id: oblikovati_model::sketch::SketchEntityId(8),
            definition: "Rib".into(),
            ordinal: 1,
        });
    browser.validate().unwrap();
    browser.block_instances[0].sketch = DocumentSketchId(99);
    assert!(browser.validate().unwrap_err().contains("existing sketch"));
    browser.block_instances[0].sketch = DocumentSketchId(7);
    browser.block_instances[0].definition = "Missing".into();
    assert!(browser.validate().is_err());
    browser.block_instances[0].definition = "Rib".into();
    browser.block_instances[0].id = oblikovati_model::sketch::SketchEntityId(0);
    assert!(browser.validate().is_err());
}

#[test]
fn block_browser_selection_requires_a_current_instance_and_idle_sketch() {
    let mut state = sample_state();
    state.sketch_at_ready = true;
    state.authoring = crate::GuiAuthoringContext::Sketch;
    let browser = state.browser.as_mut().unwrap();
    browser.block_definitions.push("Rib".into());
    browser
        .block_instances
        .push(crate::GuiBrowserBlockInstance {
            sketch: DocumentSketchId(7),
            id: oblikovati_model::sketch::SketchEntityId(8),
            definition: "Rib".into(),
            ordinal: 1,
        });
    let action = GuiShellAction::SelectBrowserBlock {
        document: 1,
        sketch: DocumentSketchId(7),
        instance: oblikovati_model::sketch::SketchEntityId(8),
        toggle: false,
    };
    assert!(action.enabled(&state));
    state.sketch_at_ready = false;
    assert!(!action.enabled(&state));
    state.sketch_at_ready = true;
    state.browser.as_mut().unwrap().block_instances.clear();
    assert!(!action.enabled(&state));
}

#[test]
fn place_block_requires_the_definition_and_an_editable_sketch() {
    let mut state = sample_state();
    state.sketch_block_names.push("Rib".into());
    let action = GuiShellAction::PlaceSketchBlock("Rib".into());
    assert!(!action.enabled(&state));
    state.authoring = crate::GuiAuthoringContext::Sketch;
    assert!(action.enabled(&state));
    state.transaction_open = true;
    assert!(!action.enabled(&state));
    state.transaction_open = false;
    state.sketch_block_names.clear();
    assert!(!action.enabled(&state));
}

#[test]
fn model_state_labels_reject_nul_before_browser_adoption() {
    let mut browser = sample_state().browser.unwrap();
    let mut states = oblikovati_model::model_state::ModelStateMembers::default();
    assert!(states.insert(2, "bad\0state").is_err());
    states.insert(2, "Valid State").unwrap();
    browser.model_states = Some(states);
    browser.validate().unwrap();
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

#[test]
fn pattern_rows_require_unique_identities_current_sketches_and_safe_text() {
    use oblikovati_model::sketch::{
        SketchEntityId, SketchRectangularPattern, SketchRectangularPatternDefinition,
    };
    use std::collections::BTreeSet;
    let mut browser = sample_state().browser.unwrap();
    browser.block_definitions.push("Rib".into());
    browser
        .block_instances
        .push(crate::GuiBrowserBlockInstance {
            sketch: DocumentSketchId(7),
            id: SketchEntityId(8),
            definition: "Rib".into(),
            ordinal: 1,
        });
    let pattern = SketchRectangularPattern {
        id: SketchEntityId(9),
        definition: SketchRectangularPatternDefinition {
            seed: vec![SketchEntityId(3)],
            counts: [1, 1],
            directions: [[1., 0.], [0., 1.]],
            spacing: [1, 2],
            fitted: false,
            suppressed: BTreeSet::new(),
        },
        members: vec![],
    };
    browser
        .rectangular_patterns
        .push(crate::GuiBrowserRectangularPattern {
            sketch: DocumentSketchId(7),
            ordinal: 1,
            pattern,
            spacing: Ok(["1 cm".into(), "2 cm".into()]),
        });
    browser.validate().unwrap();
    browser.rectangular_patterns[0].pattern.id = SketchEntityId(8);
    assert!(browser.validate().unwrap_err().contains("unique identity"));
    browser.rectangular_patterns[0].pattern.id = SketchEntityId(9);
    browser.rectangular_patterns[0].sketch = DocumentSketchId(99);
    assert!(browser.validate().unwrap_err().contains("existing sketch"));
    browser.rectangular_patterns[0].sketch = DocumentSketchId(7);
    browser.rectangular_patterns[0].spacing = Ok(["bad\0text".into(), "2 cm".into()]);
    assert!(browser.validate().unwrap_err().contains("without NUL"));
}
