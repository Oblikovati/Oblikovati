// SPDX-License-Identifier: GPL-2.0-only
//! Check the application options path with native input.
use super::dockspace_restart::native_click_label;
use super::*;
use oblikovati_model::ui_scale::UiScaleOptions;

#[test]
fn file_options_changes_text_scale_and_retains_icon_factor() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let initial = UiScaleOptions::new(1., 1.5).unwrap();
    world.app().owner.set_ui_scale_from_gui(initial).unwrap();
    world.pump(Vec::new());
    native_click_label(&mut world, "File");
    native_click_label(&mut world, "Options...");
    let control = world.run_in_window("Application Options", "200%").center();
    super::super::x11_input::click(super::super::diagnostics::pointer_pixels(control));
    world.pump(Vec::new());
    assert_eq!(
        world.state().ui_scale,
        Some(UiScaleOptions::new(2., 1.5).unwrap())
    );
    world.save_capture("options-text-scale-200");
    world.close();
}
