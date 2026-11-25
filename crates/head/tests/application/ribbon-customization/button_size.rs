// SPDX-License-Identifier: GPL-2.0-only
//! Match Shell's caption and rendered icon with the main-panel sizing choice.
use super::super::ribbon_appearance::icon_quads::{icon_images_inside, icon_quads_inside};
use super::*;

const RIBBON: &str = "##application-ribbon";

#[given(
    "`fixtures/cad/blank-metric-part.opd` is open with the full 3D Model ribbon in Normal appearance"
)]
fn open_part_with_normal_ribbon(world: &mut ApplicationWorld) {
    super::super::ribbon_panel_expansion::open_blank_metric_part(world);
    assert!(world.window_text(RIBBON).contains("Modify"));
    shell_standard(world);
}

#[given("\"Shell\" is a standard-sized button in the 3D Model \"Modify\" main panel")]
#[given("Shell uses the Standard button size in the Modify panel")]
#[given("Shell is displayed at Standard button size in the Modify panel")]
fn shell_starts_standard(world: &mut ApplicationWorld) {
    shell_standard(world);
    let chamfer = world.run_in_window(RIBBON, "Chamfer").center();
    world.observed_view = Some(diagnostics::pointer_pixels(chamfer));
}

#[when("the user right-clicks \"Shell\" and opens \"Button Size\"")]
fn open_button_size(world: &mut ApplicationWorld) {
    let shell = world.run_in_window(RIBBON, "Shell");
    world.right_click(diagnostics::pointer_pixels(shell.center()));
    world.click_text("Button Size");
    world.expect_text("Standard");
}

#[then("\"Standard\" is selected and \"Large\" is available")]
fn standard_selected(world: &mut ApplicationWorld) {
    assert!(world.checked("Standard"));
    assert!(!world.checked("Large"));
    assert!(world.run_is_enabled("Large"));
}

#[when("the user chooses Large from the Shell button size menu")]
#[when("the user changes the Shell button size to Large")]
fn choose_large(world: &mut ApplicationWorld) {
    open_button_size(world);
    world.click_text("Large");
}

#[when("the user chooses Standard from that menu")]
#[when("the user selects Standard for Shell")]
fn select_standard(world: &mut ApplicationWorld) {
    world.click_text("Standard");
}

#[then("\"Large\" is selected")]
#[then("Large is checked")]
fn large_selected(world: &mut ApplicationWorld) {
    assert!(world.checked("Large"));
    assert!(!world.checked("Standard"));
}

#[when("the user selects the Tools tab and then returns to 3D Model")]
fn switch_tabs(world: &mut ApplicationWorld) {
    world.click_text("Tools");
    world.run_in_window(RIBBON, "Measure");
    world.click_text("3D Model");
    assert_eq!(world.state().document, world.previous_document);
}

#[then("its button size menu marks Large as selected")]
fn retained_menu_selection(world: &mut ApplicationWorld) {
    open_button_size(world);
    large_selected(world);
}

#[then("\"Shell\" is displayed with a larger icon and its label below")]
#[then("Shell is still shown as a large button")]
#[then("Shell appears as a large labeled button")]
fn shell_large(world: &mut ApplicationWorld) {
    let shell = world.run_in_window(RIBBON, "Shell");
    let [x, _] = shell.center();
    let y = shell.minimum[1];
    let icons = icon_images_inside(([x - 26., y - 52.], [x + 26., y + 1.]));
    let (_, texture) = icons
        .iter()
        .find(|((first, last), _)| {
            (36. ..45.).contains(&(last[0] - first[0]))
                && (first[0]..last[0]).contains(&x)
                && (0. ..9.).contains(&(y - last[1]))
        })
        .unwrap_or_else(|| panic!("expected a large Shell icon above its caption: {icons:?}"));
    world.shell_button_texture = Some(*texture);
}

#[then("the other commands in the panel reflow")]
fn chamfer_reflows(world: &mut ApplicationWorld) {
    let chamfer = world.run_in_window(RIBBON, "Chamfer").center();
    let after = diagnostics::pointer_pixels(chamfer);
    let before = world.observed_view.unwrap();
    assert!(after[0] > before[0] + 55);
}

#[then("\"Shell\" returns to a small icon with its label beside it")]
#[then("Shell returns to its standard button presentation")]
#[then("Shell returns to its standard labeled presentation")]
fn shell_standard(world: &mut ApplicationWorld) {
    let shell = world.run_in_window(RIBBON, "Shell");
    let [_, y] = shell.center();
    let x = shell.minimum[0];
    let icons = icon_quads_inside(([x - 34., y - 15.], [x + 1., y + 15.]));
    assert!(
        icons.iter().any(|(first, last)| {
            (16. ..22.).contains(&(last[0] - first[0]))
                && (first[1]..last[1]).contains(&y)
                && (0. ..10.).contains(&(x - last[0]))
        }),
        "expected a small Shell icon left of its caption: {icons:?}"
    );
}

#[when("the user selects Text Off from Ribbon Appearance")]
fn select_text_off(world: &mut ApplicationWorld) {
    set_appearance(world, "Text Off");
}

#[when("the user restores Normal ribbon appearance")]
fn restore_normal(world: &mut ApplicationWorld) {
    set_appearance(world, "Normal");
}

fn set_appearance(world: &mut ApplicationWorld, mode: &str) {
    let tab = world.run_in_window(RIBBON, "3DModel");
    world.right_click(diagnostics::pointer_pixels(tab.center()));
    world.click_text("Ribbon Appearance");
    world.click_text(mode);
}

#[then("command labels are hidden and the Modify panel title remains visible")]
fn modify_labels_hidden(world: &mut ApplicationWorld) {
    let ribbon = world.window_text(RIBBON);
    assert!(ribbon.contains("Modify"));
    for label in ["Shell", "Hole", "Fillet", "Chamfer"] {
        assert!(!ribbon.contains(label), "unexpected {label} caption");
    }
}

#[then("Shell appears as a large icon without its label")]
fn large_without_label(world: &mut ApplicationWorld) {
    shell_icon_center(world);
    assert!(!world.window_text(RIBBON).contains("Shell"));
}

#[when("the user opens the Shell button size submenu")]
fn reopen_from_icon(world: &mut ApplicationWorld) {
    world.right_click(diagnostics::pointer_pixels(shell_icon_center(world)));
    world.click_text("Button Size");
}

fn shell_icon_center(world: &ApplicationWorld) -> [f32; 2] {
    let (position, size) = diagnostics::window_geometry(RIBBON).unwrap();
    let maximum = std::array::from_fn(|axis| position[axis] + size[axis]);
    let shell_texture = world
        .shell_button_texture
        .expect("expected a recorded Shell icon");
    let (first, last) = icon_images_inside((position, maximum))
        .into_iter()
        .find_map(|(rect, texture)| (texture == shell_texture).then_some(rect))
        .expect("expected the same large Shell image with text hidden");
    assert!((36. ..45.).contains(&(last[0] - first[0])));
    std::array::from_fn(|axis| (first[axis] + last[axis]) / 2.)
}
