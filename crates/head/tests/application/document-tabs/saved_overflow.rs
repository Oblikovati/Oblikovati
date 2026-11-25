// SPDX-License-Identifier: GPL-2.0-only
//! Use the twelve published Parts and Presentation to overflow the tab strip.
use super::super::saved_tabs;
use super::*;
use oblikovati_api::types::{DOCUMENT_PART, DOCUMENT_PRESENTATION};

#[given(
    "the user opens `Window01.opd` through `Window12.opd` from `fixtures/cad/document-tabs-native/overflow` in order"
)]
fn open_saved_overflow_parts(world: &mut ApplicationWorld) {
    for number in 1..=12 {
        let path = format!("fixtures/cad/document-tabs-native/overflow/Window{number:02}.opd");
        let id = saved_tabs::open_disposable_tab(world, &path, DOCUMENT_PART);
        world.parts.push(id);
    }
    assert!(
        world
            .state()
            .documents
            .iter()
            .map(|row| row.id)
            .eq(world.parts.iter().copied())
    );
    world.other_document = world.parts.first().copied();
}

#[given(
    "`fixtures/cad/document-creation-native-candidate/Metric/Standard Presentation.ord` is open"
)]
fn open_saved_presentation(world: &mut ApplicationWorld) {
    saved_tabs::open_disposable_tab(
        world,
        "fixtures/cad/document-creation-native-candidate/Metric/Standard Presentation.ord",
        DOCUMENT_PRESENTATION,
    );
}

#[given(
    "the tab for `fixtures/cad/document-tabs-native/overflow/Window01.opd` is outside the visible portion of the strip"
)]
fn first_saved_part_is_hidden(world: &mut ApplicationWorld) {
    assert!(world.tab_run("Window01").is_none());
}

#[given("a presentation is active while earlier tabs are out of view")]
fn presentation_active(world: &mut ApplicationWorld) {
    let presentation = if world.state().document_type == Some(DOCUMENT_PRESENTATION) {
        world.active_id()
    } else {
        world.add_template(oblikovati_gui::GuiTemplateKind::Presentation)
    };
    world.previous_document = Some(presentation);
    world.pump(Vec::new());
    for _ in 0..=MANY_DOCUMENTS {
        if world.tab_run("Home").is_some() {
            break;
        }
        world.click_text("<");
        world.pump(Vec::new());
    }
    world.expect_tab("Home");
    world.click_text(">");
    world.pump(Vec::new());
    assert!(world.tab_run("Home").is_none(), "expected Home out of view");
}
