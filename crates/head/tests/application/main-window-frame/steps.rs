// SPDX-License-Identifier: GPL-2.0-only
//! Native window-frame behavior driven by the shared application scenarios.
use super::ApplicationWorld;
use cucumber::{given, then, when};
#[path = "fixture.rs"]
mod fixture;
#[cfg(target_os = "linux")]
pub(super) use fixture::{bundled_frame_fixture, open_dirty_frame_part_from};
pub(super) use fixture::{open_dirty_frame_part, open_frame_part};

#[when("the user invokes the desktop window Close shortcut")]
#[when("the user presses Alt+F4")]
fn invoke_desktop_close(world: &mut ApplicationWorld) {
    world.application_close_requested = true;
    #[cfg(target_os = "linux")]
    if world.app().window.native.view.uses_wayland() {
        // The compositor owns Alt+F4 and sends xdg_toplevel.close to the client.
        world.pump(vec![oblikovati_viewport::ViewportEvent::CloseRequested]);
        world.pump(Vec::new());
        return;
    }
    super::x11_input::focus_window();
    super::x11_input::press_alt_f4();
    world.pump(Vec::new());
    world.pump(Vec::new());
}

#[given("the user creates a sketch on the XY datum plane without saving the part")]
fn edit_open_part(world: &mut ApplicationWorld) {
    let document = world.closed_document.expect("expected the writable Part");
    assert_eq!(world.active_id(), document);
    let storage = &world.state().document_storage[&document];
    assert!(!storage.unsaved_edits);
    let path = std::path::PathBuf::from(storage.path.as_ref().unwrap());
    world.bytes_before_save = Some(std::fs::read(&path).unwrap());
    world.saved_path = Some(path);
    world.add_finished_sketch();
    assert!(world.state().document_storage[&document].unsaved_edits);
}

#[then("the main window remains open and the part retains its unsaved changes")]
fn unsaved_part_remains_open(world: &mut ApplicationWorld) {
    let document = world.closed_document.expect("expected the edited part");
    assert_eq!(world.active_id(), document);
    assert!(world.document_is_open(document));
    assert!(world.state().document_storage[&document].unsaved_edits);
    if let (Some(path), Some(original)) = (&world.saved_path, &world.bytes_before_save) {
        assert_eq!(&std::fs::read(path).unwrap(), original);
    }
}

#[given("the MCAD Software is open in a restored window")]
fn restored_window(world: &mut ApplicationWorld) {
    assert!(!world.app().window.native.view.window_maximized().unwrap());
    if !world.app().window.native.view.uses_wayland() {
        world.window_frame_before = Some(super::x11_input::application_bounds());
        super::x11_input::focus_window();
    }
}

#[given("Home is active with no model, drawing or presentation document open")]
fn home_without_documents(world: &mut ApplicationWorld) {
    super::home_recent_documents::open_home(world);
    assert!(world.state().documents.is_empty());
}

fn saved_frame_part(world: &mut ApplicationWorld) -> u64 {
    let Some(&part) = world.parts.last() else {
        return open_frame_part(world);
    };
    assert_eq!(world.active_id(), part);
    assert!(!world.state().document_storage[&part].unsaved_edits);
    part
}

#[when("the user double-clicks an empty region of the application bar")]
#[when("the user double-clicks an empty region of the application bar again")]
#[when("the user double-clicks the application title bar to change the window state")]
fn double_click_bar(world: &mut ApplicationWorld) {
    let width = world.app().window.native.drawable_extent().unwrap()[0] as i32;
    let point = [width - 390, 18];
    super::x11_input::click(point);
    super::x11_input::click(point);
    world.pump(Vec::new());
}
#[then("the main window is maximized")]
fn maximized(world: &mut ApplicationWorld) {
    super::main_window_frame_native::pump_until(world, true);
}
#[then("the window controls show Restore")]
fn restore_tooltip(world: &mut ApplicationWorld) {
    control_tooltip(world, "Restore");
}
#[then("the window controls show Maximize")]
fn maximize_tooltip(world: &mut ApplicationWorld) {
    control_tooltip(world, "Maximize");
}
fn control_tooltip(world: &mut ApplicationWorld, label: &str) {
    hover_control(world, 1, label);
}
fn hover_control(world: &mut ApplicationWorld, index: usize, label: &str) {
    super::main_window_frame_accessibility::hover_frame_point(
        world,
        super::main_window_frame_native::frame_control_center(index),
    );
    std::thread::sleep(std::time::Duration::from_millis(700));
    world.pump(Vec::new());
    world.expect_text(label);
}
#[then("the main window returns to its previous size and position")]
fn restored_bounds(world: &mut ApplicationWorld) {
    super::main_window_frame_native::pump_until(world, false);
    assert_eq!(
        super::x11_input::application_bounds(),
        world.window_frame_before.unwrap()
    );
}

#[given("the main window of the MCAD Software is maximized")]
fn maximize_main_window(world: &mut ApplicationWorld) {
    open_frame_part(world);
    super::x11_input::focus_window();
    super::main_window_frame_native::press_maximize(world);
    super::main_window_frame_native::pump_until(world, true);
    world.window_frame_before = Some(super::x11_input::application_bounds());
}
#[when("the user attempts to resize it from the left boundary")]
fn drag_maximized_edge(world: &mut ApplicationWorld) {
    let height = super::x11_input::application_bounds()[3];
    super::main_window_frame_native::drag_resize_handle(world, [1, height / 2], [40, height / 2]);
}
#[then("the window retains its maximized bounds")]
fn maximized_bounds_unchanged(world: &mut ApplicationWorld) {
    assert_eq!(
        super::x11_input::application_bounds(),
        world.window_frame_before.unwrap()
    );
    assert!(world.app().window.native.view.window_maximized().unwrap());
}
#[then("Restore remains available in the application bar")]
fn restore_remains_available(world: &mut ApplicationWorld) {
    control_tooltip(world, "Restore");
}

#[given("the restored main window occupies a known size and position within the desktop work area")]
fn remember_restored_bounds(world: &mut ApplicationWorld) {
    save_active_part_fixture(world);
    restored_window(world);
    world.previous_document = Some(world.active_id());
}
#[when("the user selects Maximize in the application bar")]
#[when("the user selects Restore in the application bar")]
fn click_maximize_restore(world: &mut ApplicationWorld) {
    super::main_window_frame_native::press_maximize(world);
}
#[then("the window fills the monitor's work area and respects the taskbar")]
fn maximized_to_work_area(world: &mut ApplicationWorld) {
    super::main_window_frame_native::pump_until(world, true);
    assert_eq!(
        super::x11_input::application_bounds(),
        super::x11_input::desktop_work_area()
    );
}
#[then("Restore replaces Maximize in the same position")]
fn restore_replaces_maximize(world: &mut ApplicationWorld) {
    control_tooltip(world, "Restore");
}
#[then("Maximize replaces Restore in the same position")]
fn maximize_replaces_restore(world: &mut ApplicationWorld) {
    control_tooltip(world, "Maximize");
}
#[then("the ribbon and viewport lay out within the enlarged window")]
fn enlarged_window_layout(world: &mut ApplicationWorld) {
    let bounds = super::x11_input::application_bounds();
    let (position, size) =
        super::window_geometry("##application-ribbon").expect("expected the ribbon");
    assert!(position[0] >= 0. && position[1] >= 0.);
    assert!(size[0] <= bounds[2] as f32 && size[1] < bounds[3] as f32);
    assert!(world.app().window.native.drawable_extent().unwrap()[1] as i32 >= bounds[3]);
}
#[then("the window returns to its previous size and position")]
fn prior_window_bounds(world: &mut ApplicationWorld) {
    restored_bounds(world);
}

#[test]
#[ignore = "requires a Wayland desktop compositor with room to maximize"]
fn wayland_frame_maximizes_and_restores_the_saved_part() {
    use super::main_window_frame_native::{press_maximize, pump_until};

    let mut world = ApplicationWorld::new();
    assert!(world.app().window.native.view.uses_wayland());
    let document = open_frame_part(&mut world);
    let restored = world.extent();
    maximize_tooltip(&mut world);
    press_maximize(&mut world);
    pump_until(&mut world, true);
    let maximized = world.extent();
    assert!(
        (0..2).all(|axis| maximized[axis] > restored[axis]),
        "expected growth from {restored:?} to {maximized:?}"
    );
    expect_wayland_frame_layout(&mut world, maximized, document);
    restore_tooltip(&mut world);
    press_maximize(&mut world);
    pump_until(&mut world, false);
    assert_eq!(world.extent(), restored);
    expect_wayland_frame_layout(&mut world, restored, document);
    maximize_tooltip(&mut world);
    world.close();
}

fn expect_wayland_frame_layout(world: &mut ApplicationWorld, extent: [u32; 2], document: u64) {
    let [left, top, right, bottom] = world.viewport_bounds();
    assert!(left < right && right <= extent[0] && top < bottom && bottom <= extent[1]);
    let [left, top, right, bottom] =
        super::diagnostics::window_pixel_bounds("##application-ribbon").unwrap();
    assert!(0. <= left && right <= extent[0] as f32 && 0. <= top && bottom <= extent[1] as f32);
    controls_remain_in_bar(world);
    world.expect_active(document);
    super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[given("the main window is restored")]
fn remember_restored_part(world: &mut ApplicationWorld) {
    remember_part_view(world);
    assert!(!world.app().window.native.view.window_maximized().unwrap());
}
#[given("the main window is maximized")]
fn remember_maximized_part(world: &mut ApplicationWorld) {
    super::x11_input::focus_window();
    super::main_window_frame_native::press_maximize(world);
    super::main_window_frame_native::pump_until(world, true);
    remember_part_view(world);
}
fn remember_part_view(world: &mut ApplicationWorld) {
    save_active_part_fixture(world);
    world.previous_document = Some(world.active_id());
    world.observed_view_camera = Some(crate::HeadWindow::view_camera(&world.app().window).unwrap());
    super::x11_input::focus_window();
}
fn save_active_part_fixture(world: &mut ApplicationWorld) {
    let document = world.active_id();
    if world.state().document_storage[&document].path.is_none() {
        world.save_part_to_file(document, "main-window-frame.opd");
    }
    assert!(!world.state().document_storage[&document].unsaved_edits);
}

#[when("the user selects Minimize in the application bar")]
fn minimize_part_window(world: &mut ApplicationWorld) {
    super::main_window_frame_native::pump_minimize_button(world);
}
#[then("the main window is minimized and its taskbar entry remains available")]
fn minimized_with_taskbar_entry(_: &mut ApplicationWorld) {
    assert!(super::x11_input::window_has_state(b"_NET_WM_STATE_HIDDEN"));
    assert!(!super::x11_input::window_has_state(
        b"_NET_WM_STATE_SKIP_TASKBAR"
    ));
}
#[when("the user activates the application from the taskbar")]
fn activate_from_taskbar(world: &mut ApplicationWorld) {
    std::thread::sleep(std::time::Duration::from_millis(500));
    super::x11_input::click_test_taskbar();
    super::main_window_frame_native::wait_for_hidden(false);
    world.pump(Vec::new());
}
#[then("the main window returns in the restored state")]
fn returns_restored(world: &mut ApplicationWorld) {
    super::main_window_frame_native::pump_until(world, false);
}
#[then("the main window returns in the maximized state")]
fn returns_maximized(world: &mut ApplicationWorld) {
    super::main_window_frame_native::pump_until(world, true);
}
#[then("the same part and ribbon tab are active")]
fn same_part_and_ribbon(world: &mut ApplicationWorld) {
    assert_eq!(world.active_id(), world.previous_document.unwrap());
    world.expect_text("Start 2D Sketch");
}
#[then("the displayed part retains its view orientation")]
fn same_view_orientation(world: &mut ApplicationWorld) {
    assert_eq!(
        crate::HeadWindow::view_camera(&world.app().window).unwrap(),
        world.observed_view_camera.unwrap()
    );
}

#[given("the main window is restored within the desktop work area")]
fn restored_part_window(world: &mut ApplicationWorld) {
    restored_window(world);
    world.previous_document = Some(world.active_id());
}
#[when("the user drags an empty region of the application bar within the work area")]
fn drag_application_bar(world: &mut ApplicationWorld) {
    let width = world.app().window.native.drawable_extent().unwrap()[0] as i32;
    let point = [width - 390, 18];
    super::x11_input::move_pointer(point);
    world.pump(Vec::new());
    super::x11_input::left_button(point, true);
    world.pump(Vec::new());
    super::x11_input::move_pointer([point[0] + 20, 30]);
    world.pump(Vec::new());
    super::x11_input::move_pointer([point[0] + 70, 80]);
    world.pump(Vec::new());
    super::x11_input::left_button([point[0] + 70, 80], false);
    world.pump(Vec::new());
}
#[then("the entire window follows the drag without changing size")]
fn window_moved_without_resize(world: &mut ApplicationWorld) {
    let before = world.window_frame_before.unwrap();
    let after = super::x11_input::application_bounds();
    assert_ne!(&before[..2], &after[..2]);
    assert_eq!(&before[2..], &after[2..]);
}
#[then("the same part remains active in the viewport")]
fn same_part_in_viewport(world: &mut ApplicationWorld) {
    assert_eq!(world.active_id(), world.previous_document.unwrap());
    super::workspace_layout::browser_feature::expect_solid_visible(world);
    assert!(
        world
            .app()
            .window
            .native
            .drawable_extent()
            .unwrap()
            .iter()
            .all(|value| *value > 0)
    );
}

#[given("the MCAD Software is open in a restored window with an active part")]
fn restored_window_with_saved_part(world: &mut ApplicationWorld) {
    let part = saved_frame_part(world);
    world.previous_document = Some(part);
    restored_window(world);
}
#[when("the user drags the bottom right corner inward past the supported minimum size")]
fn shrink_past_minimum(world: &mut ApplicationWorld) {
    resize_from_bottom_right(world, [100, 100]);
}

fn resize_from_bottom_right(world: &mut ApplicationWorld, target: [i32; 2]) {
    let bounds = super::x11_input::application_bounds();
    super::main_window_frame_native::drag_resize_handle(
        world,
        [bounds[2] - 2, bounds[3] - 2],
        target,
    );
}
#[then("the window stops shrinking at its minimum size")]
fn minimum_window_bounds(_: &mut ApplicationWorld) {
    assert_eq!(&super::x11_input::application_bounds()[2..], &[640, 480]);
}
#[then("Minimize, Maximize and Close remain fully visible and usable")]
fn minimum_controls_usable(world: &mut ApplicationWorld) {
    super::main_window_frame_accessibility::assert_drawn_frame_controls(world);
    super::main_window_frame_native::press_maximize(world);
    super::main_window_frame_native::pump_until(world, true);
    super::main_window_frame_native::press_maximize(world);
    super::main_window_frame_native::pump_until(world, false);
    super::main_window_frame_native::pump_minimize_button(world);
    activate_from_taskbar(world);
}
#[then("the active part remains displayed in the resized viewport")]
fn minimum_viewport_keeps_part(world: &mut ApplicationWorld) {
    same_part_in_viewport(world);
    assert!(!world.state().document_storage[&world.active_id()].unsaved_edits);
    world.expect_no_text("Command failed");
    assert_eq!(&super::x11_input::application_bounds()[2..], &[640, 480]);
    let [left, _, right, _] = world.viewport_bounds();
    let width = world.extent()[0];
    assert!(
        (right - left) * 5 >= width * 2,
        "viewport {left}..{right} in {width} pixels: expected room to see the Part"
    );
}

#[test]
#[ignore = "requires an X11 window manager and the scenario source with the native frame fixture"]
fn native_close_at_minimum_size_exits_with_a_clean_part() {
    let mut world = ApplicationWorld::new();
    restored_window_with_saved_part(&mut world);
    let original_viewport = world.viewport_bounds();
    let [width, height] = world.extent().map(|value| value as i32);
    shrink_past_minimum(&mut world);
    minimum_viewport_keeps_part(&mut world);
    resize_from_bottom_right(&mut world, [width - 2, height - 2]);
    assert_eq!(
        world.viewport_bounds(),
        original_viewport,
        "restored dock widths"
    );
    shrink_past_minimum(&mut world);
    minimum_window_bounds(&mut world);
    minimum_viewport_keeps_part(&mut world);
    // Activating Close here can destroy the clean window without obscuring the
    // minimum-size scenario's final check that the Part is still displayed.
    super::application_closing::request_application_close(&mut world);
    assert!(world.app.is_none());
}

#[then("File and the ribbon tabs appear on a separate row beneath the application bar")]
fn ribbon_tabs_beneath_bar(world: &mut ApplicationWorld) {
    super::wireframe_ribbon::assert_controls_above_tabs(world);
    super::wireframe_ribbon::assert_tabs_share_row(world);
}
#[then("the active tab's command panels are below the ribbon tab row")]
fn panels_beneath_tabs(world: &mut ApplicationWorld) {
    super::wireframe_ribbon::assert_panels_below_tabs(world);
}
#[then("the window controls remain in the application bar")]
fn controls_remain_in_bar(world: &mut ApplicationWorld) {
    let file = world.run("File");
    let [_, _, right, bottom] =
        super::diagnostics::window_pixel_bounds("##application-window-controls").unwrap();
    assert!(bottom <= super::pointer_pixels(file.minimum)[1] as f32);
    let width = world.app().window.native.drawable_extent().unwrap()[0] as f32;
    assert!((right - width).abs() <= 1.);
}

#[given(regex = r"^the MCAD Software is open in a restored window showing (Home|an active part)$")]
fn restored_window_in_context(world: &mut ApplicationWorld, context: String) {
    if context == "an active part" {
        saved_frame_part(world);
    }
    restored_window(world);
}
#[then("the application bar forms the top of the window with no separate system title bar")]
fn application_bar_is_window_top(world: &mut ApplicationWorld) {
    if !world.app().window.native.view.uses_wayland() {
        assert_eq!(super::x11_input::frame_extents(), [0, 0, 0, 0]);
    }
    let (position, _) = super::window_geometry("##application-ribbon").unwrap();
    assert_eq!(position, [0., 0.]);
}
#[then("quick access command icons sit at the leading side of the application bar")]
fn quick_access_leads_bar(world: &mut ApplicationWorld) {
    let icons = super::quick_access_icons::toolbar_images();
    assert_eq!(icons.len(), 13);
    let file = world.run("File");
    let (controls, _) = super::window_geometry("##application-window-controls").unwrap();
    assert!(
        icons
            .iter()
            .all(|icon| icon.center[1] < file.minimum[1] && icon.center[0] < controls[0])
    );
    assert!(
        icons
            .windows(2)
            .all(|pair| pair[0].center[0] < pair[1].center[0])
    );
}
#[then("the application or active document title occupies the same bar")]
fn title_shares_bar(world: &mut ApplicationWorld) {
    if world.parts.last().is_some() {
        super::wireframe_ribbon::assert_document_context_above_tabs(world);
        return;
    }
    let file = world.run("File");
    assert!(
        world
            .frame
            .runs
            .iter()
            .any(|run| run.text == "Oblikovati" && run.center()[1] < file.minimum[1])
    );
}
#[then("Minimize, Maximize and Close sit at the trailing edge in that order")]
fn controls_at_trailing_edge(world: &mut ApplicationWorld) {
    controls_remain_in_bar(world);
    for (index, name) in ["Minimize", "Maximize", "Close"].into_iter().enumerate() {
        hover_control(world, index, name);
    }
}
