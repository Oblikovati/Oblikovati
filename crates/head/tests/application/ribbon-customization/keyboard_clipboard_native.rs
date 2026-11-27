// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the Keyboard tab's copy button through the native clipboard.
use super::*;
use crate::native::gui_loop_tests::application_bdd::x11_input;
use oblikovati_viewport::ViewportClipboard;

#[test]
#[ignore = "requires private X11 or fullscreen Weston nested on private X11"]
fn keyboard_clipboard_copies_the_complete_catalog_in_display_order() {
    let mut world = ApplicationWorld::new();
    assert_eq!(
        world.app().window.native.view.uses_wayland(),
        std::env::var_os("WAYLAND_DISPLAY").is_some()
    );
    world.add_part();
    open_tools_customization(&mut world);
    click_in_customize(&mut world, "Keyboard");
    let search = keyboard_initial::search_label(&world).clone();
    world.click([(search.minimum[0] - 70.) as i32, search.center()[1] as i32]);
    world.type_text("Undo");
    assert!(!world.window_text("Customize").contains("HomeView"));

    let table = copy_keyboard_table(&mut world);
    expect_catalog(&mut world, &table);
    assert_ne!(world.run_in_window("Customize", "Apply").alpha, u8::MAX);
    click_in_customize(&mut world, "Close");
    world.close();
}

fn copy_keyboard_table(world: &mut ApplicationWorld) -> String {
    let label = world.run_in_window("Customize", "CopytoClipboard");
    let point = diagnostics::pointer_pixels(label.center());
    if world.app().window.native.view.uses_wayland() {
        x11_input::window_targets::click_fullscreen_output(point);
    } else {
        x11_input::click(point);
    }
    world.pump(Vec::new());
    let clipboard = &world.app().window.native.clipboard;
    assert_eq!(clipboard.issue(), None);
    clipboard
        .read_text()
        .unwrap()
        .expect("expected copied keyboard table")
}

fn expect_catalog(world: &mut ApplicationWorld, table: &str) {
    let mut lines = table.lines();
    assert_eq!(lines.next(), Some("Keys\tCommand Name\tType\tCategory"));
    let rows: Vec<_> = lines
        .map(|line| line.split('\t').collect::<Vec<_>>())
        .collect();
    assert!(rows.iter().all(|row| row.len() == 4));
    assert_eq!(rows.len(), world.state().keymap_bindings.len());
    assert!(
        rows.iter()
            .any(|row| row == &["Ctrl+Z", "Undo", "builtin", "Application"])
    );
    assert!(rows.iter().any(|row| row[1] == "Home View"));
    assert!(
        rows.windows(2)
            .all(|pair| pair[0][1].to_lowercase() <= pair[1][1].to_lowercase()),
        "expected commands in the Keyboard list's display order, got {rows:?}"
    );
}
