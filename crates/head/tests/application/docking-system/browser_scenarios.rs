// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the observed Appearance and Material Browser docking workflows.
use super::super::x11_input;
use super::{ApplicationWorld, diagnostics, native_drag, resize_boundary, viewport};
use cucumber::{given, then, when};
const APPEARANCE: &str = "Appearance Browser";
const MATERIAL: &str = "Material Browser";

#[test]
fn narrow_asset_dock_keeps_both_chooser_labels_visible() {
    let mut world = ApplicationWorld::new();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    check_narrow_asset_selector(&mut world, APPEARANCE, "Appearance");
    check_narrow_asset_selector(&mut world, MATERIAL, "Material");
    world.close();
}

fn check_narrow_asset_selector(world: &mut ApplicationWorld, panel: &str, label: &str) {
    resize_asset_test_window(world, [1440, 900]);
    select_tab(world, panel);
    resize_asset_test_window(world, [780, 580]);
    assert_eq!(diagnostics::window_dock_tab_visible(panel), Some(true));
    let (position, size) = diagnostics::window_geometry(panel).expect("asset dock");
    assert!(
        (160. ..=180.).contains(&size[0]),
        "{panel} width {}",
        size[0]
    );
    world.save_capture(&format!("narrow-{label}-selector"));
    let caption = world.run_in_window(panel, label);
    assert!(
        caption.maximum[0] <= position[0] + size[0] - 4.,
        "{panel} clipped caption at {:?}, dock {position:?} + {size:?}",
        caption.maximum
    );
    let arrow = [caption.minimum[0] - 12., caption.center()[1]];
    open_narrow_asset_choices(world, panel, arrow);
}

fn resize_asset_test_window(world: &mut ApplicationWorld, extent: [u32; 2]) {
    x11_input::resize_application_window(extent);
    for _ in 0..3 {
        world.pump(Vec::new());
    }
    assert_eq!(world.extent(), extent);
}

fn open_narrow_asset_choices(world: &mut ApplicationWorld, panel: &str, arrow: [f32; 2]) {
    let browser = world.state().browser.expect("active Part asset catalog");
    let (assigned, choices) = if panel == APPEARANCE {
        (browser.part_appearance, browser.appearances)
    } else {
        (browser.part_material, browser.materials)
    };
    let preview = choices
        .iter()
        .find(|item| Some(&item.id) == assigned.as_ref())
        .unwrap_or(&choices[0]);
    x11_input::click(diagnostics::pointer_pixels(arrow));
    world.pump(Vec::new());
    let alternative = choices.iter().find(|item| item.id != preview.id).unwrap();
    world.expect_text(&alternative.name);
    world.save_capture(&format!("narrow-{panel}-choices"));
    world.escape();
}

#[test]
fn viewport_keeps_its_reserved_width_while_the_splitter_is_held() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    open_floating(&mut world, APPEARANCE);
    dock_appearance_right(&mut world);
    hold_splitter_past_its_limit(&mut world, false);
    world.close();
}

#[test]
fn held_model_splitter_respects_the_same_viewport_limit() {
    let mut world = ApplicationWorld::new();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    hold_splitter_past_its_limit(&mut world, true);
    world.close();
}

fn hold_splitter_past_its_limit(world: &mut ApplicationWorld, left: bool) {
    let original = viewport(world);
    let height = ((original[1] + original[3]) / 2) as i32;
    let (edge, start, end, capture) = if left {
        (0, original[0] as i32 - 1, original[2] as i32 - 100, "model")
    } else {
        (
            2,
            original[2] as i32 + 2,
            original[0] as i32 + 100,
            "appearance",
        )
    };
    let handle = [start, height];
    let beyond_limit = [end, height];
    x11_input::move_pointer(handle);
    world.pump(Vec::new());
    x11_input::left_button(handle, true);
    world.pump(Vec::new());
    x11_input::move_pointer(beyond_limit);
    world.pump(Vec::new());
    let held = viewport(world);
    let minimum = (world.extent()[0] * 55).div_ceil(100);
    world.save_capture(&format!("{capture}-splitter-held-at-cap"));
    assert!(
        held[2] - held[0] >= minimum,
        "held viewport {held:?}, expected {minimum} px"
    );
    assert!(
        held[edge].abs_diff(original[edge]) > 40,
        "expected the splitter to move: {original:?} -> {held:?}"
    );
    x11_input::move_pointer(handle);
    world.pump(Vec::new());
    let returned = viewport(world);
    x11_input::left_button(handle, false);
    world.pump(Vec::new());
    assert!(
        original[0].abs_diff(returned[0]) <= 1 && original[2].abs_diff(returned[2]) <= 1,
        "expected the splitter to return without releasing it: {original:?} -> {returned:?}"
    );
}

#[test]
fn capped_appearance_splitter_keeps_the_selected_width_across_restart() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    open_floating(&mut world, APPEARANCE);
    dock_appearance_right(&mut world);
    widen_appearance(&mut world);
    appearance_wider(&mut world);
    let widened = viewport(&mut world);
    let extent = world.extent();
    assert!((widened[2] - widened[0]) * 100 >= extent[0] * 55);
    world.save_capture("appearance-splitter-widened-at-cap");
    world.reopen_application();
    super::super::workspace_layout::browser_feature::open_visible_extruded_part(&mut world);
    assert_eq!(viewport(&mut world), widened);
    restore_appearance_boundary(&mut world);
    appearance_proportions_restored(&mut world);
    world.save_capture("appearance-splitter-dragged-back");
    world.close();
}

#[given("Appearance Browser is floating over the workspace")]
fn appearance_floating(world: &mut ApplicationWorld) {
    super::open_docking_fixture(world);
    open_floating(world, APPEARANCE);
}

fn open_floating(world: &mut ApplicationWorld, panel: &str) {
    super::super::workspace_layout::dockspace_native::open_toolbar_tool(world, panel);
    assert_eq!(diagnostics::window_dock_id(panel), Some(0));
}

#[when("the user drags the Appearance Browser title bar to the far right workspace edge")]
fn dock_appearance_right(world: &mut ApplicationWorld) {
    let area = viewport(world);
    let (position, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    let start = [
        (position[0] + size[0] * 0.4) as i32,
        (position[1] + 10.) as i32,
    ];
    native_drag(
        world,
        start,
        [area[2] as i32 - 15, ((area[1] + area[3]) / 2) as i32],
    );
}

#[given("Appearance Browser is docked on the right")]
fn appearance_docked(world: &mut ApplicationWorld) {
    appearance_floating(world);
    dock_appearance_right(world);
    appearance_on_right(world);
}

#[then("Appearance Browser docks along the right edge")]
fn appearance_on_right(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    assert_ne!(diagnostics::window_dock_id(APPEARANCE), Some(0));
    assert!((position[0] + size[0] - world.extent()[0] as f32).abs() <= 1.);
}

#[then("the Model Browser remains visible on the left")]
pub(super) fn model_browser_left(_: &mut ApplicationWorld) {
    let (position, _) = diagnostics::window_geometry("Model").expect("model browser");
    assert!(position[0].abs() <= 1., "model browser at {position:?}");
}

#[then("the viewport occupies the space between both browser panels")]
#[then("the viewport remains visible to the left of the dock region")]
fn viewport_between_browsers(world: &mut ApplicationWorld) {
    let area = viewport(world);
    let (left, left_size) = diagnostics::window_geometry("Model").unwrap();
    let (right, _) = diagnostics::window_geometry(APPEARANCE).unwrap();
    assert!(area[0] as f32 >= left[0] + left_size[0] - 1.);
    assert!(area[2] as f32 <= right[0] + 1. && area[0] < area[2]);
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[then("the active part document remains open")]
fn active_part_remains(world: &mut ApplicationWorld) {
    assert_eq!(world.state().document, world.previous_document);
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[when("the user drags the shared boundary toward the viewport to widen Appearance Browser")]
fn widen_appearance(world: &mut ApplicationWorld) {
    let area = viewport(world);
    let (_, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    world.docking_resize = Some((area, size[0]));
    resize_boundary(world, area, -80);
}

#[then("Appearance Browser becomes wider")]
fn appearance_wider(world: &mut ApplicationWorld) {
    let (_, before) = world.docking_resize.unwrap();
    let (_, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    assert!(size[0] > before + 40., "width {before} -> {}", size[0]);
}

#[then("the view orientation control remains visible in the viewport")]
fn orientation_visible(world: &mut ApplicationWorld) {
    let (position, size) =
        diagnostics::window_geometry("##orientation-control").expect("visible orientation control");
    let area = viewport(world);
    assert!(position[0] >= area[0] as f32 && position[1] >= area[1] as f32);
    assert!(position[0] + size[0] <= area[2] as f32 + 1.);
    assert!(position[1] + size[1] <= area[3] as f32 + 1.);
}

#[when("the user restores the boundary to its earlier position")]
fn restore_appearance_boundary(world: &mut ApplicationWorld) {
    let current = viewport(world);
    let (before, _) = world.docking_resize.unwrap();
    resize_boundary(world, current, before[2] as i32 - current[2] as i32);
}

#[then("the docked panel and viewport return to their earlier proportions")]
fn appearance_proportions_restored(world: &mut ApplicationWorld) {
    let (before, width) = world.docking_resize.unwrap();
    let current = viewport(world);
    let (_, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    assert!(
        before[0].abs_diff(current[0]) <= 1
            && before[2].abs_diff(current[2]) <= 1
            && (width - size[0]).abs() <= 1.,
        "viewport {before:?} -> {current:?}, Appearance width {width} -> {}",
        size[0]
    );
}

#[given("Material Browser is floating over the workspace")]
fn material_floating(world: &mut ApplicationWorld) {
    open_floating(world, MATERIAL);
}

#[when("the user drags the Material Browser title bar onto the Appearance Browser tab strip")]
fn join_browser_tabs(world: &mut ApplicationWorld) {
    let (source, size) = diagnostics::window_geometry(MATERIAL).unwrap();
    let (target, target_size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    let start = [(source[0] + size[0] * 0.4) as i32, (source[1] + 10.) as i32];
    let end = [
        (target[0] + target_size[0] * 0.5) as i32,
        (target[1] + 10.) as i32,
    ];
    native_drag(world, start, end);
}

#[then("both browsers occupy one right-side dock region as tabs")]
#[then("Appearance Browser returns as a tab in the right dock region")]
fn browsers_share_dock(_: &mut ApplicationWorld) {
    let node = diagnostics::window_dock_id(APPEARANCE).unwrap();
    assert_ne!(node, 0);
    assert_eq!(diagnostics::window_dock_id(MATERIAL), Some(node));
}

#[then("one browser is active while the other remains available")]
fn one_browser_active(_: &mut ApplicationWorld) {
    assert_ne!(
        diagnostics::window_dock_tab_visible(APPEARANCE),
        diagnostics::window_dock_tab_visible(MATERIAL)
    );
}

#[when("the user selects the Appearance Browser tab")]
fn select_appearance(world: &mut ApplicationWorld) {
    select_tab(world, APPEARANCE);
}
#[when("the user selects the Material Browser tab")]
fn select_material(world: &mut ApplicationWorld) {
    select_tab(world, MATERIAL);
}

fn select_tab(world: &mut ApplicationWorld, panel: &str) {
    let (min, max) = diagnostics::window_dock_tab_bounds(panel).unwrap();
    world.click(diagnostics::pointer_pixels([
        (min[0] + max[0]) * 0.5,
        (min[1] + max[1]) * 0.5,
    ]));
    world.pump(Vec::new());
    assert_eq!(diagnostics::window_dock_tab_visible(panel), Some(true));
}

#[then("Appearance Browser contents become visible in the dock region")]
fn appearance_contents(world: &mut ApplicationWorld) {
    world.run_in_window(APPEARANCE, "Assigntopart");
}
#[then("Material Browser contents become visible in the dock region")]
fn material_contents(world: &mut ApplicationWorld) {
    world.run_in_window(MATERIAL, "Assigntopart");
}

#[when("the user selects Appearance Browser and closes it from its panel controls")]
fn close_appearance(world: &mut ApplicationWorld) {
    select_tab(world, APPEARANCE);
    super::close_workspace_tab(world, APPEARANCE);
}

#[then("Appearance Browser is removed from the tab group")]
fn appearance_closed(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(APPEARANCE).is_none());
}
#[then("Material Browser remains visible in the right dock region")]
fn material_remains(world: &mut ApplicationWorld) {
    assert_eq!(diagnostics::window_dock_tab_visible(MATERIAL), Some(true));
    world.run_in_window(MATERIAL, "Assigntopart");
    world.save_capture("material-tab-after-closing-appearance");
}

#[when("the user opens View and then User Interface")]
fn open_view_interface(world: &mut ApplicationWorld) {
    world.click_text("View");
    world.click_text("User Interface");
}
#[when("the user selects Appearance Browser")]
fn restore_appearance(world: &mut ApplicationWorld) {
    world.click_text(APPEARANCE);
    world.pump(Vec::new());
    world.docking_selected_tab = Some(APPEARANCE.to_owned());
}

#[then("a single tab bar contains both browser tabs")]
fn browser_tabs_share_one_bar(_: &mut ApplicationWorld) {
    let appearance = diagnostics::window_dock_id(APPEARANCE).expect("appearance dock node");
    assert_ne!(appearance, 0, "expected a docked appearance browser");
    assert_eq!(diagnostics::window_dock_id(MATERIAL), Some(appearance));
    let (a_min, a_max) = diagnostics::window_dock_tab_bounds(APPEARANCE).expect("appearance tab");
    let (m_min, m_max) = diagnostics::window_dock_tab_bounds(MATERIAL).expect("material tab");
    assert!(
        (a_min[1] - m_min[1]).abs() <= 1. && (a_max[1] - m_max[1]).abs() <= 1.,
        "browser tabs must share one row: {a_min:?}..{a_max:?}, {m_min:?}..{m_max:?}"
    );
    assert!(
        a_max[0] <= m_min[0] || m_max[0] <= a_min[0],
        "browser tabs must be distinct"
    );
}

#[then("the panel presents appearance choices without a second browser tab bar")]
fn appearance_has_no_nested_tabs(world: &mut ApplicationWorld) {
    assert_single_asset_panel(world, APPEARANCE, "Appearance");
}

#[then("the panel presents material choices without a second browser tab bar")]
fn material_has_no_nested_tabs(world: &mut ApplicationWorld) {
    assert_single_asset_panel(world, MATERIAL, "Material");
}

fn assert_single_asset_panel(world: &ApplicationWorld, panel: &str, label: &str) {
    assert_eq!(diagnostics::window_dock_tab_visible(panel), Some(true));
    world.run_in_window(panel, label);
    world.run_in_window(panel, "Assigntopart");
    let (position, size) = diagnostics::window_geometry(panel).expect("browser panel geometry");
    for run in &world.frame.runs {
        let center = run.center();
        let inside = (0..2).all(|axis| {
            center[axis] >= position[axis] && center[axis] <= position[axis] + size[axis]
        });
        assert!(
            !inside || !matches!(run.text.as_str(), "Materials" | "Appearances"),
            "unexpected inner browser tab {:?}",
            run.text
        );
    }
}
