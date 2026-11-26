// SPDX-License-Identifier: GPL-2.0-only
//! Verify a saved dock group through an application restart.
use super::{ApplicationWorld, PANEL, diagnostics, native_drag, panel_docked, viewport};
use cucumber::{given, then, when};
const APPEARANCE: &str = "Appearance Browser";

#[given("`fixtures/cad/docking-visible-solid-current.opd` is open")]
fn restart_fixture(world: &mut ApplicationWorld) {
    super::named_docking_fixture(world);
    world.saved_path = Some(super::super::scenario_root().join(super::DOCKING_PART));
}

#[given("the user has changed the workspace panel arrangement")]
#[given(
    "Model Browser is docked on the left and Appearance Browser and Material Browser share a right dock as tabs"
)]
fn arrange_tabbed_panels(world: &mut ApplicationWorld) {
    super::open_docking_fixture(world);
    panel_docked(world);
    super::super::workspace_layout::dockspace_native::open_toolbar_tool(world, APPEARANCE);
    let (source, size) = diagnostics::window_geometry(APPEARANCE).unwrap();
    let (target, bounds) = diagnostics::window_geometry(PANEL).unwrap();
    let start = [(source[0] + size[0] * 0.4) as i32, (source[1] + 10.) as i32];
    let end = [
        (target[0] + bounds[0] * 0.5) as i32,
        (target[1] + 10.) as i32,
    ];
    native_drag(world, start, end);
    let node = diagnostics::window_dock_id(PANEL).unwrap();
    assert_ne!(node, 0);
    assert_eq!(diagnostics::window_dock_id(APPEARANCE), Some(node));
    super::browser_scenarios::model_browser_left(world);
    assert_ne!(diagnostics::window_dock_id("Model").unwrap(), 0);
    let (_, size) = diagnostics::window_geometry(PANEL).unwrap();
    world.docking_resize = Some((viewport(world), size[0]));
    world.docking_node = Some(node);
    world.docking_active_tab = diagnostics::window_dock_tab_visible(APPEARANCE);
}

#[given("the MCAD Software has saved that arrangement")]
fn workspace_is_saveable(world: &mut ApplicationWorld) {
    let bytes = world
        .app()
        .window
        .native
        .gui
        .as_mut()
        .unwrap()
        .save_workspace_layout()
        .unwrap();
    let store =
        crate::native::preferences::NativeWorkspaceLayout::new(world.preferences_directory());
    store.save(&bytes).unwrap();
    assert_eq!(store.load().unwrap(), Some(bytes));
}

#[when("the user closes and restarts the MCAD Software")]
fn restart_application(world: &mut ApplicationWorld) {
    let wayland = world.app().window.native.view.uses_wayland();
    world.close();
    run_desktop_restart_check(world, wayland);
    world.reopen_application();
    if world.saved_path.is_none() {
        super::open_docking_part(world);
        world.pump(Vec::new());
    }
}

#[when("the user opens the same native part")]
fn open_same_part(world: &mut ApplicationWorld) {
    assert!(world.state().documents.is_empty());
    let document = super::open_docking_part(world);
    let storage = &world.state().document_storage[&document];
    assert_eq!(
        storage.path.as_deref().map(std::path::Path::new),
        world.saved_path.as_deref()
    );
    super::workspace_layout::browser_feature::expect_solid_visible(world);
}

#[then("the saved panel positions, sizes and tab groups are restored")]
#[then(
    "the saved left Model Browser and right tabbed browsers are restored in their positions and sizes"
)]
pub(super) fn restored_group(world: &mut ApplicationWorld) {
    let node = diagnostics::window_dock_id(PANEL).unwrap();
    assert_ne!(node, 0);
    assert_eq!(diagnostics::window_dock_id(APPEARANCE), Some(node));
    super::browser_scenarios::model_browser_left(world);
    assert_ne!(diagnostics::window_dock_id("Model").unwrap(), 0);
    assert_eq!(
        diagnostics::window_dock_tab_visible(APPEARANCE),
        world.docking_active_tab
    );
    let (area, width) = world.docking_resize.unwrap();
    assert_eq!(viewport(world), area);
    assert_eq!(diagnostics::window_geometry(PANEL).unwrap().1[0], width);
}

fn desktop_executable() -> std::path::PathBuf {
    std::env::var_os("OBK_BDD_DESKTOP_BINARY")
        .map(std::path::PathBuf::from)
        .unwrap_or_else(|| {
            std::env::current_exe()
                .unwrap()
                .parent()
                .unwrap()
                .parent()
                .unwrap()
                .join("oblikovati")
        })
}

fn run_desktop_restart_check(world: &ApplicationWorld, wayland: bool) {
    let script = restart_driver();
    let layout = saved_restart_layout(world);
    let mut driver = restart_command(world, script.path(), layout.path());
    if wayland {
        driver.arg("--wayland");
    }
    let result = driver
        .output()
        .expect("docking restart: expected python3 and the native desktop test");
    assert!(
        result.status.success(),
        "desktop restart failed: {}",
        String::from_utf8_lossy(&result.stderr)
    );
}

fn restart_driver() -> tempfile::NamedTempFile {
    let script = tempfile::NamedTempFile::new().expect("restart driver: expected a temporary file");
    std::fs::write(
        script.path(),
        include_str!(concat!(
            env!("CARGO_MANIFEST_DIR"),
            "/../../tools/bdd/docking_process_restart.py"
        )),
    )
    .expect("restart driver: expected to write the embedded source");
    script
}

fn restart_command(
    world: &ApplicationWorld,
    script: &std::path::Path,
    layout: &std::path::Path,
) -> std::process::Command {
    let fixture =
        super::super::scenario_root().join("fixtures/cad/docking-visible-solid-current.opd");
    let output = super::super::world::reports_folder().join("docking-process-restart");
    let (area, _) = world.docking_resize.expect("saved dock layout bounds");
    let mut driver = std::process::Command::new("python3");
    driver
        .args([
            script,
            desktop_executable().as_path(),
            fixture.as_path(),
            output.as_path(),
        ])
        .arg("--layout")
        .arg(layout)
        .arg("--area")
        .args(area.map(|edge| edge.to_string()));
    driver
}

fn saved_restart_layout(world: &ApplicationWorld) -> tempfile::NamedTempFile {
    let store =
        crate::native::preferences::NativeWorkspaceLayout::new(world.preferences_directory());
    let bytes = store
        .load()
        .expect("read the saved workspace layout")
        .expect("scenario saved a workspace layout before closing");
    let staged = tempfile::NamedTempFile::new().expect("restart layout: expected a temporary file");
    std::fs::write(staged.path(), bytes).expect("stage the saved layout for desktop restart");
    staged
}
