// SPDX-License-Identifier: GPL-2.0-only
//! Check tab backgrounds while the active workspace is outside the visible strip.
use super::super::{ApplicationWorld, capture_framebuffer};

fn tab_background(world: &mut ApplicationWorld, name: &str) -> [u8; 4] {
    let tab = world.expect_tab(name);
    let point = [(tab.minimum[0] - 3.) as usize, tab.center()[1] as usize];
    let image = capture_framebuffer(&mut world.app().window.native);
    let index = (point[1] * image.extent[0] as usize + point[0]) * 4;
    image.rgba[index..index + 4].try_into().unwrap()
}

#[test]
fn manual_scroll_does_not_highlight_an_inactive_document() {
    let mut world = ApplicationWorld::new();
    for _ in 0..16 {
        world.add_part();
    }
    let presentation = world.add_template(oblikovati_gui::GuiTemplateKind::Presentation);
    let active_name = world.document_name(presentation);
    world.pump(Vec::new());
    for _ in 0..=16 {
        if world.tab_run("Home").is_some() {
            break;
        }
        world.click_text("<");
        world.pump(Vec::new());
    }
    world.hover([900, 500]);
    world.pump(Vec::new());
    assert!(world.tab_run(&active_name).is_none());
    let reference = tab_background(&mut world, "Home");
    for id in world.parts.clone() {
        let name = world.document_name(id);
        if world.tab_run(&name).is_some() {
            assert_eq!(
                tab_background(&mut world, &name),
                reference,
                "{name}: expected inactive background"
            );
        }
    }
    assert_eq!(world.active_id(), presentation);
    world.save_capture("tab-active-offscreen");
    for _ in 0..=16 {
        if world.tab_run(&active_name).is_some() {
            break;
        }
        world.click_text(">");
        world.pump(Vec::new());
    }
    world.hover([900, 500]);
    world.pump(Vec::new());
    assert_ne!(
        tab_background(&mut world, &active_name),
        reference,
        "active tab must regain its selected background"
    );
    assert_eq!(world.active_id(), presentation);
    world.save_capture("tab-active-visible");
    for _ in 0..=16 {
        if world.tab_run("Home").is_some() {
            break;
        }
        world.click_text("<");
        world.pump(Vec::new());
    }
    let part = world.parts[0];
    world.click_tab(part);
    world.hover([900, 500]);
    world.pump(Vec::new());
    assert_eq!(world.active_id(), part);
    let name = world.document_name(part);
    assert_ne!(
        tab_background(&mut world, &name),
        reference,
        "clicked tab must become active"
    );
    for id in world.parts.clone().into_iter().filter(|id| *id != part) {
        let name = world.document_name(id);
        if world.tab_run(&name).is_some() {
            assert_eq!(
                tab_background(&mut world, &name),
                reference,
                "{name}: expected inactive after click"
            );
        }
    }
    world.save_capture("tab-active-clicked");
    world.close();
}
