// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the published Generic appearance editing flow.
use super::{ApplicationWorld, diagnostics, material_browser_scenarios::search_in_browser};
use cucumber::{given, then, when};
use oblikovati_viewport::{ViewportCapture, ViewportPhysicalKey};

const BROWSER: &str = "Appearance Browser";
const EDITOR: &str = "###appearance-editor";
const INITIAL: &str = "Default Generic";
const ID: &str = "default-generic";
const BLUE: [u8; 3] = [0, 128, 192];

#[when("I choose New Generic Appearance from the create-appearance menu")]
#[given("a new generic appearance is open in the Appearance Editor")]
fn create_generic(world: &mut ApplicationWorld) {
    click_in(world, BROWSER, "Createappearance");
    world.click_text("New Generic Appearance");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry(EDITOR).is_some());
}

#[then(expr = "a selected appearance named {string} is added to Document Appearances")]
fn created_in_document(world: &mut ApplicationWorld, name: String) {
    let asset = owned_asset(world);
    assert!(asset.document);
    assert_eq!(asset.name, name);
    let shown = world.window_text(BROWSER);
    let name = super::world::squeezed(&name);
    assert!(shown.matches(&name).count() >= 2, "{shown}");
}

#[then(expr = "a separate Appearance Editor opens for {string}")]
fn separate_window(world: &mut ApplicationWorld, name: String) {
    assert!(diagnostics::window_geometry(EDITOR).is_some());
    let title = format!("AppearanceEditor—{}", super::world::squeezed(&name));
    world.run_in_window(EDITOR, &title);
}

#[then("the Appearance Browser remains open")]
fn browser_open(_: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry(BROWSER).is_some());
}

#[then("the editor shows a shaded preview and an editable appearance name")]
fn preview_and_name(world: &mut ApplicationWorld) {
    world.run_in_window(EDITOR, "Name");
    world.run_in_window(EDITOR, "DefaultGeneric");
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let mut shades: Vec<_> = preview_pixels(&capture, EDITOR)
        .map(|pixel| pixel[0])
        .collect();
    shades.sort_unstable();
    shades.dedup();
    assert!(shades.len() > 10, "expected a shaded preview");
}

#[then("the Generic section shows Color, Image, Image Fade, Glossiness and Highlights")]
fn generic_fields(world: &mut ApplicationWorld) {
    let shown = world.window_text(EDITOR);
    for field in ["Color", "Image", "ImageFade", "Glossiness", "Highlights"] {
        assert!(shown.contains(field), "{field}: {shown}");
    }
}

#[then("OK and Apply are disabled before edits")]
#[then("OK and Apply are disabled until another edit")]
fn clean_dialog(world: &mut ApplicationWorld) {
    for label in ["OK", "Apply"] {
        assert!(world.run_in_window(EDITOR, label).alpha < u8::MAX);
    }
}

#[when("I open its Color control")]
fn open_color(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(EDITOR).unwrap();
    let button = world
        .frame
        .runs
        .iter()
        .find(|run| {
            let [x, y] = run.center();
            run.text.starts_with("Color")
                && x > position[0]
                && x < position[0] + size[0]
                && y > position[1]
                && y < position[1] + size[1]
        })
        .unwrap();
    world.click(diagnostics::pointer_pixels(button.center()));
    world.pump(Vec::new());
}

#[then("a color dialog shows basic colors and custom color slots")]
fn color_palette(world: &mut ApplicationWorld) {
    world.run_in_window("Color", "Basiccolors");
    world.run_in_window("Color", "Customcolors");
}

#[then("the dialog offers Define Custom Colors, OK and Cancel")]
fn color_buttons(world: &mut ApplicationWorld) {
    for text in ["DefineCustomColors", "OK", "Cancel"] {
        world.run_in_window("Color", text);
    }
}

#[when("I select the blue swatch with RGB values 0, 128 and 192")]
fn choose_blue(world: &mut ApplicationWorld) {
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let (position, size) = diagnostics::window_geometry("Color").unwrap();
    let top = world.run_in_window("Color", "Basiccolors").maximum[1].ceil() as usize;
    let bottom = world.run_in_window("Color", "Customcolors").minimum[1] as usize;
    let start = position[0].ceil() as usize;
    let end = (position[0] + size[0]).floor() as usize;
    let point = (top..bottom)
        .find_map(|y| {
            (start..end).find_map(|x| {
                (capture_rgb(&capture, x, y)[..3] == BLUE).then_some([x as i32 + 3, y as i32 + 3])
            })
        })
        .expect("expected the RGB 0, 128, 192 swatch in the basic palette");
    world.click(point);
}

#[when("I confirm the color dialog")]
fn confirm_color(world: &mut ApplicationWorld) {
    click_in(world, "Color", "OK");
    assert!(diagnostics::window_geometry("Color").is_none());
}

#[then("the Color control shows RGB values 0, 128 and 192")]
fn rgb_label(world: &mut ApplicationWorld) {
    assert!(world.window_text(EDITOR).contains("Color0,128,192"));
}

#[then("the editor preview sphere appears blue")]
fn blue_editor_preview(world: &mut ApplicationWorld) {
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    assert!(preview_pixels(&capture, EDITOR).any(is_blue));
}

fn can_apply(world: &mut ApplicationWorld) {
    assert_eq!(world.run_in_window(EDITOR, "Apply").alpha, u8::MAX);
}

#[given("its color is RGB 0, 128, 192")]
fn blue_draft(world: &mut ApplicationWorld) {
    open_color(world);
    choose_blue(world);
    confirm_color(world);
}

#[when(expr = "I change its name to {string}")]
fn rename(world: &mut ApplicationWorld, name: String) {
    let label = world.run_in_window(EDITOR, "Name");
    world.click(diagnostics::pointer_pixels([
        label.minimum[0] - 50.,
        label.center()[1],
    ]));
    world.key(ViewportPhysicalKey::A, true);
    world.type_text(&name);
    world.pump(Vec::new());
}

#[when("I choose Apply")]
fn apply(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "Apply");
}

#[then(expr = "the Appearance Editor remains open with {string} in its title")]
fn applied_title(world: &mut ApplicationWorld, title: String) {
    separate_window(world, title.clone());
    let asset = owned_asset(world);
    assert_eq!(asset.name, title);
    assert_eq!(asset.swatch, Some(BLUE));
}

#[when("I close the Appearance Editor")]
fn close_editor(world: &mut ApplicationWorld) {
    let (position, size) = diagnostics::window_geometry(EDITOR).unwrap();
    let close = [position[0] + size[0] - 13., position[1] + 10.];
    world.click(diagnostics::pointer_pixels(close));
    world.pump(Vec::new());
}

#[when(expr = "I search the Appearance Browser for {string}")]
fn search_browser(world: &mut ApplicationWorld, query: String) {
    assert!(diagnostics::window_geometry(EDITOR).is_none());
    search_in_browser(world, BROWSER, &query);
}

#[then(expr = "the document results show {string} with a blue preview")]
fn search_result(world: &mut ApplicationWorld, name: String) {
    assert_eq!(owned_asset(world).name, name);
    let caption = document_appearance_caption(world, "CustomOcean").clone();
    let capture = super::capture_framebuffer(&mut world.app().window.native);
    let [x, y] = diagnostics::pointer_pixels(caption.minimum).map(|v| v as usize);
    assert!(pixels_in(&capture, [x, y - 61], [x + 76, y - 5]).any(is_blue));
}

fn document_appearance_caption<'a>(
    world: &'a ApplicationWorld,
    prefix: &str,
) -> &'a diagnostics::RenderedRun {
    let heading = world.run_in_window(BROWSER, "Documentassets").center()[1];
    let boundary = world.run_in_window(BROWSER, "Alllibraries").center()[1];
    let (position, size) = diagnostics::window_geometry(BROWSER).unwrap();
    world
        .frame
        .runs
        .iter()
        .find(|run| {
            let [x, y] = run.center();
            run.text.starts_with(prefix)
                && y > heading
                && y < boundary
                && x > position[0]
                && x < position[0] + size[0]
        })
        .unwrap_or_else(|| panic!("expected appearance {prefix:?} in the document region"))
}

#[then("its type is Generic and its category is Uncategorized")]
fn result_metadata(world: &mut ApplicationWorld) {
    click_in(world, BROWSER, "Documentview");
    world.click_text("Thumbnail View");
    world.click_text("List View");
    world.escape();
    for column in ["Generic", "Uncategorized"] {
        world.run_in_window(BROWSER, column);
    }
}

#[then("the library results report no match")]
fn no_library_results(world: &mut ApplicationWorld) {
    world.run_in_window(BROWSER, "Nomatchinglibraryassets");
}

#[given("a document appearance is open in the Appearance Editor")]
fn open_document_appearance(world: &mut ApplicationWorld) {
    create_generic(world);
}

#[given("it has unapplied appearance edits")]
fn pending_edit(world: &mut ApplicationWorld) {
    blue_draft(world);
    assert_ne!(owned_asset(world).swatch, Some(BLUE));
    can_apply(world);
}

#[then("a Save Edits question offers Yes, No and Cancel")]
fn close_question(world: &mut ApplicationWorld) {
    for name in ["Yes", "No", "Cancel"] {
        world.run_in_window("Save Edits", name);
    }
}

#[when("I choose Cancel in the question")]
fn keep_editing(world: &mut ApplicationWorld) {
    click_in(world, "Save Edits", "Cancel");
    assert!(diagnostics::window_geometry("Save Edits").is_none());
}

#[then("the Appearance Editor remains open")]
fn editor_stays_open(world: &mut ApplicationWorld) {
    separate_window(world, INITIAL.into());
}

#[then("its pending edits remain visible")]
fn pending_visible(world: &mut ApplicationWorld) {
    rgb_label(world);
    blue_editor_preview(world);
    can_apply(world);
    assert_ne!(owned_asset(world).swatch, Some(BLUE));
}

fn owned_asset(world: &mut ApplicationWorld) -> oblikovati_gui::GuiAssetChoice {
    world
        .state()
        .browser
        .unwrap()
        .appearances
        .into_iter()
        .find(|asset| asset.id == ID && asset.appearance_preview.unwrap().document_owned)
        .expect("expected the document-owned generic appearance")
}

fn click_in(world: &mut ApplicationWorld, window: &str, text: &str) {
    let at = world.run_in_window(window, text).center();
    world.click(diagnostics::pointer_pixels(at));
    world.pump(Vec::new());
}

fn preview_pixels<'a>(
    capture: &'a ViewportCapture,
    window: &str,
) -> impl Iterator<Item = [u8; 4]> + 'a {
    let (pos, size) = diagnostics::window_geometry(window).unwrap();
    let center = (pos[0] + size[0] / 2.) as usize;
    pixels_in(
        capture,
        [center - 42, pos[1] as usize + 35],
        [center + 42, pos[1] as usize + 120],
    )
}

fn pixels_in(
    capture: &ViewportCapture,
    low: [usize; 2],
    high: [usize; 2],
) -> impl Iterator<Item = [u8; 4]> + '_ {
    (low[1]..high[1]).flat_map(move |y| (low[0]..high[0]).map(move |x| capture_rgb(capture, x, y)))
}

fn is_blue([r, g, b, _]: [u8; 4]) -> bool {
    r < 70 && g > r.saturating_add(35) && b > g.saturating_add(25)
}

fn capture_rgb(capture: &ViewportCapture, x: usize, y: usize) -> [u8; 4] {
    let index = (y * capture.extent[0] as usize + x) * 4;
    capture.rgba[index..index + 4].try_into().unwrap()
}

#[given("a generic document appearance is open in the Appearance Editor")]
fn open_generic_document_appearance(world: &mut ApplicationWorld) {
    create_generic(world);
}

#[when("I expand Information")]
fn expand_information(world: &mut ApplicationWorld) {
    click_in(world, EDITOR, "Information");
}

#[then("I see Name, Description and Keywords fields")]
fn information_fields(world: &mut ApplicationWorld) {
    let heading = world.run_in_window(EDITOR, "Information").center()[1];
    assert!(world.run_in_window(EDITOR, "Description").center()[1] > heading);
    assert!(world.run_in_window(EDITOR, "Keywords").center()[1] > heading);
    assert!(world.frame.runs.iter().any(|run| {
        run.text == "Name"
            && run.center()[1] > heading
            && run.center()[1] < world.run_in_window(EDITOR, "Description").center()[1]
    }));
}

#[then("I see the appearance Type and Source")]
fn information_type_and_source(world: &mut ApplicationWorld) {
    world.run_in_window(EDITOR, "Type:Generic");
    world.run_in_window(EDITOR, "Source:Document");
}

#[then("the appearance name is also shown above the property sections")]
fn name_above_sections(world: &mut ApplicationWorld) {
    let heading = world.run_in_window(EDITOR, "Information");
    let name = world.run_in_window(EDITOR, "DefaultGeneric");
    assert!(name.center()[1] < heading.center()[1]);
}

#[test]
fn document_appearance_information_survives_reopening_the_part() {
    let mut world = ApplicationWorld::new();
    let document = world.add_part();
    super::asset_browser_shared::open_tool(&mut world, BROWSER);
    create_generic(&mut world);
    expand_information(&mut world);
    let text = [
        ("Description", "Used on machined parts"),
        ("Keywords", "blue, machined"),
    ];
    for (label, value) in text {
        let field = world.run_in_window(EDITOR, label);
        world.click(diagnostics::pointer_pixels([
            field.minimum[0] - 50.,
            field.center()[1],
        ]));
        world.type_text(value);
        world.pump(Vec::new());
    }
    can_apply(&mut world);
    apply(&mut world);
    clean_dialog(&mut world);
    let applied = owned_asset(&mut world).appearance_metadata.unwrap();
    assert_eq!(applied.description, text[0].1);
    assert_eq!(applied.keywords, text[1].1);
    let path = world.save_part_to_file(document, "appearance-information.opd");
    close_editor(&mut world);
    world.reopen_application();
    world.open_fixture(&path);
    assert_eq!(owned_asset(&mut world).appearance_metadata, Some(applied));
    super::asset_browser_shared::open_tool(&mut world, BROWSER);
    let tile = document_appearance_caption(&world, "DefaultGeneric").center();
    world.click(diagnostics::pointer_pixels(tile));
    world.pump(Vec::new());
    click_in(&mut world, BROWSER, "Editappearance");
    assert!(diagnostics::window_geometry(EDITOR).is_some());
    if !world.window_text(EDITOR).contains("Description") {
        expand_information(&mut world);
    }
    let current = world.window_text(EDITOR);
    for (_, value) in text {
        assert!(
            current.contains(&super::world::squeezed(value)),
            "{current}"
        );
    }
    clean_dialog(&mut world);
}

mod effects;
