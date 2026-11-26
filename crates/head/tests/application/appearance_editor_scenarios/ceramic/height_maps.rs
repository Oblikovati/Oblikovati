// SPDX-License-Identifier: GPL-2.0-only
//! Verify chooser cancellation and that imported maps travel with the Part.
use super::*;

const CHOOSER: &str = "Choose Image File";

#[given("Finish Bumps is unchecked")]
fn bumps_disabled(world: &mut ApplicationWorld) {
    assert!(!world.checked_run(super::super::effects::effect_checkbox(world, "FinishBumps")));
}

#[when("I expand Finish Bumps")]
fn expand_bumps(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "FinishBumps");
}

#[then("its Type and Amount controls are disabled")]
fn disabled_controls(world: &mut ApplicationWorld) {
    check_bump_controls(world, false);
}

#[when("I enable Finish Bumps")]
fn enable_bumps(world: &mut ApplicationWorld) {
    toggle_effect(world, "FinishBumps");
}

#[then("its Type and Amount controls become enabled")]
fn enabled_controls(world: &mut ApplicationWorld) {
    check_bump_controls(world, true);
}

#[then("Type offers Wavy and Custom")]
fn bump_type_choices(world: &mut ApplicationWorld) {
    check_choices(world, "Wavy", &["Wavy", "Custom"]);
}

#[when("I choose Custom for the bump type")]
fn choose_custom(world: &mut ApplicationWorld) {
    world.last_capture = Some(capture_framebuffer(&mut world.app().window.native));
    click_in(world, EDITOR, "Wavy");
    select_menu_item(world, "Custom");
}

#[then("an image file chooser opens")]
fn chooser_open(world: &mut ApplicationWorld) {
    world.run_in_window(CHOOSER, "Filename");
    assert!(world.window_text(CHOOSER).contains("ImageFiles"));
    assert!(world.run_in_window(CHOOSER, "Open").alpha < u8::MAX);
}

#[when("I cancel the file chooser")]
fn cancel_chooser(world: &mut ApplicationWorld) {
    click_in(world, CHOOSER, "Cancel");
    assert!(diagnostics::window_geometry(CHOOSER).is_none());
}

#[then("the bump Type remains Wavy")]
fn wavy_remains(world: &mut ApplicationWorld) {
    check_bump_controls(world, true);
    let before = world.last_capture.take().unwrap();
    let after = capture_framebuffer(&mut world.app().window.native);
    assert!(preview_pixels(&before, EDITOR).eq(preview_pixels(&after, EDITOR)));
    assert!(
        ceramic_asset(world)
            .appearance_metadata
            .unwrap()
            .ceramic
            .unwrap()
            .custom_bump
            .is_none()
    );
}

#[test]
fn ceramic_maps_are_saved_without_the_source_images() {
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    asset_browser_shared::open_tool(&mut world, BROWSER);
    create_ceramic(&mut world);
    let bump = write_map(&mut world, "finish-bump.png", false);
    let relief = write_map(&mut world, "relief.png", true);

    open_bump_chooser(&mut world);
    choose_file(&mut world, &bump);
    world.run_in_window(EDITOR, "Custom");
    world.run_in_window(EDITOR, "finish-bump.png");
    preview_changed(&mut world, 4, 100);
    click_in(&mut world, EDITOR, "FinishBumps");
    expand_relief(&mut world);
    toggle_effect(&mut world, "ReliefPattern");
    click_in(&mut world, EDITOR, "Image");
    choose_file(&mut world, &relief);
    assert!(world.window_text(EDITOR).contains("relief.png"));

    for path in [&bump, &relief] {
        std::fs::remove_file(path).unwrap();
    }
    apply(&mut world);
    let path = world.save_part_to_file(document, "ceramic-height-maps.opd");
    let saved = DocumentPackage::decode(&std::fs::read(path).unwrap()).unwrap();
    let assets = saved.restore_assets().unwrap();
    let ceramic = assets
        .appearance_metadata(CERAMIC_ID)
        .unwrap()
        .ceramic
        .as_ref()
        .unwrap();
    let bump = ceramic.custom_bump.as_ref().unwrap();
    let relief = ceramic.relief_image.as_ref().unwrap();
    assert_eq!(
        [bump.name(), relief.name()],
        ["finish-bump.png", "relief.png"]
    );
    assert_eq!(bump.samples().len(), 32 * 32);
    assert_ne!(bump.samples(), relief.samples());
}

fn open_bump_chooser(world: &mut ApplicationWorld) {
    expand_bumps(world);
    enable_bumps(world);
    choose_custom(world);
    chooser_open(world);
}

pub(in super::super) fn choose_file(world: &mut ApplicationWorld, file: &std::path::Path) {
    world.type_text(file.to_str().unwrap());
    world.pump(Vec::new());
    click_in(world, CHOOSER, "Open");
    assert!(
        diagnostics::window_geometry(CHOOSER).is_none(),
        "{}",
        world.window_text(CHOOSER)
    );
}

pub(in super::super) fn write_map(
    world: &mut ApplicationWorld,
    name: &str,
    vertical: bool,
) -> std::path::PathBuf {
    let path = world.scratch_folder().join(name);
    let mut png = png::Encoder::new(std::fs::File::create(&path).unwrap(), 32, 32);
    png.set_color(png::ColorType::Grayscale);
    let values: Vec<u8> = (0..32 * 32)
        .map(|index| {
            let axis = if vertical { index / 32 } else { index % 32 };
            if axis % 8 < 4 { 0 } else { 255 }
        })
        .collect();
    png.write_header()
        .unwrap()
        .write_image_data(&values)
        .unwrap();
    path
}

fn check_bump_controls(world: &ApplicationWorld, enabled: bool) {
    for label in ["Wavy", "Amount"] {
        assert_eq!(
            world.run_in_window(EDITOR, label).alpha == u8::MAX,
            enabled,
            "{label}"
        );
    }
}

fn toggle_effect(world: &mut ApplicationWorld, title: &str) {
    let check = super::super::effects::effect_checkbox(world, title);
    world.click(diagnostics::pointer_pixels(check.center()));
    world.pump(Vec::new());
}
