// SPDX-License-Identifier: GPL-2.0-only
//! The dock's other panes, and the dock itself.
use super::*;
#[cfg(target_os = "linux")]
#[path = "panes_native.rs"]
mod native;

const FAVORITES: &str = "Favorites";
const TREE_VIEW: &str = "Tree View";
const THUMBNAIL_VIEW: &str = "Thumbnail View";
const DOCK_TARGET_INSET: f32 = 15.;
const MAX_SPLITTER_GAP: f32 = 12.;
/// What a floating or docked browser's tree shows, whatever else is on screen.
const TREE_ROW: &str = "Center Point";

fn tab_bar_of(world: &ApplicationWorld, tab: &str) -> [f32; 2] {
    world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == tab)
        .max_by(|first, second| first.minimum[1].total_cmp(&second.minimum[1]))
        .unwrap_or_else(|| panic!("expected the {tab} tab in {:?}", world.text()))
        .center()
}
fn click_tab(world: &mut ApplicationWorld, tab: &str) {
    let at = tab_bar_of(world, tab);
    world.click(at.map(|value| value as i32));
    settle(world);
}
fn open_favorites(world: &mut ApplicationWorld) {
    world.click_text("+");
    settle(world);
    choose_menu_row(world, FAVORITES);
    settle(world);
}
/// A tab shows its close mark at its right end while the pointer is over it.
fn close_favorites(world: &mut ApplicationWorld) {
    let at = tab_bar_of(world, FAVORITES).map(|value| value as i32);
    world.hover(at);
    settle(world);
    let run = world.run(FAVORITES).clone();
    world.click([run.maximum[0] as i32 + 10, run.center()[1] as i32]);
    settle(world);
}
#[given("the Model browser tab is active")]
fn model_active(world: &mut ApplicationWorld) {
    world.expect_text(TREE_ROW);
}
#[when("I open the add-pane menu in the browser header")]
#[when("the user opens the add-pane menu in the browser header")]
fn open_add_menu(world: &mut ApplicationWorld) {
    world.click_text("+");
    settle(world);
}
#[when("I choose Favorites")]
#[when("the user chooses Favorites")]
fn choose_favorites(world: &mut ApplicationWorld) {
    choose_menu_row(world, FAVORITES);
    settle(world);
}
#[then("a Favorites tab appears beside the Model tab and becomes active")]
fn favorites_active(world: &mut ApplicationWorld) {
    world.expect_text(FAVORITES);
    world.expect_no_text(TREE_ROW);
}
#[then("the Favorites pane shows its content controls")]
fn favorites_controls(world: &mut ApplicationWorld) {
    world.expect_text(TREE_VIEW);
    world.expect_text("No favorites yet");
}
#[when("I select the Model tab")]
#[when("the user selects the Model tab")]
fn select_model(world: &mut ApplicationWorld) {
    click_tab(world, "Model");
}
#[then("the model hierarchy is visible again")]
fn tree_again(world: &mut ApplicationWorld) {
    world.expect_text(TREE_ROW);
}
#[when("I select Favorites and close its tab")]
#[when("the user selects Favorites and closes its tab")]
fn select_and_close(world: &mut ApplicationWorld) {
    click_tab(world, FAVORITES);
    close_favorites(world);
}
#[then("the Favorites tab disappears and the Model hierarchy is visible")]
fn favorites_gone(world: &mut ApplicationWorld) {
    world.expect_no_text(FAVORITES);
    world.expect_text(TREE_ROW);
}
#[given("the Favorites pane is open with Tree View selected")]
fn favorites_tree(world: &mut ApplicationWorld) {
    open_favorites(world);
    world.expect_text(TREE_VIEW);
}
#[when("I open the Favorites view-mode menu")]
#[when("the user opens the Favorites view-mode menu")]
#[when("I reopen the view-mode menu")]
#[when("the user reopens the view-mode menu")]
fn open_view_menu(world: &mut ApplicationWorld) {
    let shown = if world.shows(THUMBNAIL_VIEW) {
        THUMBNAIL_VIEW
    } else {
        TREE_VIEW
    };
    world.click_text(shown);
    settle(world);
}
#[then("Tree View is checked and Thumbnail View is available")]
fn tree_checked(world: &mut ApplicationWorld) {
    world.expect_text(THUMBNAIL_VIEW);
    let row = world
        .frame
        .runs
        .iter()
        .rfind(|run| run.text.replace(' ', "") == TREE_VIEW.replace(' ', ""))
        .cloned()
        .expect("expected the menu row");
    let marked = world.frame.marks.iter().any(|mark| {
        mark.bright && mark.list == row.list && (mark.position[1] - row.center()[1]).abs() < 10.
    });
    assert!(marked, "expected {TREE_VIEW} checked");
}
#[when("I choose Thumbnail View")]
#[when("the user chooses Thumbnail View")]
fn choose_thumbnails(world: &mut ApplicationWorld) {
    world.click_text(THUMBNAIL_VIEW);
    settle(world);
}
#[then("the view-mode control displays Thumbnail View")]
fn shows_thumbnails(world: &mut ApplicationWorld) {
    world.expect_text(THUMBNAIL_VIEW);
}
#[then("the pane shows its content-location control")]
fn location_control(world: &mut ApplicationWorld) {
    world.expect_text("Content Location");
}
#[then("Thumbnail View is checked")]
fn thumbnails_checked(world: &mut ApplicationWorld) {
    assert!(
        world.checked(THUMBNAIL_VIEW),
        "expected {THUMBNAIL_VIEW} checked"
    );
}
#[when("I choose Tree View")]
#[when("the user chooses Tree View")]
fn choose_tree(world: &mut ApplicationWorld) {
    world.click_text(TREE_VIEW);
    settle(world);
}
#[then("the view-mode control displays Tree View")]
fn shows_tree(world: &mut ApplicationWorld) {
    world.expect_text(TREE_VIEW);
}
#[when("I close the Favorites tab")]
#[when("the user closes the Favorites tab")]
fn close_tab(world: &mut ApplicationWorld) {
    close_favorites(world);
}
#[then("the Model browser is displayed again")]
fn model_again(world: &mut ApplicationWorld) {
    world.expect_text(TREE_ROW);
}
// The dock
fn browser_geometry() -> ([f32; 2], [f32; 2]) {
    diagnostics::window_geometry("Model").expect("expected the browser")
}
fn drag_model_tab(world: &mut ApplicationWorld, to: [i32; 2]) {
    let at = diagnostics::pointer_pixels(tab_bar_of(world, "Model"));
    world.drag(at, to);
    settle(world);
}
fn divider() -> [i32; 2] {
    let (position, size) = browser_geometry();
    diagnostics::pointer_pixels([position[0] + size[0] + 3., position[1] + size[1] / 2.])
}
fn expect_left_docked(world: &mut ApplicationWorld) {
    expect_dock_adjacent(world, false);
}
fn expect_dock_adjacent(world: &mut ApplicationWorld, right_side: bool) {
    let (position, size) = browser_geometry();
    let viewport = world.viewport_bounds();
    let dock = diagnostics::window_dock_id("Model").expect("expected a visible model browser");
    assert_ne!(dock, 0, "expected a docked browser");
    let left = diagnostics::pointer_pixels(position)[0];
    let right = diagnostics::pointer_pixels([position[0] + size[0], position[1]])[0];
    let gap = if right_side {
        left - viewport[2] as i32
    } else {
        viewport[0] as i32 - right
    };
    let max_gap = diagnostics::pointer_pixels([MAX_SPLITTER_GAP, 0.])[0];
    assert!(
        (-1..=max_gap).contains(&gap),
        "browser {position:?} {size:?}, viewport {viewport:?}: expected adjacent dock"
    );
}
fn edge_target(world: &mut ApplicationWorld, right: bool) -> [i32; 2] {
    let area = world.viewport_bounds();
    let inset = diagnostics::pointer_pixels([DOCK_TARGET_INSET, 0.])[0] as u32;
    assert!(
        area[2] > area[0] + inset * 2,
        "viewport {area:?}: expected space for docking targets"
    );
    let x = if right {
        area[2] - inset
    } else {
        area[0] + inset
    };
    [x as i32, ((area[1] + area[3]) / 2) as i32]
}
fn remember_viewport(world: &mut ApplicationWorld) {
    let area = world.viewport_bounds();
    world.docking_resize = Some((area, 0.));
}
fn expect_viewport_change(world: &mut ApplicationWorld, narrowed: bool) {
    let before = world
        .docking_resize
        .expect("expected the previous viewport")
        .0;
    let after = world.viewport_bounds();
    let changed = if narrowed {
        after[0] > before[0]
    } else {
        after[0] < before[0]
    };
    assert!(
        changed && after[2] == before[2],
        "viewport {before:?} -> {after:?}"
    );
}
#[given("a part is open in the MCAD Software with the model browser docked at the left")]
#[given("a part is open in the MCAD Software with the model browser docked on the left")]
fn docked_part(world: &mut ApplicationWorld) {
    open_block(world);
    expect_left_docked(world);
}
#[given("the browser tree and a shaded block are visible")]
#[given("the browser tree and shaded block are visible")]
#[then("the tree and block remain visible")]
#[then("the tree and shaded block remain visible")]
fn tree_and_block(world: &mut ApplicationWorld) {
    world.expect_text(TREE_ROW);
    expect_solid_visible(world);
}
#[when("I drag the divider between the browser and viewport to the right")]
#[when("the user drags the divider between the browser and viewport to the right")]
fn divider_right(world: &mut ApplicationWorld) {
    world.observed_row_gap = Some(browser_geometry().1[0]);
    remember_viewport(world);
    let from = divider();
    let delta = (world.extent()[0] / 12) as i32;
    world.drag(from, [from[0] + delta, from[1]]);
    settle(world);
}
#[then("the browser becomes wider and the viewport becomes narrower")]
fn browser_wider(world: &mut ApplicationWorld) {
    let earlier = world.observed_row_gap.expect("expected the earlier width");
    assert!(
        browser_geometry().1[0] > earlier,
        "browser width from {earlier}"
    );
    expect_viewport_change(world, true);
}
#[then("the tree hierarchy remains visible")]
#[then("its tree remains visible")]
#[then("the browser tree remains available in the floating panel")]
fn tree_stays(world: &mut ApplicationWorld) {
    world.expect_text(TREE_ROW);
}
#[then("the block remains visible with the same apparent size and orientation")]
#[then("the shaded block remains visible with the same apparent orientation")]
fn block_kept(world: &mut ApplicationWorld) {
    expect_block_kept(world);
}
#[when("I drag the divider back to the left")]
#[when("the user drags the divider back to the left")]
fn divider_left(world: &mut ApplicationWorld) {
    world.observed_row_gap = Some(browser_geometry().1[0]);
    remember_viewport(world);
    let from = divider();
    let delta = (world.extent()[0] / 12) as i32;
    world.drag(from, [from[0] - delta, from[1]]);
    settle(world);
}
#[then("the browser becomes narrower and the viewport becomes wider")]
fn browser_narrower(world: &mut ApplicationWorld) {
    let earlier = world.observed_row_gap.expect("expected the earlier width");
    assert!(
        browser_geometry().1[0] < earlier,
        "browser width from {earlier}"
    );
    expect_viewport_change(world, false);
}
#[when("I drag the Model tab into the viewport and release it")]
#[when("the user drags the Model tab into the viewport and release it")]
fn tab_into_view(world: &mut ApplicationWorld) {
    let area = world.viewport_bounds();
    let center = [
        ((area[0] + area[2]) / 2) as i32,
        ((area[1] + area[3]) / 2) as i32,
    ];
    drag_model_tab(world, center);
}
#[then("the browser becomes a floating panel over the viewport")]
fn browser_floats(_: &mut ApplicationWorld) {
    let (position, _) = browser_geometry();
    assert!(position[0] > 100., "browser at {position:?}");
}
#[then("the viewport occupies the space formerly used by the docked panel")]
fn view_spreads(world: &mut ApplicationWorld) {
    super::super::workspace_layout::left_space_free(world);
}
#[when("I drag the floating Model tab to the left edge and release it")]
#[when("the user drags the floating Model tab to the left edge and release it")]
#[when("I drag the Model tab to the left edge and release it")]
#[when("the user drags the Model tab to the left edge and release it")]
fn tab_to_left(world: &mut ApplicationWorld) {
    let target = edge_target(world, false);
    drag_model_tab(world, target);
}
#[then("the browser docks on the left again")]
fn docks_left(world: &mut ApplicationWorld) {
    expect_left_docked(world);
}
#[when("I drag the Model tab to the right edge and release it")]
#[when("the user drags the Model tab to the right edge and release it")]
fn tab_to_right(world: &mut ApplicationWorld) {
    let target = edge_target(world, true);
    drag_model_tab(world, target);
}
#[then("the browser docks on the right of the viewport")]
fn docks_right(world: &mut ApplicationWorld) {
    expect_dock_adjacent(world, true);
}
