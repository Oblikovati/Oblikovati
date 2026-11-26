// SPDX-License-Identifier: GPL-2.0-only
//! Drive both editor duplication commands and verify independent document assets.
use super::*;

const COPY_ID: &str = "default-ceramic-copy-1";
const COPY_NAME: &str = "Default Ceramic(1)";

#[given("an applied ceramic appearance is open in the Appearance Editor")]
fn applied_ceramic(world: &mut ApplicationWorld) {
    create_ceramic(world);
    choose_ceramic(world);
    choose_matte(world);
    let check = super::super::effects::effect_checkbox(world, "FinishBumps");
    world.click(diagnostics::pointer_pixels(check.center()));
    world.pump(Vec::new());
    apply(world);
    applied_wavy(world);
}

#[given("its type is Ceramic, its finish is Matte, and Wavy Finish Bumps are enabled")]
fn applied_wavy(world: &mut ApplicationWorld) {
    let ceramic = ceramic_asset(world)
        .appearance_metadata
        .unwrap()
        .ceramic
        .unwrap();
    assert_eq!(ceramic.kind, CeramicKind::Ceramic);
    assert_eq!(ceramic.finish, CeramicFinish::Matte);
    assert!(ceramic.finish_bumps.enabled && ceramic.custom_bump.is_none());
    clean_dialog(world);
}

#[when("I choose Duplicate from the editor creation menu")]
fn choose_duplicate(world: &mut ApplicationWorld) {
    duplicate_menu_item(world, "Duplicate");
}

#[when("I choose Duplicate as Generic from the editor creation menu")]
fn choose_generic_duplicate(world: &mut ApplicationWorld) {
    world.last_capture = Some(capture_framebuffer(&mut world.app().window.native));
    duplicate_menu_item(world, "DuplicateasGeneric");
}

#[then("the editor switches to a new appearance with a distinct name")]
fn opens_duplicate(world: &mut ApplicationWorld) {
    separate_window(world, COPY_NAME.into());
    let copy = asset(world, COPY_ID);
    assert_eq!(copy.name, COPY_NAME);
    applied_wavy(world);
    let text = world.window_text(BROWSER);
    let selection = super::super::super::world::squeezed(COPY_NAME);
    assert!(
        text.contains(&selection),
        "expected the copy to be selected: {text}"
    );
}

#[then("the new appearance retains the visible type, finish and bump settings")]
fn duplicate_retains_settings(world: &mut ApplicationWorld) {
    let text = world.window_text(EDITOR);
    assert!(text.contains("Matte") && text.contains("Ceramic"), "{text}");
    let check = super::super::effects::effect_checkbox(world, "FinishBumps");
    assert!(world.checked_run(check));
    click_in(world, EDITOR, "FinishBumps");
    world.run_in_window(EDITOR, "Wavy");
    let source = ceramic_asset(world);
    let copy = asset(world, COPY_ID);
    assert_eq!(copy.appearance_metadata, source.appearance_metadata);
    assert_eq!(copy.appearance_preview, source.appearance_preview);
}

#[then("the editor shows Generic appearance controls")]
fn generic_controls(world: &mut ApplicationWorld) {
    generic_fields(world);
    world.run_in_window(EDITOR, "Reflectivity");
    let metadata = asset(world, COPY_ID).appearance_metadata.unwrap();
    assert!(metadata.ceramic.is_none());
    let effects = metadata.generic_effects.unwrap();
    assert!(effects.bump.enabled);
    assert!(effects.rough_surface.unwrap().bump_image.is_none());
}

#[then("the preview resembles the source appearance")]
fn preview_matches_source(world: &mut ApplicationWorld) {
    let before = world.last_capture.take().unwrap();
    let after = capture_framebuffer(&mut world.app().window.native);
    let count = preview_pixels(&before, EDITOR)
        .zip(preview_pixels(&after, EDITOR))
        .filter(|(before, after)| before != after)
        .count();
    assert_eq!(count, 0, "converted surface changed {count} preview pixels");
}

#[test]
fn ceramic_copies_have_independent_names_and_survive_saving() {
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    asset_browser_shared::open_tool(&mut world, BROWSER);
    applied_ceramic(&mut world);
    choose_duplicate(&mut world);
    opens_duplicate(&mut world);
    choose_generic_duplicate(&mut world);
    separate_window(&mut world, "Default Ceramic(2)".into());
    preview_matches_source(&mut world);
    rename(&mut world, "Converted Matte".into());
    apply(&mut world);
    let path = world.save_part_to_file(document, "appearance-copies.opd");
    let saved = DocumentPackage::decode(&std::fs::read(path).unwrap()).unwrap();
    let assets = saved.restore_assets().unwrap();
    let names: Vec<_> = assets
        .appearances()
        .ordered()
        .map(|asset| asset.display_name())
        .collect();
    assert_eq!(
        names,
        ["Default Ceramic", "Default Ceramic(1)", "Converted Matte"]
    );
    assert_eq!(
        assets.appearance_metadata(CERAMIC_ID),
        assets.appearance_metadata(COPY_ID)
    );
    let generic = assets
        .appearance_metadata("default-ceramic-copy-2")
        .unwrap();
    assert!(generic.ceramic.is_none());
    assert!(generic.generic_effects.as_ref().unwrap().bump.enabled);
}

fn duplicate_menu_item(world: &mut ApplicationWorld, label: &str) {
    click_in(world, EDITOR, "Createappearance");
    select_menu_item(world, label);
}

fn asset(world: &mut ApplicationWorld, id: &str) -> oblikovati_gui::GuiAssetChoice {
    world
        .state()
        .browser
        .unwrap()
        .appearances
        .into_iter()
        .find(|asset| asset.id == id && asset.document)
        .unwrap_or_else(|| panic!("expected owned appearance {id}"))
}
