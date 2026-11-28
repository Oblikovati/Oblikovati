// SPDX-License-Identifier: GPL-2.0-only
//! Shared Emboss desktop BDD helpers for emboss-01..emboss-07.
use super::super::super::*;
use oblikovati_gui::GuiEmbossPreview;
use oblikovati_model::feature::sketch_emboss::EmbossType;
use oblikovati_viewport::ViewportPhysicalKey;
use std::path::Path;

pub(super) fn click_emboss_button(world: &mut ApplicationWorld, label: &str) {
    let pixel = super::super::super::diagnostics::pointer_pixels(
        world.run_in_window("Emboss", label).center(),
    );
    world.click(pixel);
    world.pump(Vec::new());
}

pub(super) fn click_modal_button(world: &mut ApplicationWorld, window: &str, label: &str) {
    let pixel = super::super::super::diagnostics::pointer_pixels(
        world.run_in_window(window, label).center(),
    );
    world.click(pixel);
    world.pump(Vec::new());
}

pub(super) fn focus_and_replace_emboss_field(
    world: &mut ApplicationWorld,
    label: &str,
    value: &str,
) {
    let row = world.run_in_window("Emboss", label).clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry("Emboss").unwrap();
    let field = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        row.center()[1],
    ]);
    world.click(field);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::A, true);
    world.pump(Vec::new());
    world.key(ViewportPhysicalKey::Backspace, false);
    world.pump(Vec::new());
    world.type_text(value);
    world.pump(Vec::new());
}

pub(super) fn choose_emboss_operation(world: &mut ApplicationWorld, popup_text: &str) {
    let label = world.run_in_window("Emboss", "Operation").clone();
    let (origin, size) = super::super::super::diagnostics::window_geometry("Emboss").unwrap();
    let combo = super::super::super::diagnostics::pointer_pixels([
        origin[0] + size[0] * 0.72,
        label.center()[1],
    ]);
    world.click(combo);
    world.pump(Vec::new());
    let item = world
        .frame
        .runs
        .iter()
        .find(|row| world.frame.is_popup_row(row) && row.text == popup_text)
        .unwrap_or_else(|| {
            panic!(
                "expected {popup_text} in Emboss Operation dropdown, saw {}",
                world.text()
            )
        })
        .center();
    world.click(super::super::super::diagnostics::pointer_pixels(item));
    world.pump(Vec::new());
}

pub(super) fn start_emboss(world: &mut ApplicationWorld) {
    if !world.shows("Emboss") {
        let create = world.run("CREATE▼").center();
        world.click(super::super::super::diagnostics::pointer_pixels(create));
        world.pump(Vec::new());
    }
    world.click_text("Emboss");
    world.pump(Vec::new());
    world.pump(Vec::new());
    world.pump(Vec::new());
}

pub(super) fn double_click_emboss1(world: &mut ApplicationWorld) {
    let center = world.run("Emboss1").center();
    world.double_click(super::super::super::diagnostics::pointer_pixels(center));
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert!(
        super::super::super::diagnostics::window_geometry("Emboss").is_some(),
        "expected Emboss panel after double-clicking Emboss1"
    );
}

pub(super) fn click_top_planar_face(world: &mut ApplicationWorld) {
    for candidate in [[3.4, 2.3, 2.0], [3.4, 0.7, 2.0], [0.6, 2.3, 2.0]] {
        let pixel = crate::HeadWindow::model_pixel(&world.app().window.native, candidate).unwrap();
        world.click(pixel);
        world.pump(Vec::new());
        world.pump(Vec::new());
        if world
            .state()
            .emboss_preview
            .as_ref()
            .is_some_and(|preview| preview.request.wrap_face.is_some())
        {
            return;
        }
    }
    panic!(
        "expected clicking top planar face to populate Emboss wrap face, saw {:?}",
        world.state().emboss_preview
    );
}

pub(super) fn scene_bounds(world: &mut ApplicationWorld) -> ([f32; 3], [f32; 3]) {
    let mut lo = [f32::INFINITY; 3];
    let mut hi = [f32::NEG_INFINITY; 3];
    for tri in &world.app().window.scene_triangles {
        for p in tri.positions {
            for i in 0..3 {
                lo[i] = lo[i].min(p[i]);
                hi[i] = hi[i].max(p[i]);
            }
        }
    }
    (lo, hi)
}

fn open_solid_and_start_top_face_sketch(world: &mut ApplicationWorld) {
    let source = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/docking-visible-solid.opd");
    let copy = world.scratch_folder().join("emboss-start.opd");
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("emboss fixture {source:?}: {error}"));
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&copy, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
    let visible_sketches: Vec<_> = world
        .state()
        .browser
        .unwrap()
        .sketches
        .into_iter()
        .filter(|sketch| sketch.visible)
        .map(|sketch| sketch.id)
        .collect();
    for sketch in visible_sketches {
        world
            .app()
            .owner
            .set_sketch_visibility_from_gui(document, sketch, false)
            .unwrap();
    }
    world.pump(Vec::new());

    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    let top_face =
        crate::HeadWindow::model_pixel(&world.app().window.native, [2.0, 1.5, 2.0]).unwrap();
    world.click(top_face);
    world.pump(Vec::new());
    world.pump(Vec::new());
    assert_eq!(
        world.state().authoring,
        oblikovati_gui::GuiAuthoringContext::Sketch,
        "expected sketch started on top planar face"
    );
}

fn project_to_sketch_2d(
    plane: oblikovati_model::sketch::SketchPlane,
    world_pt: [f64; 3],
) -> [f64; 2] {
    let r = plane.recipe();
    let d = [
        world_pt[0] - r.origin[0],
        world_pt[1] - r.origin[1],
        world_pt[2] - r.origin[2],
    ];
    let u = d[0] * r.x_axis[0] + d[1] * r.x_axis[1] + d[2] * r.x_axis[2];
    let v = d[0] * r.y_axis[0] + d[1] * r.y_axis[1] + d[2] * r.y_axis[2];
    [(u * 2.0).round() / 2.0, (v * 2.0).round() / 2.0]
}

fn open_solid_with_top_face_sketch_index(
    world: &mut ApplicationWorld,
) -> (i64, oblikovati_model::sketch::SketchPlane) {
    open_solid_and_start_top_face_sketch(world);
    let plane = world.app().owner.active_sketch_edit().unwrap().plane;
    world.finish_sketch();
    world.pump(Vec::new());
    let sketch_index = (world.state().browser.unwrap().sketches.len() - 1) as i64;
    (sketch_index, plane)
}

pub(super) fn open_solid_with_top_open_line_sketch(world: &mut ApplicationWorld) {
    let source = Path::new(env!("CARGO_MANIFEST_DIR"))
        .join("../../test-utilities/fixtures/docking-visible-solid.opd");
    let copy = world.scratch_folder().join("emboss-open-line.opd");
    std::fs::copy(&source, &copy)
        .unwrap_or_else(|error| panic!("emboss fixture {source:?}: {error}"));
    let document = world
        .app()
        .owner
        .document_host()
        .open_path(&copy, true)
        .unwrap();
    world.pump(Vec::new());
    world.activate_through_tab(document);
    let visible_sketches: Vec<_> = world
        .state()
        .browser
        .unwrap()
        .sketches
        .into_iter()
        .filter(|sketch| sketch.visible)
        .map(|sketch| sketch.id)
        .collect();
    for sketch in visible_sketches {
        world
            .app()
            .owner
            .set_sketch_visibility_from_gui(document, sketch, false)
            .unwrap();
    }
    world.pump(Vec::new());
    world.click_text("Start 2D Sketch");
    world.pump(Vec::new());
    world.click_text("XY Plane");
    world.pump(Vec::new());
    world.finish_sketch();
    world.pump(Vec::new());
    let sketch_index = (world.state().browser.unwrap().sketches.len() - 1) as i64;
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .sketch()
        .add_entity(&oblikovati_api::wire::AddSketchEntityArgs {
            sketch_index,
            kind: "line".into(),
            points: Some(vec![Some(vec![1.0, 1.5]), Some(vec![3.0, 1.5])]),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
}

pub(super) fn open_solid_with_top_circle_sketch(world: &mut ApplicationWorld) {
    let (sketch_index, plane) = open_solid_with_top_face_sketch_index(world);
    let center = project_to_sketch_2d(plane, [2.0, 1.5, 2.0]);
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .sketch()
        .add_entity(&oblikovati_api::wire::AddSketchEntityArgs {
            sketch_index,
            kind: "circle".into(),
            points: Some(vec![Some(center.to_vec())]),
            radius: "5 mm".into(),
            ..Default::default()
        })
        .unwrap();
    world.pump(Vec::new());
}

pub(super) fn prepare_emboss1_raised(world: &mut ApplicationWorld) {
    open_solid_with_top_circle_sketch(world);
    start_emboss(world);
    focus_and_replace_emboss_field(world, "Depth", "1 mm");
    assert_eq!(
        world.state().emboss_preview.unwrap().result,
        GuiEmbossPreview::Ready
    );
    click_emboss_button(world, "OK");
    assert!(super::super::super::diagnostics::window_geometry("Emboss").is_none());
}

pub(super) fn prepare_emboss1_recess(world: &mut ApplicationWorld) {
    prepare_emboss1_raised(world);
    double_click_emboss1(world);
    choose_emboss_operation(world, "EngravefromFace");
    focus_and_replace_emboss_field(world, "Depth", "1 mm");
    assert_eq!(
        world.state().emboss_preview.unwrap().result,
        GuiEmbossPreview::Ready
    );
    click_emboss_button(world, "OK");
    assert!(super::super::super::diagnostics::window_geometry("Emboss").is_none());
}

pub(super) fn prepare_emboss1_wrapped_open(world: &mut ApplicationWorld) {
    prepare_emboss1_recess(world);
    double_click_emboss1(world);
    choose_emboss_operation(world, "EmbossfromFace");
    click_emboss_button(world, "WraptoFace");
    click_top_planar_face(world);
    focus_and_replace_emboss_field(world, "Depth", "1 mm");
    assert_eq!(
        world.state().emboss_preview.unwrap().result,
        GuiEmbossPreview::Ready
    );
    click_emboss_button(world, "OK");
    assert!(super::super::super::diagnostics::window_geometry("Emboss").is_none());
    double_click_emboss1(world);
    let preview = world.state().emboss_preview.unwrap();
    assert_eq!(preview.request.emboss_type, EmbossType::EmbossFromFace);
    assert!(preview.request.wrap_to_face);
    assert!(preview.request.wrap_face.is_some());
}
