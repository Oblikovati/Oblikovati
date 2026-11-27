// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn only_running_navigation_owns_viewport_input() {
    let mut navigation = NavigationBar::default();
    let mut actions = Vec::new();
    assert!(!navigation.owns_viewport_input());
    for mode in [
        GuiNavigationMode::Pan,
        GuiNavigationMode::Zoom,
        GuiNavigationMode::Orbit,
        GuiNavigationMode::ConstrainedOrbit,
        GuiNavigationMode::ZoomWindow,
        GuiNavigationMode::ZoomSelected,
    ] {
        navigation.start(mode, &mut actions);
        assert!(navigation.owns_viewport_input());
        navigation.end(&mut actions);
        assert!(!navigation.owns_viewport_input());
    }
}
#[test]
fn a_drag_becomes_the_running_commands_motion() {
    assert_eq!(
        drag_command(GuiNavigationMode::Pan, [3., -2.]),
        GuiViewCommand::Pan([3, -2])
    );
    assert_eq!(
        drag_command(GuiNavigationMode::Zoom, [5., 0.]),
        GuiViewCommand::Zoom(1.)
    );
    let GuiViewCommand::Orbit { yaw, pitch } = drag_command(GuiNavigationMode::Orbit, [10., 0.])
    else {
        panic!("expected an orbit");
    };
    assert!((yaw - 0.1).abs() < 1e-9 && pitch == 0.);
}
