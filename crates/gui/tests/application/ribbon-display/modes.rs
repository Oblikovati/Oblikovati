// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn the_control_cycles_through_every_mode_and_returns_to_full() {
    let mut state = RibbonDisplayState::default();
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
