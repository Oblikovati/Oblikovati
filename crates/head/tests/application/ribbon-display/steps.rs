// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, diagnostics::window_geometry};
use cucumber::{given, then, when};
mod compact_options;
mod double_click;
mod tabs_only;
const VIEW_GROUPS: [&str; 4] = ["Visibility", "Appearance", "Windows", "Navigate"];
const WINDOWS_COMMAND: &str = "User Interface";
fn band_height() -> f32 {
    window_geometry("##application-ribbon")
        .expect("ribbon: expected a drawn band")
        .1[1]
}
fn activate_display_control(world: &mut ApplicationWorld) {
    world.ribbon_height = band_height();
    world.click_text("Ribbon Display");
}
fn open_part_on_view(world: &mut ApplicationWorld) {
    use crate::HeadWork;
    world.app().owner.create_part_document().unwrap();
    world.previous_document = world.app().owner.gui_command_state().unwrap().document;
    world.pump(Vec::new());
    world.click_text("View");
}
fn expect_shorter_band(world: &ApplicationWorld) {
    assert!(
        band_height() < world.ribbon_height,
        "ribbon {} after {}: expected a shorter band",
        band_height(),
        world.ribbon_height
    );
}
fn expect_compact_groups(world: &ApplicationWorld) {
    for group in VIEW_GROUPS {
        world.expect_text(group);
    }
    world.expect_no_text(WINDOWS_COMMAND);
}
pub(super) fn click_display_menu(world: &mut ApplicationWorld) {
    let control = world.run("Ribbon Display");
    let center = control.center();
    let arrow = world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.text == "v"
                && run.center()[0] > control.maximum[0]
                && (run.center()[1] - center[1]).abs() < 5.
        })
        .min_by(|left, right| left.center()[0].total_cmp(&right.center()[0]))
        .expect("expected the menu arrow beside Ribbon Display")
        .center();
    world.click(super::diagnostics::pointer_pixels(arrow));
}
#[given("the ribbon display control is configured to cycle through all modes")]
#[given("the ribbon display control is set to cycle through all modes")]
fn cycling(world: &mut ApplicationWorld) {
    click_display_menu(world);
    assert!(
        world.checked("Cycle through All"),
        "expected the cycling default"
    );
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
}
#[then("the full View command groups remain visible")]
fn view_groups_remain(world: &mut ApplicationWorld) {
    world.expect_text(WINDOWS_COMMAND);
}
#[given("the View ribbon tab displays its full groups of commands")]
#[given("the View ribbon is expanded")]
fn view_expanded(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.expect_text(WINDOWS_COMMAND);
}
#[when("I activate the ribbon display control")]
#[when("I activate the ribbon display control once")]
#[when("I activate the ribbon display control again")]
fn activate(world: &mut ApplicationWorld) {
    activate_display_control(world);
}
#[then("the View groups are shown as compact buttons")]
#[then("the View ribbon shows compact buttons for Visibility, Appearance, Windows and Navigate")]
fn compact_buttons(world: &mut ApplicationWorld) {
    expect_compact_groups(world);
}
#[then("the viewport has more vertical space")]
fn more_space(world: &mut ApplicationWorld) {
    expect_shorter_band(world);
}
#[when("I select the Windows group button")]
#[when("I select the Windows group title")]
fn open_windows_group(world: &mut ApplicationWorld) {
    world.click_text("Windows");
}
#[then("a temporary panel displays that group's commands")]
#[then("a temporary panel displays the Windows commands")]
fn windows_panel(world: &mut ApplicationWorld) {
    world.expect_text(WINDOWS_COMMAND);
}
#[given("the View ribbon displays compact group buttons")]
fn compact_state(world: &mut ApplicationWorld) {
    open_part_on_view(world);
    activate_display_control(world);
    expect_compact_groups(world);
}
#[then("the View group titles appear in a narrow strip")]
#[then("the View ribbon displays group titles in a narrow strip")]
fn title_strip(world: &mut ApplicationWorld) {
    expect_compact_groups(world);
    expect_shorter_band(world);
}
#[given("the View ribbon displays its group titles")]
fn title_state(world: &mut ApplicationWorld) {
    compact_state(world);
    activate_display_control(world);
    expect_compact_groups(world);
}
#[then("the tabs remain visible without a permanent group strip")]
#[then("the ribbon shows its tabs without the group strip")]
fn tabs_only(world: &mut ApplicationWorld) {
    world.expect_text("3D Model");
    world.expect_text("View");
    world.expect_no_text("Windows");
    expect_shorter_band(world);
}
#[when("I select the View tab")]
fn select_view_tab(world: &mut ApplicationWorld) {
    world.click_text("View");
}
#[then("its commands appear in a temporary panel")]
#[then("the View commands appear in a temporary panel below the tabs")]
fn view_commands_below(world: &mut ApplicationWorld) {
    world.expect_text(WINDOWS_COMMAND);
    let below = band_height();
    assert!(
        world.run(WINDOWS_COMMAND).minimum[1] > below,
        "expected the panel below the tabs"
    );
}
#[when("I click in an empty area of the viewport")]
#[when("I click an empty area of the viewport")]
fn click_viewport(world: &mut ApplicationWorld) {
    let extent = world.extent();
    world.click([extent[0] as i32 - 100, extent[1] as i32 - 100]);
}
#[then("the expanded View ribbon remains visible")]
fn expanded_remains(world: &mut ApplicationWorld) {
    world.expect_text(WINDOWS_COMMAND);
    assert!(
        world.run(WINDOWS_COMMAND).minimum[1] < band_height(),
        "expected the full ribbon"
    );
}
#[when("I open the ribbon display control menu")]
fn open_menu(world: &mut ApplicationWorld) {
    click_display_menu(world);
}
#[then("the menu offers minimization to tabs, panel titles and panel buttons")]
fn menu_choices(world: &mut ApplicationWorld) {
    for choice in [
        "Minimize to Tabs",
        "Minimize to Panel Titles",
        "Minimize to Panel Buttons",
    ] {
        world.expect_text(choice);
    }
}
#[then("it offers a cycle-through-all choice")]
fn cycle_choice(world: &mut ApplicationWorld) {
    world.expect_text("Cycle through All");
}
#[when("I choose Minimize to Tabs")]
fn choose_tabs(world: &mut ApplicationWorld) {
    world.click_text("Minimize to Tabs");
}
#[then("the command groups are hidden and the ribbon tabs remain visible")]
fn groups_hidden(world: &mut ApplicationWorld) {
    world.expect_no_text("Navigate");
    world.expect_text("3D Model");
}
#[when("I select the 3D Model tab")]
fn select_model(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
}
#[then("its command groups appear temporarily below the tabs")]
fn model_commands(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
}
#[then("the temporary command groups close")]
fn groups_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Start 2D Sketch");
    assert!(window_geometry("##ribbon-temporary-panel").is_none());
}
#[then("the 3D Model tab remains selected")]
fn model_selected(world: &mut ApplicationWorld) {
    // A selected tab has its own fill, so the pixel left of each label tells them apart.
    let selected = world.pixel_left_of("3D Model");
    let other = world.pixel_left_of("View");
    assert_ne!(selected, other, "expected the 3D Model tab to be selected");
}
#[then("Minimize to Tabs is checked")]
fn tabs_checked(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Minimize to Tabs"),
        "expected a checked choice"
    );
}
const DRAWING_GROUPS: [&str; 4] = ["Create", "Modify", "Sketch", "Sheets"];
#[given("the Place Views tab shows its full command groups")]
fn place_views_full(world: &mut ApplicationWorld) {
    world.expect_text("Base View");
    world.ribbon_height = band_height();
}
fn double_click_place_views(world: &mut ApplicationWorld) {
    let center = world.run("Place Views").center().map(|value| value as i32);
    world.pump(crate::native::gui_loop_tests::pointer_click_events(
        center, 2,
    ));
    world.pump(Vec::new());
}
#[when("I double-click the active Place Views tab")]
#[when("I double-click the active Place Views tab again")]
fn double_click(world: &mut ApplicationWorld) {
    double_click_place_views(world);
}
fn expect_compact_drawing(world: &mut ApplicationWorld) {
    for group in DRAWING_GROUPS {
        world.expect_text(group);
    }
    world.expect_no_text("Base View");
}
#[then("the ribbon shows compact Create, Modify, Sketch and Sheets group buttons")]
#[then("its ribbon displays compact Create, Modify, Sketch and Sheets group buttons")]
fn compact_drawing(world: &mut ApplicationWorld) {
    expect_compact_drawing(world);
}
#[then("the ribbon shows the group titles in a narrow strip")]
fn titles_strip(world: &mut ApplicationWorld) {
    for group in DRAWING_GROUPS {
        world.expect_text(group);
    }
    world.expect_no_text("Base View");
    assert!(
        band_height() < world.ribbon_height,
        "expected a narrower ribbon"
    );
}
#[then("the ribbon shows tabs without the group strip")]
fn tabs_without_strip(world: &mut ApplicationWorld) {
    world.expect_text("Place Views");
    for group in ["Modify", "Sheets"] {
        world.expect_no_text(group);
    }
}
#[then("the full Place Views command groups are visible")]
fn full_groups(world: &mut ApplicationWorld) {
    world.expect_text("Base View");
    world.expect_text("New Sheet");
}
#[then("the drawing document remains active")]
fn drawing_remains(world: &mut ApplicationWorld) {
    let state = world.state();
    assert_eq!(state.document, world.previous_document);
    assert_eq!(
        state.document_type,
        Some(oblikovati_api::types::DOCUMENT_DRAWING)
    );
}
#[given("a drawing and a part are open in the MCAD Software")]
fn drawing_and_part(world: &mut ApplicationWorld) {
    world.other_document = Some(world.add_part());
    world.previous_document = Some(world.add_drawing());
}
#[given("the drawing is active with its ribbon in compact group-button mode")]
#[given("the Place Views ribbon shows compact group buttons")]
fn compact_mode(world: &mut ApplicationWorld) {
    activate_display_control(world);
    expect_compact_drawing(world);
}
#[when("I activate the part document")]
fn activate_part(world: &mut ApplicationWorld) {
    let part = world.other_document.expect("expected a part");
    world.click_tab(part);
}
#[then("the part ribbon displays its command groups as compact buttons")]
fn part_compact(world: &mut ApplicationWorld) {
    world.expect_no_text("Start 2D Sketch");
    world.expect_no_text("Extrude");
}
#[then("those buttons include Sketch, Create and Modify")]
fn part_groups(world: &mut ApplicationWorld) {
    for group in ["Sketch", "Create", "Modify"] {
        world.expect_text(group);
    }
}
#[when("I return to the drawing document")]
fn back_to_drawing(world: &mut ApplicationWorld) {
    let drawing = world.previous_document.expect("expected a drawing");
    world.click_tab(drawing);
}
#[when("I select the Create group button")]
fn select_create(world: &mut ApplicationWorld) {
    world.click_text("Create");
    world.pump(Vec::new());
}
#[then("a temporary Create panel displays drawing-view commands")]
fn create_panel(world: &mut ApplicationWorld) {
    world.expect_text("Base View");
}
#[when("I select the Modify group button")]
fn select_modify(world: &mut ApplicationWorld) {
    world.click_text("Modify");
    world.pump(Vec::new());
}
#[then("the Create panel closes")]
fn create_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Base View");
    world.expect_no_text("Extrude");
}
#[then("a temporary Modify panel displays modification commands")]
fn modify_panel(world: &mut ApplicationWorld) {
    world.expect_text("Break");
}
#[when("I click an empty area of the drawing outside the panel")]
fn click_drawing(world: &mut ApplicationWorld) {
    world.click([1000, 600]);
}
#[then("the temporary Modify panel closes")]
fn modify_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Break");
}
#[then("the compact group buttons remain visible")]
fn compact_remains(world: &mut ApplicationWorld) {
    expect_compact_drawing(world);
}
#[given("a drawing is active in the MCAD Software")]
fn a_drawing_active(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_drawing());
}
#[given("the temporary Create group panel is open")]
fn create_group_open(world: &mut ApplicationWorld) {
    select_create(world);
    world.expect_text("Base View");
}
#[when("I select the Annotate tab")]
fn select_annotate(world: &mut ApplicationWorld) {
    world.click_text("Annotate");
    world.pump(Vec::new());
}
#[then("the temporary Create panel closes")]
fn temporary_create_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Base View");
}
#[then("the ribbon displays compact Annotate group buttons")]
fn compact_annotate(world: &mut ApplicationWorld) {
    world.expect_no_text("Baseline");
    world.expect_no_text("Sheets");
}
#[then("those buttons include Dimension, Text and Symbols")]
fn annotate_groups(world: &mut ApplicationWorld) {
    for group in ["Dimension", "Text", "Symbols"] {
        world.expect_text(group);
    }
}
#[then("the drawing remains active")]
fn drawing_stays(world: &mut ApplicationWorld) {
    drawing_remains(world);
}
#[given("the Annotate ribbon is expanded")]
fn annotate_expanded(world: &mut ApplicationWorld) {
    select_annotate(world);
    world.expect_text("Baseline");
    world.ribbon_height = band_height();
}
#[when("I choose Minimize to Panel Titles")]
fn minimize_to_titles(world: &mut ApplicationWorld) {
    world.click_text("Minimize to Panel Titles");
}
fn expect_annotate_titles(world: &mut ApplicationWorld) {
    annotate_groups(world);
    world.expect_no_text("Baseline");
    assert!(
        band_height() < world.ribbon_height,
        "expected a narrower ribbon"
    );
}
#[then("the Annotate command groups are reduced to a narrow strip of titles")]
#[then("the ribbon displays the group-title strip")]
fn annotate_titles(world: &mut ApplicationWorld) {
    expect_annotate_titles(world);
}
#[then("the full Annotate command groups are visible")]
fn full_annotate(world: &mut ApplicationWorld) {
    world.expect_text("Baseline");
    world.expect_text("Leader");
}
#[then("Minimize to Panel Titles is checked")]
fn titles_checked(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Minimize to Panel Titles"),
        "expected the chosen mode checked"
    );
}
#[when("I choose Minimize to Panel Buttons")]
fn minimize_to_buttons(world: &mut ApplicationWorld) {
    world.click_text("Minimize to Panel Buttons");
}
#[then("the Annotate command groups appear as compact buttons")]
fn annotate_buttons(world: &mut ApplicationWorld) {
    annotate_groups(world);
    world.expect_no_text("Baseline");
}
#[then("Minimize to Panel Buttons is checked")]
fn buttons_checked(world: &mut ApplicationWorld) {
    assert!(
        world.checked("Minimize to Panel Buttons"),
        "expected the chosen mode checked"
    );
}
fn choose_tabs_mode(world: &mut ApplicationWorld) {
    click_display_menu(world);
    world.click_text("Minimize to Tabs");
    world.expect_no_text("Start 2D Sketch");
}
#[given("the ribbon displays tabs without permanent command groups")]
#[given("Minimize to Tabs is the selected ribbon display mode")]
fn tabs_only_mode(world: &mut ApplicationWorld) {
    choose_tabs_mode(world);
}
#[when("the pointer rests over the ribbon display control")]
fn rest_on_control(world: &mut ApplicationWorld) {
    let at = world
        .run("Ribbon Display")
        .center()
        .map(|value| value as i32);
    world.hover(at);
    world.pump(Vec::new());
    world.pump(Vec::new());
}
#[then("a tooltip identifies the action as Show Full Ribbon")]
fn full_ribbon_tooltip(world: &mut ApplicationWorld) {
    world.expect_text("Show Full Ribbon");
}
#[then("the part viewport remains visible")]
fn viewport_visible(world: &mut ApplicationWorld) {
    world.expect_blank_region([400, 1400], [250, 650]);
}
#[given("the 3D Model ribbon is expanded")]
fn model_ribbon_expanded(world: &mut ApplicationWorld) {
    // After Minimize to Tabs, the control expands it; a full ribbon already is.
    if !world.shows("Start 2D Sketch") {
        activate_display_control(world);
    }
    world.expect_text("Start 2D Sketch");
}
#[then("the permanent command groups disappear and the ribbon tabs remain visible")]
fn groups_gone(world: &mut ApplicationWorld) {
    world.expect_no_text("Start 2D Sketch");
    world.expect_text("3D Model");
}
#[then("the full 3D Model command groups are visible")]
#[then("the full 3D Model ribbon is visible")]
#[given("the full 3D Model ribbon is visible")]
fn model_groups_visible(world: &mut ApplicationWorld) {
    world.expect_text("Start 2D Sketch");
    world.expect_text("Extrude");
}
#[then("the ribbon returns to tabs-only display")]
fn back_to_tabs_only(world: &mut ApplicationWorld) {
    groups_gone(world);
}

#[given("a part is active in the MCAD Software with the full ribbon visible")]
fn part_with_full_ribbon(world: &mut ApplicationWorld) {
    use crate::HeadWork;
    world.app().owner.create_part_document().unwrap();
    world.previous_document = world.app().owner.gui_command_state().unwrap().document;
    world.pump(Vec::new());
    world.expect_text("Extrude");
}
#[given("the 3D Model tab is the first visible command tab and is active")]
fn model_is_first(world: &mut ApplicationWorld) {
    world.expect_text("3D Model");
    world.expect_text("Extrude");
}
fn scroll_tab(world: &mut ApplicationWorld, label: &str, notches: i32) {
    use oblikovati_viewport::{ViewportEvent, ViewportInput};
    let position = super::diagnostics::pointer_pixels(world.run(label).center());
    world.hover(position);
    world.pump(vec![ViewportEvent::Input(ViewportInput::Wheel {
        notches,
        position,
    })]);
}
#[when("the user scrolls up over the 3D Model tab")]
fn wheel_up_on_model(world: &mut ApplicationWorld) {
    scroll_tab(world, "3D Model", 1);
}
#[then("the 3D Model tab remains active")]
fn model_remains(world: &mut ApplicationWorld) {
    world.expect_text("Extrude");
}
#[when("the user selects the last visible command tab")]
fn select_last_tab(world: &mut ApplicationWorld) {
    world.click_text("Annotate");
}
#[when("the user scrolls down over that tab")]
fn wheel_down_on_last(world: &mut ApplicationWorld) {
    scroll_tab(world, "Annotate", -1);
}
#[then("the last visible command tab remains active")]
fn last_remains(world: &mut ApplicationWorld) {
    world.expect_text("General Note");
}
#[then("its command groups remain visible")]
fn current_groups_remain(world: &mut ApplicationWorld) {
    assert!(
        world.shows("Extrude") || world.shows("General Note"),
        "expected current tab commands"
    );
}

#[test]
fn mouse_wheel_selects_adjacent_ribbon_tabs() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    let camera = crate::HeadWindow::view_camera(&world.app().window).unwrap();
    scroll_tab(&mut world, "3D Model", -1);
    world.expect_text("Application Options");
    world.expect_no_text("Extrude");
    scroll_tab(&mut world, "Tools", -1);
    world.expect_text("Parameters");
    scroll_tab(&mut world, "Manage", 1);
    world.expect_text("Application Options");
    scroll_tab(&mut world, "Tools", 1);
    world.expect_text("Extrude");
    assert_eq!(
        crate::HeadWindow::view_camera(&world.app().window).unwrap(),
        camera
    );
    world.close();
}

#[test]
fn mouse_wheel_keeps_tabs_only_closed_until_a_tab_is_clicked() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    choose_tabs_mode(&mut world);
    scroll_tab(&mut world, "3D Model", -1);
    world.expect_no_text("Extrude");
    assert_eq!(
        world
            .frame
            .runs
            .iter()
            .filter(|run| run.text == "Parameters")
            .count(),
        1
    );
    world.click_text("Tools");
    world.expect_text("Application Options");
    world.close();
}

#[test]
fn mouse_wheel_switches_an_open_temporary_ribbon() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    choose_tabs_mode(&mut world);
    world.click_text("3D Model");
    world.expect_text("Extrude");
    scroll_tab(&mut world, "3D Model", -1);
    world.expect_no_text("Extrude");
    world.expect_text("Application Options");
    world.close();
}

#[given("the full Tools ribbon is visible")]
fn full_tools(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.expect_text("Application Options");
}
#[when("the user scrolls down over the Tools tab")]
fn wheel_down_on_tools(world: &mut ApplicationWorld) {
    scroll_tab(world, "Tools", -1);
}
#[then("Manage becomes the active tab")]
#[then("the full Manage command groups are visible")]
fn manage_active(world: &mut ApplicationWorld) {
    assert!(
        world
            .frame
            .runs
            .iter()
            .filter(|run| run.text == "Parameters")
            .count()
            > 1
    );
}
#[when("the user scrolls up over the same position on the tab row")]
fn wheel_up_over_same_tab_position(world: &mut ApplicationWorld) {
    scroll_tab(world, "Tools", 1);
}
#[then("Tools becomes the active tab")]
#[then("the full Tools command groups are visible")]
fn tools_active(world: &mut ApplicationWorld) {
    world.expect_text("Application Options");
    world.expect_text("Document Settings");
}

#[test]
fn tools_ribbon_opens_application_options() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    world.click_text("Application Options");
    world.expect_text("Text size");
    world.key(oblikovati_viewport::ViewportPhysicalKey::Escape, false);
    world.expect_no_text("Text size");
    world.close();
}

#[test]
fn tools_document_settings_edit_the_active_part_units() {
    use crate::HeadWork;
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    let units = world
        .app()
        .owner
        .gui_command_state()
        .unwrap()
        .document_units
        .unwrap();
    let selected = if units.length_unit == "in" {
        "mm"
    } else {
        "in"
    };
    world.click_text("Document Settings");
    world.expect_text("Length precision");
    world.click_text(&units.length_unit);
    world.click_text(selected);
    assert_eq!(
        world
            .app()
            .owner
            .gui_command_state()
            .unwrap()
            .document_units
            .unwrap()
            .length_unit,
        selected
    );
    let angle = if units.angle_unit == "rad" {
        "deg"
    } else {
        "rad"
    };
    world.click_text(&units.angle_unit);
    world.click_text(angle);
    let mass = if units.mass_unit == "lb" { "kg" } else { "lb" };
    world.click_text(&units.mass_unit);
    world.click_text(mass);
    let time = if units.time_unit == "min" { "s" } else { "min" };
    world.click_text(&units.time_unit);
    world.click_text(time);
    let format = if units.length_display_format == "fractional" {
        "decimal"
    } else {
        "fractional"
    };
    world.click_text(&units.length_display_format);
    world.click_text(format);
    let updated = world
        .app()
        .owner
        .gui_command_state()
        .unwrap()
        .document_units
        .unwrap();
    assert_eq!(
        (
            updated.angle_unit.as_str(),
            updated.mass_unit.as_str(),
            updated.time_unit.as_str(),
            updated.length_display_format.as_str()
        ),
        (angle, mass, time, format)
    );
    world.click_text("Done");
    world.expect_no_text("Length precision");
    world.close();
}

#[test]
fn compact_tools_closes_its_panel_when_document_settings_opens() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    click_display_menu(&mut world);
    world.click_text("Minimize to Panel Buttons");
    world.click_text("Options");
    world.expect_text("Document Settings");
    let position = super::diagnostics::pointer_pixels(world.run("Document Settings").center());
    world.hover(position);
    let button = |pressed| {
        oblikovati_viewport::ViewportEvent::Input(oblikovati_viewport::ViewportInput::Button {
            button: oblikovati_viewport::ViewportButton::Left,
            pressed,
            position,
            modifiers: Default::default(),
        })
    };
    world.pump(vec![button(true)]);
    world.expect_text("Document Settings");
    world.pump(vec![button(false)]);
    world.expect_text("Length precision");
    world.click_text("Done");
    world.expect_no_text("Length precision");
    world.expect_no_text("Document Settings");
    world.close();
}

#[test]
fn tabs_only_tools_closes_temporary_commands_after_opening_settings() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    choose_tabs_mode(&mut world);
    world.click_text("Tools");
    world.expect_text("Document Settings");
    world.click_text("Document Settings");
    world.expect_text("Length precision");
    world.expect_no_text("Application Options");
    world.click_text("Done");
    world.expect_no_text("Length precision");
    world.expect_no_text("Document Settings");
    world.close();
}

#[test]
fn widening_an_open_width_collapsed_panel_closes_its_popup() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    world.app().window.native.view.resize([700, 600]).unwrap();
    world.pump(Vec::new());
    let modify = world.run("Modify").center();
    world.click(super::diagnostics::pointer_pixels([
        modify[0],
        modify[1] - 35.,
    ]));
    world.expect_text("Hole");
    world.app().window.native.view.resize([1600, 900]).unwrap();
    world.pump(Vec::new());
    world.expect_text("Hole");
    world.app().window.native.view.resize([700, 600]).unwrap();
    world.pump(Vec::new());
    world.expect_no_text("Hole");
    world.close();
}

#[test]
fn compact_options_stays_highlighted_when_clicked_again() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    click_display_menu(&mut world);
    world.click_text("Minimize to Panel Buttons");
    let label = world.run("Options").clone();
    let sample = [label.minimum[0] as usize - 4, label.center()[1] as usize];
    world.hover([1000, 600]);
    let idle = world.pixel_at(sample);
    world.click_text("Options");
    world.hover([1000, 600]);
    world.expect_text("Document Settings");
    let selected = world.pixel_at(sample);
    assert_ne!(idle, selected, "open Options button: expected a highlight");
    world.click(super::diagnostics::pointer_pixels(
        compact_options::options_button(&world).center(),
    ));
    world.hover([1000, 600]);
    world.expect_text("Document Settings");
    assert_eq!(world.pixel_at(sample), selected);
    world.close();
}

#[test]
fn document_settings_is_unavailable_while_home_is_in_front() {
    let mut world = ApplicationWorld::new();
    part_with_full_ribbon(&mut world);
    full_tools(&mut world);
    world.click_text("Home");
    world.click_text("Document Settings");
    world.expect_no_text("Length precision");
    world.click_tab(world.previous_document.unwrap());
    world.click_text("Document Settings");
    world.expect_text("Length precision");
    world.close();
}
