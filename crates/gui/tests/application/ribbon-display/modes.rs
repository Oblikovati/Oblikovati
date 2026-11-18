// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn the_control_cycles_through_every_mode_and_returns_to_full() {
    let mut state = RibbonDisplayState::default();
    state.cycle_through_all();
    for expected in [
        RibbonDisplay::PanelButtons,
        RibbonDisplay::PanelTitles,
        RibbonDisplay::Tabs,
        RibbonDisplay::Full,
    ] {
        state.advance();
        assert_eq!(state.mode, expected);
    }
}
#[test]
fn the_default_policy_toggles_between_full_and_tabs() {
    let mut state = RibbonDisplayState::default();
    assert_eq!(state.mode, RibbonDisplay::Full);
    assert_eq!(state.minimized, Some(RibbonDisplay::Tabs));
    state.advance_from_tab();
    assert_eq!(state.mode, RibbonDisplay::Tabs);
    assert!(!state.take_tab_release());
    state.advance_from_tab();
    assert_eq!(state.mode, RibbonDisplay::Full);
}
#[test]
fn a_chosen_minimization_alternates_with_the_full_ribbon() {
    let mut state = RibbonDisplayState::default();
    state.choose(RibbonDisplay::Tabs);
    assert_eq!(state.mode, RibbonDisplay::Tabs);
    state.advance();
    assert_eq!(state.mode, RibbonDisplay::Full);
    state.advance();
    assert_eq!(state.mode, RibbonDisplay::Tabs);
    state.cycle_through_all();
    state.advance();
    assert_eq!(state.mode, RibbonDisplay::Full);
}
#[test]
fn each_reduction_gives_the_workspace_more_height() {
    let heights: Vec<f32> = CYCLE.iter().map(|mode| mode.band_height(0.)).collect();
    assert!(
        heights.windows(2).all(|pair| pair[0] > pair[1]),
        "{heights:?}"
    );
}
#[test]
fn changing_mode_closes_the_temporary_panel() {
    let mut state = RibbonDisplayState {
        temporary: Some(TemporaryPanel::WholeTab),
        ..Default::default()
    };
    state.advance();
    assert_eq!(state.temporary, None);
}
#[test]
fn switching_tabs_expands_only_an_open_compact_group() {
    let mut compact = RibbonDisplayState {
        mode: RibbonDisplay::PanelButtons,
        minimized: Some(RibbonDisplay::PanelButtons),
        temporary: Some(TemporaryPanel::Group(RibbonPanel::Create)),
        ..Default::default()
    };
    compact.switch_from_open_compact_panel();
    assert_eq!(compact.mode, RibbonDisplay::Full);
    assert_eq!(compact.temporary, None);
    assert_eq!(compact.minimized, Some(RibbonDisplay::PanelButtons));

    let mut tabs = RibbonDisplayState {
        mode: RibbonDisplay::Tabs,
        temporary: Some(TemporaryPanel::WholeTab),
        ..Default::default()
    };
    tabs.switch_from_open_compact_panel();
    assert_eq!(tabs.mode, RibbonDisplay::Tabs);
    assert_eq!(tabs.temporary, Some(TemporaryPanel::WholeTab));
}
#[test]
fn a_double_click_advances_and_its_release_opens_no_panel() {
    let mut state = RibbonDisplayState {
        mode: RibbonDisplay::PanelTitles,
        ..Default::default()
    };
    state.advance_from_tab();
    assert_eq!(state.mode, RibbonDisplay::Tabs);
    assert!(!state.take_tab_release(), "the double click's own release");
    assert!(state.take_tab_release(), "the next plain click's release");
}
#[test]
fn the_control_says_what_it_will_do_next() {
    let mut state = RibbonDisplayState::default();
    state.choose(RibbonDisplay::Tabs);
    assert_eq!(state.control_tooltip(), "Show Full Ribbon");
    state.advance();
    assert_eq!(state.control_tooltip(), "Minimize to Tabs");
}
#[test]
fn moving_the_ribbon_closes_a_temporary_panel_and_its_anchor() {
    let mut state = RibbonDisplayState {
        temporary: Some(TemporaryPanel::WholeTab),
        side_anchor: Some(SideAnchor {
            x: 1.,
            y: 2.,
            ends_at_x: false,
        }),
        ..Default::default()
    };
    state.place(RibbonPlacement::Left);
    assert_eq!(state.placement, RibbonPlacement::Left);
    assert!(state.temporary.is_none() && state.side_anchor.is_none());
}
#[test]
fn a_panel_toggles_for_its_own_tab_only() {
    let mut state = RibbonDisplayState::default();
    let navigate = (RibbonTab::View, RibbonPanel::Navigate);
    assert!(!state.hidden_panels.contains(&navigate));
    state.toggle_panel(navigate.0, navigate.1);
    assert!(state.hidden_panels.contains(&navigate));
    assert!(
        !state
            .hidden_panels
            .contains(&(RibbonTab::Manage, RibbonPanel::Navigate))
    );
    state.toggle_panel(navigate.0, navigate.1);
    assert!(!state.hidden_panels.contains(&navigate));
}
#[test]
fn parameters_start_hidden_on_the_model_tab_and_shown_on_manage() {
    let state = RibbonDisplayState::default();
    assert!(
        state
            .hidden_panels
            .contains(&(RibbonTab::Model, RibbonPanel::Parameters))
    );
    assert!(
        !state
            .hidden_panels
            .contains(&(RibbonTab::Manage, RibbonPanel::Parameters))
    );
}
