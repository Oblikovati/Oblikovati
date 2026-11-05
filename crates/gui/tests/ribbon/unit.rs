// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn part_state() -> GuiCommandState {
    GuiCommandState {
        document: Some(7),
        document_type: Some(DOCUMENT_PART),
        authoring: GuiAuthoringContext::Part,
        ..Default::default()
    }
}
const DOCUMENT_RIBBON_CASES: [(oblikovati_api::types::DocumentType, RibbonKey); 4] = [
    (DOCUMENT_PART, PART_RIBBON),
    (DOCUMENT_ASSEMBLY, ASSEMBLY_RIBBON),
    (DOCUMENT_DRAWING, DRAWING_RIBBON),
    (DOCUMENT_PRESENTATION, PRESENTATION_RIBBON),
];
#[test]
fn each_document_selects_the_canonical_contract_ribbon() {
    let mut state = part_state();
    for (kind, key) in DOCUMENT_RIBBON_CASES {
        state.document_type = Some(kind);
        assert_eq!(document_ribbon_key(&state), key);
    }
}
#[test]
fn no_document_and_unresolved_document_kinds_have_distinct_ribbons() {
    let empty = GuiCommandState::default();
    assert_eq!(document_ribbon_key(&empty), ZERO_DOC_RIBBON);
    let mut state = part_state();
    state.document_type = None;
    assert_eq!(document_ribbon_key(&state), UNKNOWN_DOCUMENT_RIBBON);
}
#[test]
fn manual_selection_is_preserved_between_environment_changes() {
    let mut ribbon = DocumentRibbon::default();
    let mut state = part_state();
    assert_eq!(ribbon.prepare(&state).select, Some(RibbonTab::CreateModify));
    state.authoring = GuiAuthoringContext::Sketch;
    assert_eq!(ribbon.prepare(&state).select, Some(RibbonTab::Sketch));
    ribbon.active = RibbonTab::Manage;
    assert_eq!(ribbon.prepare(&state).select, None);
    assert_eq!(ribbon.active, RibbonTab::Manage);
    state.authoring = GuiAuthoringContext::Part;
    assert_eq!(ribbon.prepare(&state).select, Some(RibbonTab::CreateModify));
}
#[test]
fn changing_document_identity_keeps_the_tab_but_changing_kind_selects_its_base() {
    let mut ribbon = DocumentRibbon::default();
    let mut state = part_state();
    ribbon.prepare(&state);
    ribbon.active = RibbonTab::View;
    state.document = Some(8);
    assert_eq!(ribbon.prepare(&state).select, None);
    assert_eq!(ribbon.active, RibbonTab::View);
    state.document_type = Some(DOCUMENT_DRAWING);
    assert_eq!(ribbon.prepare(&state).select, Some(RibbonTab::View));
}
#[test]
fn document_tabs_keep_sketch_context_off_other_document_types() {
    for key in [DRAWING_RIBBON, PRESENTATION_RIBBON, UNKNOWN_DOCUMENT_RIBBON] {
        assert_eq!(document_tabs(&key, true), [RibbonTab::View]);
    }
    let zero_tabs = document_tabs(&ZERO_DOC_RIBBON, true);
    assert_eq!(zero_tabs, [RibbonTab::GetStarted]);
    for key in [PART_RIBBON, ASSEMBLY_RIBBON] {
        assert!(document_tabs(&key, true).contains(&RibbonTab::Sketch));
        assert!(!document_tabs(&key, false).contains(&RibbonTab::Sketch));
    }
}
#[test]
fn tab_names_match_the_go_command_registry() {
    for (tab, label) in [
        (RibbonTab::GetStarted, "Get Started"),
        (RibbonTab::CreateModify, "Create & Modify"),
        (RibbonTab::Sketch, "Sketch"),
        (RibbonTab::Manage, "Manage"),
        (RibbonTab::View, "View"),
    ] {
        assert_eq!(tab.label(), label);
    }
}
