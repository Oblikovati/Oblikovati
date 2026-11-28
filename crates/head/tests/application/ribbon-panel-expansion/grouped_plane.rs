// SPDX-License-Identifier: GPL-2.0-only
//! Moving the grouped Plane control between ribbon areas.
use super::{ApplicationWorld, diagnostics};
use cucumber::{given, then, when};

const EXPANSION: &str = "##ribbon-additional-PartWorkFeatures";

#[given("the grouped \"Plane\" control is in the 3D Model Work Features main panel")]
fn plane_in_main_panel(world: &mut ApplicationWorld) {
    world.expect_text("Work Features");
    world.expect_text("Plane");
}

#[given("the 3D Model Work Features panel contains a grouped \"Plane\" command")]
fn grouped_plane_on_a_visible_part(world: &mut ApplicationWorld) {
    let has_solid = world
        .state()
        .browser
        .is_some_and(|browser| browser.bodies.iter().any(|body| body.visible));
    if !has_solid {
        // The pinned scenario predates the named Part supplied by the published source.
        super::open_visible_extruded_part(world);
        world.previous_document = Some(world.active_id());
    }
    plane_in_main_panel(world);
}

#[when("the user right-clicks \"Plane\" and selects \"Ungroup Commands from Drop-Down Menu\"")]
fn ungroup_plane_choices(world: &mut ApplicationWorld) {
    let plane = world.run_in_window("##application-ribbon", "Plane");
    world.right_click(diagnostics::pointer_pixels(plane.center()));
    world.click_text("Ungroup Commands from Drop-Down Menu");
    world.pump(Vec::new());
}

#[when("the user opens the compact Work Features panel")]
#[when("the user opens the Work Features panel again")]
fn show_compact_work_features(world: &mut ApplicationWorld) {
    let button = compact_work_features(world);
    world.click(diagnostics::pointer_pixels(button.center()));
    assert!(diagnostics::command_popup_open());
}

fn compact_work_features(world: &ApplicationWorld) -> &diagnostics::RenderedRun {
    // Expanded panels use uppercase captions, while the compact button uses its title.
    world.run_in_window("##application-ribbon", "WorkFeatures")
}

#[then(
    "separate plane command buttons are visible including \"Offset from Plane\" and \"Three Points\""
)]
fn individual_plane_choices(world: &mut ApplicationWorld) {
    let plane = world.run("Plane");
    let offset = world.run("Offset from Plane");
    let three_points = world.run("Three Points");
    assert_eq!(offset.list, plane.list);
    assert_eq!(three_points.list, plane.list);
    assert_eq!(offset.alpha, u8::MAX);
    assert_ne!(offset.center(), three_points.center());
}

#[when("the user right-clicks \"Plane\" and selects \"Group Commands into Drop-Down Menu\"")]
fn regroup_plane_choices(world: &mut ApplicationWorld) {
    let plane = world.run("Plane");
    world.right_click(diagnostics::pointer_pixels(plane.center()));
    world.click_text("Group Commands into Drop-Down Menu");
    world.pump(Vec::new());
}

#[then("the plane choices are represented by a grouped \"Plane\" control")]
fn grouped_plane_choices(world: &mut ApplicationWorld) {
    assert!(diagnostics::command_popup_open());
    world.expect_text("Plane");
    for name in ["Offset from Plane", "Three Points"] {
        world.expect_no_text(name);
    }
    assert!(world.previous_document.is_some());
    assert_eq!(world.state().document, world.previous_document);
}

#[cfg(target_os = "linux")]
#[given("the 3D Model ribbon is compact after regrouping the Plane choices")]
fn compact_after_regrouping(world: &mut ApplicationWorld) {
    assert!(!world.app().window.native.view.window_maximized().unwrap());
    ungroup_plane_choices(world);
    show_compact_work_features(world);
    regroup_plane_choices(world);
    world.escape();
    world.app().window.native.view.resize([1240, 850]).unwrap();
    for _ in 0..4 {
        world.pump(Vec::new());
    }
    assert_eq!(world.extent()[0], 1240);
    compact_work_features(world);
    assert!(!diagnostics::command_popup_open());
    world.observed_counts = vec![model_ribbon_caption_count(world)];
}

#[cfg(target_os = "linux")]
#[then("the ribbon reflows to show more command labels and panels")]
fn expanded_after_maximize(world: &mut ApplicationWorld) {
    super::super::main_window_frame_native::pump_until(world, true);
    assert!(
        model_ribbon_caption_count(world) > world.observed_counts[0],
        "expected more command captions after maximizing"
    );
    for text in ["Plane", "Axis", "Point"] {
        world.run_in_window("##application-ribbon", text);
    }
}

#[cfg(target_os = "linux")]
fn model_ribbon_caption_count(world: &ApplicationWorld) -> usize {
    let title = world.run("Work Features");
    world
        .frame
        .runs
        .iter()
        .filter(|run| {
            run.list == title.list && run.text != "v" && run.maximum[1] < title.minimum[1]
        })
        .count()
}

#[then("the Plane choices remain grouped")]
fn grouped_after_reflow(world: &mut ApplicationWorld) {
    plane_menu_arrow(world);
    world.expect_no_text("Three Points");
    world.expect_no_text("Offset from Plane");
    assert_eq!(world.state().document, world.previous_document);
}

#[when("the user clicks the Plane drop-down arrow")]
fn open_plane_variants(world: &mut ApplicationWorld) {
    let arrow = plane_menu_arrow(world);
    world.click(diagnostics::pointer_pixels(arrow.center()));
    assert!(diagnostics::command_popup_open());
    world.expect_text("Three Points");
}

fn plane_menu_arrow(world: &ApplicationWorld) -> &diagnostics::RenderedRun {
    let plane = world.run_in_window("##application-ribbon", "Plane");
    world
        .frame
        .runs
        .iter()
        .filter(|run| run.list == plane.list && run.text == "▼")
        .filter(|run| {
            run.minimum[0] > plane.maximum[0] && (run.center()[1] - plane.center()[1]).abs() < 18.
        })
        .min_by(|left, right| left.minimum[0].total_cmp(&right.minimum[0]))
        .expect("expected the arrow beside Plane")
}

#[then("a vertical menu opens beneath the control")]
fn menu_beneath_plane(world: &mut ApplicationWorld) {
    let control = world.run_in_window("##application-ribbon", "Plane");
    let rows = menu_rows(world);
    assert_ne!(rows[0].list, control.list);
    assert!(rows[0].minimum[1] > control.center()[1]);
    assert!(
        rows.windows(2)
            .all(|pair| pair[0].maximum[1] < pair[1].minimum[1])
    );
}

#[then("the menu contains \"Plane\", \"Offset from Plane\", and \"Three Points\"")]
fn menu_contains_plane_choices(world: &mut ApplicationWorld) {
    let rows = menu_rows(world);
    assert_eq!(
        rows.map(|row| row.text.as_str()),
        ["Plane", "OffsetfromPlane", "ThreePoints"]
    );
}

fn menu_rows(world: &ApplicationWorld) -> [&diagnostics::RenderedRun; 3] {
    let list = world.run("Offset from Plane").list;
    ["Plane", "OffsetfromPlane", "ThreePoints"].map(|label| {
        world
            .frame
            .runs
            .iter()
            .find(|run| run.list == list && run.text == label)
            .unwrap_or_else(|| panic!("expected {label:?} in the Plane menu"))
    })
}

#[then("the menu presents icons and dividers between groups of choices")]
fn icons_and_rules(world: &mut ApplicationWorld) {
    let [_, offset, three_points] = menu_rows(world);
    #[cfg(target_os = "linux")]
    for choice in [offset, three_points] {
        assert!(
            diagnostics::has_caption_icon(choice),
            "expected an icon beside {}",
            choice.text
        );
    }
    let left = offset.maximum[0].max(three_points.maximum[0]) + 4.;
    let first_group = [
        offset.maximum[1],
        world.run("Midplane between Two Planes").minimum[1],
    ];
    let second_group = [
        world.run("Two Coplanar Edges").maximum[1],
        world.run("Angle to Plane around Edge").minimum[1],
    ];
    let frame = super::super::capture_framebuffer(&mut world.app().window.native);
    let pixel = |x: usize, y: usize| {
        let at = (y * frame.extent[0] as usize + x) * 4;
        &frame.rgba[at..at + 4]
    };
    for [top, bottom] in [first_group, second_group] {
        let [x, start] = diagnostics::pointer_pixels([left, top]).map(|value| value as usize);
        let end = diagnostics::pointer_pixels([left, bottom])[1] as usize;
        assert!(
            (start + 2..end - 1).any(|y| {
                (x..x + 36)
                    .all(|x| pixel(x, y - 2) == pixel(x, y + 2) && pixel(x, y) != pixel(x, y - 2))
            }),
            "expected a horizontal menu divider between y={start} and y={end}"
        );
    }
}

#[then("the part remains visible without a plane command being started")]
fn dismissed_without_action(world: &mut ApplicationWorld) {
    assert!(!diagnostics::command_popup_open());
    offset_command_closes(world);
    plane_and_part_remain_visible(world);
    no_work_plane_created(world);
}

#[when("the user opens the Plane drop-down and selects \"Offset from Plane\"")]
fn select_offset_from_plane(world: &mut ApplicationWorld) {
    open_plane_variants(world);
    world.click_text("Offset from Plane");
    assert!(!diagnostics::command_popup_open());
}

#[then("the software prompts the user to select a planar face")]
fn offset_waits_for_face(world: &mut ApplicationWorld) {
    world.run_in_window("Offset Plane", "SelectPlane");
    assert!(!world.run_is_enabled("OK"));
}

#[when("the user presses Escape without selecting a face")]
fn cancel_offset_before_pick(world: &mut ApplicationWorld) {
    world.escape();
}

#[then("the software returns to the ready state")]
fn offset_command_closes(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Offset Plane").is_none());
    assert!(!world.state().transaction_open);
}

#[then("the grouped Plane control and the part remain visible")]
fn plane_and_part_remain_visible(world: &mut ApplicationWorld) {
    plane_in_main_panel(world);
    assert_eq!(world.state().document, world.previous_document);
    super::super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[then("no new work plane is created")]
fn no_work_plane_created(world: &mut ApplicationWorld) {
    let browser = world.state().browser.expect("expected the Part browser");
    assert!(
        browser
            .features
            .iter()
            .all(|feature| feature.kind == "Extrusion")
    );
}
#[when("the user right-clicks \"Plane\" and selects \"Move to Expanded Panel\"")]
fn move_plane_to_expanded(world: &mut ApplicationWorld) {
    let plane = diagnostics::pointer_pixels(world.run("Plane").center());
    world.right_click(plane);
    world.expect_text("Move to Expanded Panel");
    world.click_text("Move to Expanded Panel");
}
#[then("\"Plane\" is no longer visible in the main panel")]
fn plane_leaves_main_panel(world: &mut ApplicationWorld) {
    assert!(!world.window_text("##application-ribbon").contains("Plane"));
}
#[when("the user opens the Work Features expansion")]
fn open_work_features(world: &mut ApplicationWorld) {
    super::click_expansion_arrow(world, "Work Features");
}
#[then("a \"Plane\" control appears in the expanded area")]
fn plane_in_expansion(world: &mut ApplicationWorld) {
    world.run_in_window(EXPANSION, "Plane");
}
#[then("its context menu offers \"Ungroup Commands from Drop-Down Menu\"")]
fn grouped_context_menu(world: &mut ApplicationWorld) {
    let plane = world.run_in_window(EXPANSION, "Plane");
    world.right_click(diagnostics::pointer_pixels(plane.center()));
    world.expect_text("Ungroup Commands from Drop-Down Menu");
    world.expect_text("Move to Main Panel");
}
#[when("the user selects \"Move to Main Panel\" from that context menu")]
fn restore_plane_to_main(world: &mut ApplicationWorld) {
    world.click_text("Move to Main Panel");
}
#[then("\"Plane\" returns to the main panel")]
fn plane_returns(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(EXPANSION).is_none());
    world.run_in_window("##application-ribbon", "Plane");
}
