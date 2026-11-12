// SPDX-License-Identifier: GPL-2.0-only
use super::{ApplicationWorld, diagnostics::window_geometry};
use cucumber::{given, then, when};
const COLUMN: &str = "##ribbon-column";
const FLOATING: &str = "Ribbon";
const SIDE_WIDTH: f32 = 212.;

/// A point on the ribbon with no control under it, wherever the ribbon sits now.
fn ribbon_blank_point() -> [i32; 2] {
    if let Some((position, _)) = window_geometry(FLOATING) {
        return [position[0] as i32 + 195, position[1] as i32 + 55];
    }
    if let Some((position, size)) = window_geometry(COLUMN) {
        return [position[0] as i32 + 30, (position[1] + size[1]) as i32 - 20];
    }
    [800, 120]
}
pub(super) fn open_context_menu(world: &mut ApplicationWorld) {
    let at = ribbon_blank_point();
    world.right_click(at);
    world.pump(Vec::new());
}
fn open_docking_positions(world: &mut ApplicationWorld) {
    world.click_text("Docking Positions");
    world.pump(Vec::new());
}
/// Each Escape closes one menu level. The pointer leaves first: resting on Docking Positions, it
/// would reopen the submenu that Escape just closed, as any hover-opened menu does.
fn close_ribbon_menus(world: &mut ApplicationWorld) {
    world.hover([700, 500]);
    for _ in 0..3 {
        if !world.shows("Undock Ribbon") {
            return;
        }
        world.escape();
        world.pump(Vec::new());
    }
}
fn choose_position(world: &mut ApplicationWorld, position: &str) {
    close_ribbon_menus(world);
    open_context_menu(world);
    open_docking_positions(world);
    world.click_text(position);
    world.pump(Vec::new());
}
pub(super) fn undock_ribbon(world: &mut ApplicationWorld) {
    close_ribbon_menus(world);
    open_context_menu(world);
    world.click_text("Undock Ribbon");
    world.pump(Vec::new());
}
fn expect_top_dock(world: &mut ApplicationWorld) {
    world.expect_text("Ribbon Display");
    world.expect_text("Start 2D Sketch");
    assert!(window_geometry(COLUMN).is_none(), "expected no side column");
    assert!(
        window_geometry(FLOATING).is_none(),
        "expected no floating ribbon"
    );
}
fn expect_vertical(world: &mut ApplicationWorld, labels: &[&str]) {
    let rows: Vec<[f32; 2]> = labels
        .iter()
        .map(|label| world.run(label).center())
        .collect();
    for pair in rows.windows(2) {
        assert!(
            pair[1][1] > pair[0][1],
            "rows {rows:?}: expected each below the last"
        );
        assert!(
            (pair[1][0] - pair[0][0]).abs() < 40.,
            "rows {rows:?}: expected one column"
        );
    }
}
#[given("a part is active in the MCAD Software")]
fn part_active(world: &mut ApplicationWorld) {
    world.previous_document = Some(world.add_part());
}
#[given("the ribbon is docked at the top with the 3D Model tab selected")]
fn docked_top_on_model_tab(world: &mut ApplicationWorld) {
    world.click_text("3D Model");
    world.pump(Vec::new());
    expect_top_dock(world);
}
#[given("the ribbon is docked at the top")]
fn docked_top(world: &mut ApplicationWorld) {
    expect_top_dock(world);
}
#[when("I open the ribbon context menu")]
#[when("I open the floating ribbon context menu")]
fn context_menu(world: &mut ApplicationWorld) {
    open_context_menu(world);
}
#[when("I open Docking Positions")]
fn docking_positions(world: &mut ApplicationWorld) {
    open_docking_positions(world);
}
#[then("the menu offers Top, Left and Right")]
fn positions_offered(world: &mut ApplicationWorld) {
    for position in ["Top", "Left", "Right"] {
        world.expect_text(position);
    }
}
fn expect_checked(world: &mut ApplicationWorld, position: &str) {
    assert!(
        world.checked(position),
        "{position}: expected the active position checked"
    );
}
#[then("Top is checked")]
fn top_checked(world: &mut ApplicationWorld) {
    expect_checked(world, "Top");
}
#[then("Left is checked")]
fn left_checked(world: &mut ApplicationWorld) {
    expect_checked(world, "Left");
}
#[then("Right is checked")]
fn right_checked(world: &mut ApplicationWorld) {
    expect_checked(world, "Right");
}
#[given("the model browser is docked on the left")]
#[then("the model browser is visible on the left of the viewport")]
fn browser_left_side(world: &mut ApplicationWorld) {
    let (position, _) = window_geometry("Model").expect("expected the browser");
    assert!(
        position[0] < 40.,
        "browser at {position:?}: expected the left edge"
    );
    world.expect_text("Origin");
}
#[when("I choose Left from the ribbon Docking Positions menu")]
fn choose_left(world: &mut ApplicationWorld) {
    choose_position(world, "Left");
}
#[when("I choose Right from the ribbon Docking Positions menu")]
fn choose_right(world: &mut ApplicationWorld) {
    choose_position(world, "Right");
}
#[when("I choose Top from the ribbon Docking Positions menu")]
fn choose_top(world: &mut ApplicationWorld) {
    choose_position(world, "Top");
}
#[when("I open the ribbon Docking Positions menu")]
fn open_positions_menu(world: &mut ApplicationWorld) {
    close_ribbon_menus(world);
    open_context_menu(world);
    open_docking_positions(world);
}
#[then("the ribbon tabs are arranged vertically along the left edge")]
fn tabs_on_left(world: &mut ApplicationWorld) {
    expect_vertical(world, &["3D Model", "Manage", "View"]);
    assert!(
        world.run("3D Model").center()[0] < 90.,
        "expected tabs at the left edge"
    );
}
#[then("the 3D Model command groups are stacked beside the tabs")]
fn groups_beside_tabs(world: &mut ApplicationWorld) {
    expect_vertical(world, &["Sketch", "Create", "Modify"]);
    let groups = world.run("Create").center()[0];
    assert!(
        groups > world.run("3D Model").center()[0] + 40.,
        "expected groups beside the tabs"
    );
}
#[then("the model browser appears to the right of the ribbon")]
fn browser_right_of_ribbon(_world: &mut ApplicationWorld) {
    let (position, _) = window_geometry("Model").expect("expected the browser");
    assert!(
        position[0] >= SIDE_WIDTH,
        "browser at {position:?}: expected past the ribbon"
    );
}
#[then("the ribbon tabs and command groups are arranged vertically on the right")]
fn ribbon_on_right(world: &mut ApplicationWorld) {
    expect_vertical(world, &["3D Model", "Manage", "View"]);
    expect_vertical(world, &["Sketch", "Create", "Modify"]);
    let edge = world.extent()[0] as f32 - SIDE_WIDTH;
    assert!(
        world.run("3D Model").center()[0] > edge,
        "expected tabs at the right edge"
    );
}
#[then("the ribbon tabs and command groups are arranged horizontally above the viewport")]
#[then("the ribbon returns to a horizontal position above the viewport")]
fn ribbon_on_top(world: &mut ApplicationWorld) {
    expect_top_dock(world);
}
#[when("I choose Undock Ribbon")]
fn choose_undock(world: &mut ApplicationWorld) {
    world.click_text("Undock Ribbon");
    world.pump(Vec::new());
}
fn expect_floating(_world: &mut ApplicationWorld) {
    let (_, size) = window_geometry(FLOATING).expect("expected a floating ribbon");
    assert!(
        size[1] > size[0],
        "ribbon {size:?}: expected a vertical panel"
    );
}
#[then("the ribbon floats over the workspace in a vertical panel")]
#[then("the ribbon remains floating")]
fn ribbon_floats(world: &mut ApplicationWorld) {
    expect_floating(world);
}
#[then("its tabs and 3D Model group buttons remain visible")]
fn floating_contents(world: &mut ApplicationWorld) {
    for shown in ["3D Model", "Sketch", "Create", "Modify"] {
        world.expect_text(shown);
    }
}
#[then("Undock Ribbon is unavailable")]
fn undock_muted(world: &mut ApplicationWorld) {
    let alpha = world.run("Undock Ribbon").alpha;
    assert!(alpha < 200, "Undock Ribbon alpha {alpha}: expected muted");
}
#[when("I choose Top from Docking Positions")]
fn top_from_positions(world: &mut ApplicationWorld) {
    open_docking_positions(world);
    world.click_text("Top");
    world.pump(Vec::new());
}
#[given("the ribbon is floating with the 3D Model tab selected")]
#[given("the floating ribbon shows the 3D Model groups at the top of its scroll area")]
fn floating_on_model(world: &mut ApplicationWorld) {
    undock_ribbon(world);
    expect_floating(world);
    world.expect_text("3D Model");
}
fn expect_beside_ribbon(world: &mut ApplicationWorld, command: &str) {
    let (position, size) = window_geometry(FLOATING)
        .or_else(|| window_geometry(COLUMN))
        .expect("expected a side or floating ribbon");
    let x = world.run(command).center()[0];
    assert!(
        x > position[0] + size[0],
        "{command} at {x}: expected beside a ribbon ending at {}",
        position[0] + size[0]
    );
}
#[then("a temporary panel opens beside the ribbon")]
#[then("a temporary Create command panel opens beside the ribbon")]
fn panel_beside(world: &mut ApplicationWorld) {
    expect_beside_ribbon(world, "Extrude");
}
#[then("the panel displays Create commands including Extrude and Revolve")]
#[then("the panel includes Extrude and Revolve")]
fn create_commands(world: &mut ApplicationWorld) {
    world.expect_text("Extrude");
    world.expect_text("Revolve");
}
#[when("I click an empty area of the viewport outside the panel")]
fn click_outside_panel(world: &mut ApplicationWorld) {
    world.click([1000, 600]);
    world.pump(Vec::new());
}
#[then("the temporary panel closes")]
fn panel_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Extrude");
    // The browser's own rows hold these words too, so only a whole row counts as the panel's.
    let commands = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Plane")
        .count();
    assert_eq!(commands, 0, "expected the Work Features commands gone");
}
#[given("the ribbon is docked on the left with the 3D Model tab selected")]
#[given("the 3D Model ribbon is docked on the left with Modify as a compact group")]
#[given("the 3D Model ribbon is docked on the left")]
pub(super) fn docked_left_on_model(world: &mut ApplicationWorld) {
    choose_position(world, "Left");
    world.expect_text("3D Model");
}
#[then("a temporary Modify command panel opens beside the ribbon")]
fn modify_beside(world: &mut ApplicationWorld) {
    expect_beside_ribbon(world, "Fillet");
}
#[then("the panel includes Hole and Fillet")]
fn modify_commands(world: &mut ApplicationWorld) {
    world.expect_text("Hole");
    world.expect_text("Fillet");
}
#[then("the Modify panel closes")]
fn modify_closed(world: &mut ApplicationWorld) {
    world.expect_no_text("Fillet");
}
#[then("the ribbon remains docked on the left")]
fn remains_left(world: &mut ApplicationWorld) {
    let (position, _) = window_geometry(COLUMN).expect("expected the side column");
    assert_eq!(
        position[0], 0.,
        "column at {position:?}: expected the left edge"
    );
    world.expect_text("Save");
}
fn work_features_y(world: &ApplicationWorld) -> f32 {
    world.run("Work Features").center()[1]
}
#[given("the Work Features group is partially visible at the bottom")]
fn work_features_partial(world: &mut ApplicationWorld) {
    let (position, size) = window_geometry(FLOATING).expect("expected a floating ribbon");
    let bottom = position[1] + size[1];
    let label = work_features_y(world);
    // A centred label this near the edge means its 48-pixel button crosses the edge.
    assert!(
        bottom - label < 32.,
        "label at {label}: expected near the bottom {bottom}"
    );
    world.ribbon_height = label;
}
#[when("I scroll down in the floating ribbon group area")]
fn scroll_groups(world: &mut ApplicationWorld) {
    use oblikovati_viewport::{ViewportEvent, ViewportInput};
    let at = world.run("Create").center().map(|value| value as i32);
    world.hover(at);
    world.pump(vec![ViewportEvent::Input(ViewportInput::Wheel {
        notches: -2,
        position: at,
    })]);
    world.pump(Vec::new());
}
#[then("the Work Features group becomes fully visible")]
fn work_features_visible(world: &mut ApplicationWorld) {
    let (position, size) = window_geometry(FLOATING).expect("expected a floating ribbon");
    let label = work_features_y(world);
    assert!(
        label < world.ribbon_height,
        "label at {label}: expected it to move up"
    );
    assert!(
        position[1] + size[1] - label > 24.,
        "label at {label}: expected clear of the edge"
    );
}
#[when("I select the Work Features group")]
fn select_work_features(world: &mut ApplicationWorld) {
    world.click_text("Work Features");
    world.pump(Vec::new());
}
#[then("a temporary panel shows Plane, Axis and Point commands")]
fn work_commands(world: &mut ApplicationWorld) {
    for command in ["Plane", "Axis", "Point"] {
        world.expect_text(command);
    }
}
#[then("the same part remains active")]
fn same_part(world: &mut ApplicationWorld) {
    let id = world
        .previous_document
        .or(world.parts.last().copied())
        .expect("expected a remembered part");
    assert_eq!(world.active_id(), id);
}
#[when("I select the Annotate tab on the floating ribbon")]
fn floating_annotate(world: &mut ApplicationWorld) {
    world.click_text("Annotate");
    world.pump(Vec::new());
}
#[then("the ribbon displays annotation groups including Notes")]
fn notes_group(world: &mut ApplicationWorld) {
    world.expect_text("Notes");
}
#[when("I select the Notes group")]
fn select_notes(world: &mut ApplicationWorld) {
    world.click_text("Notes");
    world.pump(Vec::new());
}
#[then("a temporary panel displays note commands")]
fn note_commands(world: &mut ApplicationWorld) {
    expect_beside_ribbon(world, "General Note");
    world.expect_text("Leader Note");
}
#[then("the Notes panel closes")]
fn notes_closed(world: &mut ApplicationWorld) {
    world.pump(Vec::new());
    world.expect_no_text("General Note");
}
#[then("the floating ribbon displays its 3D Model groups again")]
fn model_groups_again(world: &mut ApplicationWorld) {
    expect_floating(world);
    for group in ["Sketch", "Create", "Modify"] {
        world.expect_text(group);
    }
}
#[given("a part, an assembly and a sheet-metal part are open in the MCAD Software")]
fn three_documents(world: &mut ApplicationWorld) {
    use oblikovati_gui::GuiTemplateKind;
    world.previous_document = Some(world.add_part());
    world.other_document = Some(world.add_template(GuiTemplateKind::Assembly));
    let sheet_metal = world.add_template(GuiTemplateKind::SheetMetal);
    world.parts.push(sheet_metal);
}
#[given("the part is active with the ribbon docked on the left")]
fn part_docked_left(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected the part");
    world.activate_through_tab(part);
    choose_position(world, "Left");
    let (position, _) = window_geometry(COLUMN).expect("expected the side column");
    assert_eq!(position[0], 0.);
}
#[when("I activate the assembly document")]
fn activate_assembly(world: &mut ApplicationWorld) {
    let assembly = world.other_document.expect("expected the assembly");
    world.click_tab(assembly);
    world.pump(Vec::new());
}
#[then("it displays assembly tabs and commands")]
fn assembly_tabs(world: &mut ApplicationWorld) {
    world.expect_text("Assemble");
    world.expect_text("Component");
}
#[when("I activate the sheet-metal part")]
fn activate_sheet_metal(world: &mut ApplicationWorld) {
    let sheet_metal = *world.parts.last().expect("expected the sheet-metal part");
    world.click_tab(sheet_metal);
    world.pump(Vec::new());
}
#[then("it displays the Sheet Metal tab and its commands")]
fn sheet_metal_tabs(world: &mut ApplicationWorld) {
    world.expect_text("Sheet Metal");
    world.expect_text("Sketch");
}
#[then("the sheet-metal part is active")]
fn sheet_metal_active(world: &mut ApplicationWorld) {
    let sheet_metal = *world.parts.last().expect("expected the sheet-metal part");
    assert_eq!(world.state().document, Some(sheet_metal));
}
pub(super) fn dock_ribbon_top(world: &mut ApplicationWorld) {
    choose_position(world, "Top");
}

#[path = "active_tab.rs"]
mod active_tab;

#[path = "right_docked.rs"]
mod right_docked;

#[path = "floating_title.rs"]
mod floating_title;
