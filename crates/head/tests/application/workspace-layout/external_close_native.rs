// SPDX-License-Identifier: GPL-2.0-only
//! Verify a host-owned panel can close without a blank shared dock frame.
use super::dockspace_native::{open_toolbar_tool, viewport};
use super::dockspace_restart::select_dock_tab;
use super::orientation_close_native::drag_title;
use super::*;

#[test]
fn measure_done_selects_the_other_tab_in_the_first_frame() {
    let mut world = ApplicationWorld::new();
    super::browser_feature::open_visible_extruded_part(&mut world);
    let area = viewport(&mut world);
    open_toolbar_tool(&mut world, "Material Browser");
    drag_title(
        &mut world,
        "Material Browser",
        [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
    world
        .app()
        .owner
        .inspect_from_gui(oblikovati_gui::GuiInspectAction::StartMeasure)
        .unwrap();
    world.pump(Vec::new());
    let (position, size) = window_geometry("Material Browser").unwrap();
    drag_title(
        &mut world,
        "Measure",
        [
            (position[0] + size[0] * 0.5) as i32,
            (position[1] + 10.) as i32,
        ],
    );
    let node = super::super::diagnostics::window_dock_id("Material Browser").unwrap();
    assert_ne!(node, 0);
    assert_eq!(
        super::super::diagnostics::window_dock_id("Measure"),
        Some(node)
    );
    select_dock_tab(&mut world, "Measure");
    let close = world.run_in_window("Measure", "Done").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(close));
    for _ in 0..8 {
        let app = world.app();
        crate::frames::run_frames(
            &mut app.window,
            crate::HeadOptions { frames: Some(1) },
            &mut app.owner,
        )
        .unwrap();
        if window_geometry("Measure").is_none() {
            assert_eq!(
                super::super::diagnostics::window_dock_tab_visible("Material Browser"),
                Some(true)
            );
            world.close();
            return;
        }
    }
    panic!("Measure: expected close within eight frames");
}
