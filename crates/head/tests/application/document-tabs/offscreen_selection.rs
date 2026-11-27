// SPDX-License-Identifier: GPL-2.0-only
//! Check tab selection while the active workspace is outside the visible strip.
use super::ApplicationWorld;
use super::narrow_resize::tab_has_active_underline;

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
    assert!(!tab_has_active_underline(&mut world, "Home"));
    for id in world.parts.clone() {
        let name = world.document_name(id);
        if world.tab_run(&name).is_some() {
            assert!(
                !tab_has_active_underline(&mut world, &name),
                "{name}: expected an inactive tab"
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
    assert!(
        tab_has_active_underline(&mut world, &active_name),
        "active tab must regain its underline"
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
    assert!(
        tab_has_active_underline(&mut world, &name),
        "clicked tab must become active"
    );
    for id in world.parts.clone().into_iter().filter(|id| *id != part) {
        let name = world.document_name(id);
        if world.tab_run(&name).is_some() {
            assert!(
                !tab_has_active_underline(&mut world, &name),
                "{name}: expected inactive after click"
            );
        }
    }
    world.save_capture("tab-active-clicked");
    world.close();
}
