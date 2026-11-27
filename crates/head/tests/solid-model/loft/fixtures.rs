// SPDX-License-Identifier: GPL-2.0-only
//! Supply the pinned Loft scenarios' implicit starting Part.
use super::*;
use oblikovati_gui::GuiTemplateKind;

#[given("a new empty Standard (mm) part")]
fn open_empty_metric_part(world: &mut ApplicationWorld) {
    world.add_template(GuiTemplateKind::Part);
    empty_part_has_no_loft_inputs(world);
}

#[given("two visible circular sketches on separate parallel planes")]
#[given("two circular sketches on separate parallel planes in a new part")]
fn open_parallel_sections(world: &mut ApplicationWorld) {
    parallel_sections_if_needed(world);
    independent_circles_are_visible(world);
}

pub(super) fn parallel_sections_if_needed(world: &mut ApplicationWorld) {
    embedded_fixture_if_needed(
        world,
        "parallel-circles.opd",
        include_bytes!("../../../../../test-utilities/fixtures/loft/parallel-circles.opd"),
    );
}

pub(super) fn center_line_sections_if_needed(world: &mut ApplicationWorld) {
    embedded_fixture_if_needed(
        world,
        "circles-center-line.opd",
        include_bytes!("../../../../../test-utilities/fixtures/loft/circles-center-line.opd"),
    );
}

pub(super) fn short_center_line_sections_if_needed(world: &mut ApplicationWorld) {
    embedded_fixture_if_needed(
        world,
        "circles-center-line-20.opd",
        include_bytes!("../../../../../test-utilities/fixtures/loft/circles-center-line-20.opd"),
    );
}

fn embedded_fixture_if_needed(world: &mut ApplicationWorld, name: &str, bytes: &[u8]) {
    if world.state().document.is_some() {
        return;
    }
    let path = world.scratch_folder().join(name);
    std::fs::write(&path, bytes).unwrap();
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&path, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
}
