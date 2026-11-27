// SPDX-License-Identifier: GPL-2.0-only
//! Switch native Part, Assembly, and Sheet Metal documents while the ribbon is docked.
use super::super::{diagnostics::pointer_pixels, scenario_root, world::squeezed};
use super::*;

#[given("a part, an assembly and a sheet-metal part are open in the MCAD Software")]
fn three_documents(world: &mut ApplicationWorld) {
    use oblikovati_gui::GuiTemplateKind;
    world.previous_document = Some(world.add_part());
    world.other_document = Some(world.add_template(GuiTemplateKind::Assembly));
    let sheet_metal = world.add_template(GuiTemplateKind::SheetMetal);
    world.parts.push(sheet_metal);
}

#[given(
    "`fixtures/cad/blank-metric-part.opd`, `fixtures/cad/blank-metric-assembly.oad` and `fixtures/cad/ribbon-docking/native-sheet-metal.opd` are open in the MCAD Software"
)]
fn three_saved_documents(world: &mut ApplicationWorld) {
    use oblikovati_gui::GuiBrowserKind;
    let root = scenario_root();
    let [part, assembly, sheet_metal] = [
        ("fixtures/cad/blank-metric-part.opd", GuiBrowserKind::Part),
        (
            "fixtures/cad/blank-metric-assembly.oad",
            GuiBrowserKind::Assembly,
        ),
        (
            "fixtures/cad/ribbon-docking/native-sheet-metal.opd",
            GuiBrowserKind::SheetMetal,
        ),
    ]
    .map(|(relative, kind)| {
        let path = root.join(relative);
        world.open_fixture(&path);
        let document = world.active_id();
        let state = world.state();
        assert_eq!(state.browser.unwrap().kind, kind);
        let storage = &state.document_storage[&document];
        assert_eq!(
            storage.path.as_deref().map(std::path::Path::new),
            Some(path.as_path())
        );
        assert!(!storage.unsaved_edits);
        document
    });
    assert!(
        world
            .state()
            .documents
            .iter()
            .map(|row| row.id)
            .eq([part, assembly, sheet_metal])
    );
    world.previous_document = Some(part);
    world.other_document = Some(assembly);
    world.parts.push(sheet_metal);
}

fn select_document(world: &mut ApplicationWorld, document: u64) {
    let name = world.document_name(document);
    let point = match world.tab_run(&name) {
        Some(tab) => pointer_pixels(tab.center()),
        None => document_list_point(world, &name),
    };
    click_document_control(world, point);
    assert_eq!(world.state().document, Some(document), "activating {name}");
}

fn document_list_point(world: &mut ApplicationWorld, name: &str) -> [i32; 2] {
    let control = world.run_in_window("##document-list-control", "Documents");
    click_document_control(world, pointer_pixels(control.center()));
    let list = world.run("Close Tabs").list;
    let wanted = squeezed(name);
    let row = world.frame.runs.iter().find(|run| {
        run.list == list && squeezed(&run.text).split('<').next() == Some(wanted.as_str())
    });
    pointer_pixels(
        row.unwrap_or_else(|| panic!("expected {name:?} in Documents"))
            .center(),
    )
}

fn click_document_control(world: &mut ApplicationWorld, point: [i32; 2]) {
    #[cfg(target_os = "linux")]
    super::super::desktop_input::click(world, point);
    #[cfg(not(target_os = "linux"))]
    world.click(point);
    world.pump(Vec::new());
}

#[given("the part is active with the ribbon docked on the left")]
#[given("`blank-metric-part.opd` is active with the ribbon docked on the left")]
fn part_docked_left(world: &mut ApplicationWorld) {
    let part = world.previous_document.expect("expected the part");
    select_document(world, part);
    choose_position(world, "Left");
    let (position, _) = window_geometry(COLUMN).expect("expected the side column");
    assert_eq!(position[0], 0.);
}
#[when("I activate the assembly document")]
#[when("the user activates `blank-metric-assembly.oad`")]
fn activate_assembly(world: &mut ApplicationWorld) {
    let assembly = world.other_document.expect("expected the assembly");
    select_document(world, assembly);
}
#[then("it displays assembly tabs and commands")]
#[then("assembly tabs and commands are displayed")]
fn assembly_tabs(world: &mut ApplicationWorld) {
    world.expect_text("Assemble");
    world.expect_text("Component");
}
#[when("I activate the sheet-metal part")]
#[when("the user activates the sheet-metal part")]
#[when("the user activates `native-sheet-metal.opd`")]
fn activate_sheet_metal(world: &mut ApplicationWorld) {
    let sheet_metal = *world.parts.last().expect("expected the sheet-metal part");
    select_document(world, sheet_metal);
}
#[then("it displays the Sheet Metal tab and its commands")]
#[then("the Sheet Metal tab and its commands are displayed")]
fn sheet_metal_tabs(world: &mut ApplicationWorld) {
    world.expect_text("Sheet Metal");
    world.expect_text("Sketch");
}
#[then("the sheet-metal part is active")]
fn sheet_metal_active(world: &mut ApplicationWorld) {
    let sheet_metal = *world.parts.last().expect("expected the sheet-metal part");
    assert_eq!(world.state().document, Some(sheet_metal));
}

#[then("the browser and viewport show the assembly")]
fn shows_assembly(world: &mut ApplicationWorld) {
    let assembly = world.other_document.expect("expected the saved Assembly");
    expect_docked_document(world, assembly, oblikovati_gui::GuiBrowserKind::Assembly);
}
#[then("the browser and viewport show the sheet-metal part")]
fn shows_sheet_metal(world: &mut ApplicationWorld) {
    let sheet = *world
        .parts
        .last()
        .expect("expected the saved sheet-metal Part");
    expect_docked_document(world, sheet, oblikovati_gui::GuiBrowserKind::SheetMetal);
}
fn expect_docked_document(
    world: &mut ApplicationWorld,
    document: u64,
    kind: oblikovati_gui::GuiBrowserKind,
) {
    world.expect_active(document);
    assert_eq!(world.state().browser.unwrap().kind, kind);
    let [left, top, right, bottom] = world.viewport_bounds();
    assert!(left < right && top < bottom);
}
