// SPDX-License-Identifier: GPL-2.0-only
//! Exercise Home Recent controls through the native desktop input path (#3898).
use super::super::{diagnostics::pointer_pixels, toolbar_icon_center, x11_input};
use super::{
    ASSEMBLY, ApplicationWorld, HOME, OTHERS, SEARCHED, home_run, home_shows, lists::rows_by_size,
    save_recent_files,
};
use oblikovati_model::application_recent::{RecentDocument, RecentDocumentAccess};

#[test]
fn home_recent_last_row_clears_the_document_strip_in_a_short_window() {
    let mut world = ApplicationWorld::new();
    seed_scrollable_recent(&mut world);
    x11_input::resize_application_window([1454, 540]);
    world.pump(Vec::new());
    assert_eq!(world.extent(), [1454, 540]);
    assert!(!home_shows(&world, "RecentPart00"));
    for _ in 0..5 {
        x11_input::scroll([460, 380], -10);
        world.pump(Vec::new());
    }
    let bottom = world.viewport_bounds()[3] as f32;
    let last = home_run(&world, "RecentPart00").expect("expected the last Recent row");
    let clearance = bottom - last.maximum[1];
    println!("Last Recent filename clears the document strip by {clearance} px");
    world.save_capture("home-recent-at-scroll-end-1454x540");
    assert!(
        clearance >= 24.,
        "expected 24 px between the last filename and document strip, got {clearance}"
    );
    world.close();
}

fn seed_scrollable_recent(world: &mut ApplicationWorld) {
    world.close();
    let mut options =
        crate::native::preferences::load_native_application_options(world.preferences_directory())
            .unwrap();
    let recent = (0..24)
        .map(|index| RecentDocument {
            path: world
                .scratch_folder()
                .join(format!("RecentPart{index:02}.opd"))
                .to_string_lossy()
                .into_owned(),
            last_opened: index,
            pinned: false,
        })
        .collect();
    options.set_recent_documents(recent).unwrap();
    world.reopen_application();
}

#[test]
fn home_recent_grid_filters_and_restores_with_x11_input() {
    let mut world = open_native_recent();
    click_recent_control(&mut world, "Grid View");
    wait_for_home_label(&mut world, "Sort by", true);
    type_recent_search(&mut world, "part26");
    wait_for_home_label(&mut world, OTHERS[0], false);
    assert!(home_shows(&world, SEARCHED));
    for name in [OTHERS[0], OTHERS[1], ASSEMBLY] {
        assert!(!home_shows(&world, name), "expected {name} filtered out");
    }
    world.save_capture("x11-home-grid-search");
    click_recent_control(&mut world, "×");
    wait_for_home_label(&mut world, "Search recent", true);
    click_recent_control(&mut world, "Search");
    wait_for_home_label(&mut world, OTHERS[0], true);
    for name in [SEARCHED, OTHERS[0], OTHERS[1], ASSEMBLY] {
        assert!(home_shows(&world, name), "expected {name} restored");
    }
    world.save_capture("x11-home-grid-restored");
    world.close();
}

#[test]
fn home_recent_view_controls_show_which_layout_is_active() {
    let mut world = open_native_recent();
    world.hover([100, 300]);
    let selected = world.pixel_left_of("List View");
    let normal = world.pixel_left_of("Grid View");
    assert_ne!(selected, normal);
    click_recent_control(&mut world, "Grid View");
    world.hover([100, 300]);
    assert_eq!(world.pixel_left_of("Grid View"), selected);
    assert_eq!(world.pixel_left_of("List View"), normal);
    world.save_capture("home-recent-active-grid");
    world.close();
}

#[test]
fn home_recent_visible_size_heading_reverses_order_with_x11_input() {
    let mut world = open_native_recent();
    assert!(world.state().document.is_some());
    assert!(super::super::diagnostics::window_dock_id("Material Browser").is_none());
    assert!(super::super::diagnostics::window_dock_id("Appearance Browser").is_none());
    click_recent_control(&mut world, "Size");
    wait_for_home_label(&mut world, "Size ▾", true);
    let descending: Vec<u64> = rows_by_size(&mut world)
        .into_iter()
        .map(|(_, size)| size)
        .collect();
    assert_eq!(descending.len(), 4);
    assert!(descending.windows(2).all(|sizes| sizes[0] >= sizes[1]));
    assert!(
        descending.first() > descending.last(),
        "expected different sizes"
    );
    click_recent_control(&mut world, "Size ▾");
    wait_for_home_label(&mut world, "Size ▴", true);
    let ascending = rows_by_size(&mut world).into_iter().map(|(_, size)| size);
    assert!(ascending.eq(descending.into_iter().rev()));
    world.save_capture("x11-home-size-ascending");
    world.close();
}

fn open_native_recent() -> ApplicationWorld {
    let mut world = ApplicationWorld::new();
    save_recent_files(&mut world);
    x11_input::focus_window();
    x11_input::click(toolbar_icon_center("Home"));
    world.pump(Vec::new());
    world
}

fn type_recent_search(world: &mut ApplicationWorld, query: &str) {
    click_recent_control(world, "Search recent");
    for character in query.chars() {
        x11_input::press_keysym(u32::from(character));
        world.pump(Vec::new());
    }
    wait_for_home_label(world, query, true);
    click_recent_control(world, "Search");
}

fn wait_for_home_label(world: &mut ApplicationWorld, label: &str, visible: bool) {
    for _ in 0..3 {
        if home_shows(world, label) == visible {
            return;
        }
        world.pump(Vec::new());
    }
    assert_eq!(
        home_shows(world, label),
        visible,
        "visibility of {label:?} in Home: {:?}",
        world.window_text(HOME)
    );
}

fn click_recent_control(world: &mut ApplicationWorld, label: &str) {
    let control = home_run(world, label).unwrap_or_else(|| panic!("expected {label:?} in Home"));
    x11_input::click(pointer_pixels(control.center()));
    world.pump(Vec::new());
}
