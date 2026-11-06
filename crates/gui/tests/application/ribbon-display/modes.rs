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
