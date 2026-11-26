// SPDX-License-Identifier: GPL-2.0-only
//! Exercise the Loft command through native ribbon and viewport input.
use super::*;
use cucumber::{given, then, when};
use oblikovati_gui::GuiLoftElement;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchEntityId};
use oblikovati_viewport::ViewportNavigation;

#[given("the part contains no visible sketches or solid bodies")]
fn empty_part_has_no_loft_inputs(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert!(!browser.sketches.iter().any(|sketch| sketch.visible));
    assert!(browser.bodies.is_empty());
}

#[when("Loft is clicked")]
fn open_loft(world: &mut ApplicationWorld) {
    world.click_text("Loft");
    world.pump(Vec::new());
    assert!(diagnostics::window_geometry("Loft").is_some());
}

#[then("a tip says Loft requires a body or at least two visible sketches")]
fn loft_geometry_tip(world: &mut ApplicationWorld) {
    let text = world::squeezed(&world.window_text("Loft"));
    assert!(
        text.contains("Loftrequiresabodyoratleasttwovisiblesketches"),
        "{text}"
    );
}

#[then("the tip closes")]
fn loft_tip_is_dismissed(world: &mut ApplicationWorld) {
    assert!(diagnostics::window_geometry("Loft").is_none());
    assert!(world.state().loft_sections.is_empty());
}

#[given("Loft is open and Sketch1 contains a visible circle and center point")]
fn loft_is_open_on_the_parallel_circles(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    assert!(
        browser
            .sketches
            .iter()
            .any(|sketch| sketch.name == "Sketch1" && sketch.visible)
    );
    open_loft(world);
    crate::HeadWindow::navigate_view(
        &mut world.app().window.native,
        ViewportNavigation::Home { margin: 0.25 },
    )
    .unwrap();
}

#[when("the center point and then the circle edge are clicked")]
fn pick_center_then_circumference(world: &mut ApplicationWorld) {
    for point in [[0., 0., 0.], [1., 0., 0.]] {
        let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, point).unwrap();
        world.click(pixel);
        world.pump(Vec::new());
    }
}

#[then("the Sections list contains Point Sketch1 and Sketch1 as separate entries")]
fn loft_lists_both_sketch_entities(world: &mut ApplicationWorld) {
    let sections = world.state().loft_sections;
    assert_eq!(sections.len(), 2, "{sections:?}");
    assert!(
        sections
            .iter()
            .all(|section| section.sketch == DocumentSketchId(1))
    );
    assert_eq!(
        sections[0].element,
        GuiLoftElement::Point(SketchEntityId(2))
    );
    assert_eq!(
        sections[1].element,
        GuiLoftElement::Profile(SketchEntityId(3))
    );
    let text = world::squeezed(&world.window_text("Loft"));
    assert!(text.contains("SectionsPointSketch1Sketch1"), "{text}");
}
