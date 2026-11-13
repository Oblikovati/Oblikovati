// SPDX-License-Identifier: GPL-2.0-only
//! Sketch Parameters panel visibility through the native ribbon.
use super::super::diagnostics;
use super::*;
use cucumber::{given, then, when};

#[given("the Sketch tab is active")]
fn sketch_tab(world: &mut ApplicationWorld) {
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    // These scenarios inspect commands inside expanded panels, not collapsed flyouts.
    world.app().window.native.view.resize([2400, 900]).unwrap();
    world.pump(Vec::new());
    world.expect_text("Finish Sketch");
}

#[then("Parameters is unchecked in the panel list")]
fn parameters_unchecked_in_list(world: &mut ApplicationWorld) {
    assert!(!world.checked(PARAMETERS));
}

#[when("the user selects Parameters")]
fn select_parameters(world: &mut ApplicationWorld) {
    world.click_text(PARAMETERS);
    world.pump(Vec::new());
}

fn panel_caption<'a>(
    world: &'a ApplicationWorld,
    name: &str,
    y: f32,
) -> &'a super::super::diagnostics::RenderedRun {
    world
        .frame
        .runs
        .iter()
        .find(|run| run.text == name && (run.center()[1] - y).abs() < 8.)
        .unwrap_or_else(|| {
            panic!(
                "expected ribbon caption {name:?} near {y}: {:?}",
                world.text()
            )
        })
}

#[then("the Parameters panel appears after Format and before Exit")]
fn parameters_order(world: &mut ApplicationWorld) {
    let format = world.run("Format");
    let y = format.center()[1];
    let parameters = panel_caption(world, PARAMETERS, y);
    let exit = panel_caption(world, "Exit", y);
    assert!(
        format.center()[0] < parameters.center()[0] && parameters.center()[0] < exit.center()[0]
    );
}

#[then("the panel displays a large parameter symbol")]
fn parameter_icon(world: &mut ApplicationWorld) {
    let y = world.run("Format").center()[1];
    let caption = panel_caption(world, PARAMETERS, y);
    let center = [caption.center()[0] as i32, caption.center()[1] as i32 - 45];
    let capture = super::super::super::capture_framebuffer(&mut world.app().window.native);
    let width = capture.extent[0] as usize;
    let colors: std::collections::BTreeSet<_> = ((center[1] - 12)..=(center[1] + 12))
        .flat_map(|row| {
            let rgba = &capture.rgba;
            ((center[0] - 12)..=(center[0] + 12)).map(move |column| {
                let at = (row as usize * width + column as usize) * 4;
                [rgba[at], rgba[at + 1], rgba[at + 2]]
            })
        })
        .collect();
    assert!(
        colors.len() > 3,
        "expected a visible parameter icon near {center:?}"
    );
}

#[then("the Parameters panel is hidden again")]
fn parameters_hidden_again(world: &mut ApplicationWorld) {
    super::parameters_hidden(world);
}

#[test]
fn sketch_parameters_panel_opens_the_live_editor_without_finishing_the_sketch() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let before = world.edited_geometry();
    let create = world.run("Create").center();
    world.right_click(diagnostics::pointer_pixels(create));
    world.click_text("Show Panels");
    select_parameters(&mut world);
    let y = world.run("Format").center()[1];
    let caption = panel_caption(&world, PARAMETERS, y);
    let button = [caption.center()[0] as i32, caption.center()[1] as i32 - 35];
    world.click(button);
    // A narrow ribbon opens the panel flyout before the command can be selected.
    if diagnostics::window_geometry("Parameters").is_none() {
        let command = world.run(PARAMETERS).center();
        world.click(diagnostics::pointer_pixels([command[0], command[1] - 20.]));
    }
    assert!(diagnostics::window_geometry("Parameters").is_some());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch
    );
    assert!(world.edited_geometry().same_recipe(&before));
    for (label, value) in [("Name", "size"), ("Equation", "10 mm")] {
        let label = world.run_in_window("Parameters", label);
        world.click([(label.minimum[0] - 20.) as i32, label.center()[1] as i32]);
        world.type_text(value);
    }
    let add = world.run_in_window("Parameters", "Add").center();
    world.click(diagnostics::pointer_pixels(add));
    assert!(
        world
            .state()
            .parameters
            .iter()
            .any(|parameter| parameter.name == "size" && parameter.expression == "10 mm")
    );
    let cancel = world.run_in_window("Parameters", "Cancel").center();
    world.click(diagnostics::pointer_pixels(cancel));
    assert!(diagnostics::window_geometry("Parameters").is_none());
    world.undo();
    assert!(
        world
            .state()
            .parameters
            .iter()
            .all(|parameter| parameter.name != "size")
    );
    world.redo();
    assert!(
        world
            .state()
            .parameters
            .iter()
            .any(|parameter| parameter.name == "size" && parameter.expression == "10 mm")
    );
    world.close();
}

#[test]
fn scaled_sketch_ribbon_exposes_the_rectangle_tool() {
    use crate::HeadWork;
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    let scale = oblikovati_model::ui_scale::UiScaleOptions::new(1.5, 1.).unwrap();
    world.app().owner.set_ui_scale_from_gui(scale).unwrap();
    world.pump(Vec::new());
    world.save_capture("scaled-sketch-ribbon");
    world.expect_text("Create");
    let label = world.run("Create").center();
    world.click(diagnostics::pointer_pixels([label[0], label[1] - 35.]));
    world.expect_text("Rectangle");
    world.click_text("Rectangle");
    assert!(world.app().owner.placement_active());
    world.close();
}
