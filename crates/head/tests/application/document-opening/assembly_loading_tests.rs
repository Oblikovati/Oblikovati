// SPDX-License-Identifier: GPL-2.0-only
//! Native assembly opening and placement checks.
use super::*;

#[test]
fn express_open_without_saved_view_keeps_existing_document_active() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    let folder = models_folder(&mut world);
    saved_document(
        &mut world,
        GuiTemplateKind::Assembly,
        &folder,
        ASSEMBLY_FILE,
    );
    close_active(&mut world);
    let path = folder.join(ASSEMBLY_FILE);
    let package =
        oblikovati_persistence::package::DocumentPackage::decode(&std::fs::read(&path).unwrap())
            .unwrap()
            .without_stream("cache/assembly-view-v1");
    std::fs::write(&path, package.encode().unwrap()).unwrap();
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    toggle_express(&mut world);
    choose_open(&mut world);
    assert_eq!(world.active_id(), part);
    world.expect_text("expected a saved assembly view");
    world.close();
}

#[test]
fn populated_assembly_reports_a_missing_component() {
    let mut world = ApplicationWorld::new();
    let part = world.add_part();
    let folder = models_folder(&mut world);
    std::fs::write(folder.join(ASSEMBLY_FILE), b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Part:1', component: part.opd}]}\n").unwrap();
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    choose_open(&mut world);
    assert_ne!(world.active_id(), part);
    let status = world.app().window.native.scene_status().unwrap();
    assert!(
        status
            .scene_errors
            .iter()
            .any(|error| error.contains("part.opd"))
    );
    assert_eq!(status.visible_bodies, 0);
    world.expect_text("Part:1");
    world.expect_text("[unresolved]");
    world.close();
}

#[test]
fn opening_an_assembly_displays_its_referenced_part() {
    let mut world = ApplicationWorld::new();
    world.add_part();
    let folder = models_folder(&mut world);
    let fixture = crate::scene_fixture::selected_prism_document();
    let part = oblikovati_persistence::package::DocumentPackage::decode(
        b"schemaVersion: 2\ndocumentType: 1\n",
    )
    .unwrap()
    .with_parameters(&fixture.parameters, Default::default())
    .unwrap()
    .with_sketches(&fixture.sketches, 1, 104)
    .unwrap()
    .with_work_geometry(&fixture.work_geometry, 1)
    .unwrap();
    std::fs::write(folder.join("part.opd"), part.encode().unwrap()).unwrap();
    std::fs::write(folder.join("nested.oad"), b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Part:2', component: part.opd}]}\n").unwrap();
    std::fs::write(folder.join(ASSEMBLY_FILE), b"schemaVersion: 2\ndocumentType: 2\nmodel: {occurrences: [{name: 'Part:1', component: part.opd}, {name: 'Nested:1', component: nested.oad, transform: [1, 0, 0, 5, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]}, {name: 'Suppressed:1', component: part.opd, suppressed: true}]}\n").unwrap();
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    choose_open(&mut world);
    settle(&mut world);
    let status = world.app().window.native.scene_status().unwrap();
    assert_eq!(status.visible_bodies, 2);
    assert!(status.scene_errors.is_empty(), "{:?}", status.scene_errors);
    world.expect_text("Part:1");
    world.expect_text("Nested:1");
    world.expect_text("Part:2");
    let brightness = |run: &super::super::super::diagnostics::RenderedRun| {
        run.color.iter().map(|value| u32::from(*value)).sum::<u32>() * u32::from(run.alpha)
    };
    let active = brightness(world.run("Part:1"));
    let suppressed = brightness(world.run("Suppressed:1"));
    assert!(suppressed < active);
    world.click_text("Part:2");
    let selected = world
        .state()
        .browser
        .as_ref()
        .unwrap()
        .selected_occurrence
        .clone()
        .unwrap();
    assert_eq!(selected.len(), 2);
    assert_selected_and_unselected_solids(&mut world, true);
    world.click_text("Nested:1");
    let parent = world
        .state()
        .browser
        .as_ref()
        .unwrap()
        .selected_occurrence
        .clone()
        .unwrap();
    assert_eq!(parent.len(), 1);
    assert_selected_and_unselected_solids(&mut world, true);
    world.expect_text("Part:2");
    world.click_text("Part:1");
    let other = world
        .state()
        .browser
        .as_ref()
        .unwrap()
        .selected_occurrence
        .clone()
        .unwrap();
    assert_eq!(other.len(), 1);
    assert_ne!(selected, other);
    assert_ne!(parent, other);
    assert_selected_and_unselected_solids(&mut world, false);
    world.save_capture("assembly-referenced-part");
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let (minimum, maximum) = super::super::super::solid_visibility::visible_fixture_bounds(
        &capture,
        &super::super::super::diagnostics::window_occlusions(),
    )
    .expect("expected the unselected neutral occurrence");
    let point = std::array::from_fn(|axis| ((minimum[axis] + maximum[axis]) / 2) as i32);
    let select_button = world
        .frame
        .runs
        .iter()
        .filter(|run| run.text == "Select")
        .min_by(|a, b| a.center()[1].total_cmp(&b.center()[1]))
        .unwrap()
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(
        select_button,
    ));
    world.click_text("Faces");
    world.click(point);
    assert_eq!(
        world.state().browser.as_ref().unwrap().selected_occurrence,
        Some(other)
    );
    world.click(super::super::super::diagnostics::pointer_pixels(
        select_button,
    ));
    world.click_text("Components");
    world.click(point);
    assert_eq!(
        world.state().browser.as_ref().unwrap().selected_occurrence,
        Some(selected)
    );
    assert_selected_and_unselected_solids(&mut world, true);
    world.save_capture("assembly-picked-occurrence");
    world.click([400, 700]);
    assert!(
        world
            .state()
            .browser
            .as_ref()
            .unwrap()
            .selected_occurrence
            .is_none()
    );
    let before = neutral_assembly_bounds(&mut world);
    occurrence_menu(&mut world, "Part:1", "Suppress");
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        1
    );
    let after = neutral_assembly_bounds(&mut world);
    assert!(
        after.0[0] > before.0[0],
        "expected the left placement to disappear"
    );
    world.save_capture("assembly-suppressed-occurrence");
    world.key(ViewportPhysicalKey::Z, true);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        2
    );
    world.key(ViewportPhysicalKey::Y, true);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        1
    );
    occurrence_menu(&mut world, "Part:1", "Unsuppress");
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        2
    );
    world.click_text("Save");
    close_active(&mut world);
    open_dialog(&mut world);
    look_in(&mut world, &folder);
    select_row(&mut world, ASSEMBLY_FILE);
    choose_open(&mut world);
    assert_eq!(
        world
            .app()
            .window
            .native
            .scene_status()
            .unwrap()
            .visible_bodies,
        2
    );
    world.click_text("Search");
    world.pump(Vec::new());
    super::super::super::workspace_layout::browser_tree::type_in_search(&mut world, "Part:2");
    world.expect_text("Nested:1");
    world.expect_text("Part:2");
    world.expect_no_text("Part:1");
    world.close();
}
fn occurrence_menu(world: &mut ApplicationWorld, name: &str, command: &str) {
    let center = world.run(name).center();
    world.right_click(super::super::super::diagnostics::pointer_pixels(center));
    world.click_text(command);
}
fn neutral_assembly_bounds(world: &mut ApplicationWorld) -> ([usize; 2], [usize; 2]) {
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    super::super::super::solid_visibility::visible_fixture_bounds(
        &capture,
        &super::super::super::diagnostics::window_occlusions(),
    )
    .expect("expected a visible assembly component")
}
fn assert_selected_and_unselected_solids(world: &mut ApplicationWorld, selected_right: bool) {
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let width = capture.extent[0] as usize;
    let mut colors = [0usize; 2];
    let mut x_totals = [0usize; 2];
    for (index, pixel) in capture.rgba.as_chunks::<4>().0.iter().enumerate() {
        if index % width < 320 || index / width < 200 {
            continue;
        }
        if super::super::super::solid_visibility::neutral_fixture_hue(pixel) {
            colors[0] += 1;
            x_totals[0] += index % width;
        }
        if pixel[2] > pixel[0].saturating_add(30) && pixel[2] > pixel[1].saturating_add(30) {
            colors[1] += 1;
            x_totals[1] += index % width;
        }
    }
    assert!(
        colors.iter().all(|count| *count > 100),
        "expected neutral and blue solids, got {colors:?}"
    );
    let neutral_x = x_totals[0] / colors[0];
    let blue_x = x_totals[1] / colors[1];
    assert_eq!(
        blue_x > neutral_x,
        selected_right,
        "blue at {blue_x}, neutral at {neutral_x}"
    );
}
