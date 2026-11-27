// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::{capture_framebuffer, diagnostics};
use super::super::acceptance::{feature_count, feature_distance};
use super::super::edit_properties::{start_pinned_edit, start_published_edit};
use super::super::viewport_interaction::click_at;
use super::ApplicationWorld;
use cucumber::{given, then, when};
use oblikovati_gui::GuiAuthoringContext;

#[given("a feature Properties panel shows the accepted feature and its source sketch")]
fn open_source_breadcrumb(world: &mut ApplicationWorld) {
    start_published_edit(world);
    assert_edit_breadcrumb(world);
}

fn assert_edit_breadcrumb(world: &mut ApplicationWorld) {
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
    assert!(world.state().extrude_preview_active);
    assert_eq!(breadcrumb_underlines(world), [true, false]);
}

#[when("the user chooses the source sketch in the breadcrumb")]
fn choose_breadcrumb_sketch(world: &mut ApplicationWorld) {
    let (_, sketch) = breadcrumb_segments(world);
    let point = diagnostics::pointer_pixels([sketch.maximum[0] - 8., sketch.center()[1]]);
    click_at(world, point);
    world.pump(Vec::new());
}

#[then("the source sketch becomes the active edit context")]
fn source_is_active(world: &mut ApplicationWorld) {
    let state = world.state();
    assert_eq!(state.authoring, GuiAuthoringContext::Sketch);
    let sketch = &state.browser.unwrap().sketches[0];
    assert_eq!(state.active_sketch, Some(sketch.id));
    assert!(!state.extrude_preview_active);
    assert_eq!(feature_count(world), 1);
    world.run("Finish Sketch");
}

#[then("the breadcrumb distinguishes the current sketch from its parent feature")]
fn source_receives_active_accent(world: &mut ApplicationWorld) {
    assert_eq!(breadcrumb_underlines(world), [false, true]);
}

fn breadcrumb_segments(
    world: &mut ApplicationWorld,
) -> (diagnostics::RenderedRun, diagnostics::RenderedRun) {
    let source = world.state().browser.unwrap().sketches[0].name.clone();
    let (position, size) = diagnostics::window_geometry("Extrude").expect("expected Extrude");
    let inside = |run: &diagnostics::RenderedRun| {
        let center = run.center();
        (0..2).all(|axis| {
            center[axis] >= position[axis] && center[axis] <= position[axis] + size[axis]
        })
    };
    let combined = format!("Extrusion1›{source}");
    if let Some(row) = world
        .frame
        .runs
        .iter()
        .find(|run| run.text == combined && inside(run))
    {
        return (row.clone(), row.clone());
    }
    let feature = world.run_in_window("Extrude", "Extrusion1›").clone();
    let sketch = world
        .frame
        .runs
        .iter()
        .find(|run| {
            inside(run)
                && run.text == source
                && run.minimum[0] >= feature.minimum[0]
                && (run.center()[1] - feature.center()[1]).abs() < 12.
        })
        .unwrap_or_else(|| panic!("expected breadcrumb sketch {source:?} beside Extrusion1›"))
        .clone();
    (feature, sketch)
}

fn breadcrumb_underlines(world: &mut ApplicationWorld) -> [bool; 2] {
    let (feature, sketch) = breadcrumb_segments(world);
    let capture = capture_framebuffer(&mut world.app().window.native);
    let [width, height] = capture.extent.map(|pixels| pixels as usize);
    let accent =
        unsafe { imgui::sys::igGetColorU32_Col(imgui::sys::ImGuiCol_NavHighlight as i32, 1.) }
            .to_le_bytes();
    [
        [feature.minimum[0] + 8., feature.maximum[1]],
        [sketch.maximum[0] - 8., sketch.maximum[1]],
    ]
    .map(|anchor| {
        let [x, top] = diagnostics::pointer_pixels(anchor).map(|pixel| pixel as usize);
        (top..(top + 8).min(height)).any(|y| {
            let at = (y * width + x) * 4;
            capture.rgba.get(at..at + 3) == Some(&accent[..3])
        })
    })
}

#[test]
fn native_extrude_source_breadcrumb_closes_after_finish_sketch() {
    let mut world = ApplicationWorld::new();
    start_pinned_edit(&mut world);
    assert_edit_breadcrumb(&mut world);
    choose_breadcrumb_sketch(&mut world);
    source_is_active(&mut world);
    source_receives_active_accent(&mut world);
    world.finish_sketch();
    assert_eq!(world.state().authoring, GuiAuthoringContext::Part);
    assert!(diagnostics::window_geometry("Extrude").is_none());
    assert_eq!(feature_count(&mut world), 1);
    assert_eq!(feature_distance(&mut world), 8.);
}
