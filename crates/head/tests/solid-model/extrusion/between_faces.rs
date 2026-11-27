// SPDX-License-Identifier: GPL-2.0-only
use super::super::super::{diagnostics, world::ApplicationWorld};
use super::{activate_from, feature_count, request_rows};
use crate::HeadWindow;
use cucumber::{given, then, when};
use oblikovati_gui::GuiExtrudeReferenceExtent;
use oblikovati_model::document_sketch::DocumentSketchId;
use oblikovati_viewport::ViewportCamera;

const FIXTURE: &str = "fixtures/cad/extrusion/midplane-circle-between-faces-candidate.opd";
const SKETCH: DocumentSketchId = DocumentSketchId(2);
const CAPS: [f32; 2] = [0., 2.];

#[given(
    "a writable copy of `fixtures/cad/extrusion/midplane-circle-between-faces-candidate.opd` is open"
)]
fn open_midplane_part(world: &mut ApplicationWorld) {
    let source = super::super::super::scenario_root().join(FIXTURE);
    let copy = world
        .scratch_folder()
        .join("midplane-circle-between-faces.opd");
    std::fs::copy(source, &copy).unwrap();
    world.open_fixture(&copy);
    assert_eq!(feature_count(world), 2, "extrusion and fixed midplane");
    let browser = world.state().browser.unwrap();
    assert_eq!(browser.bodies.len(), 1);
    assert_eq!(browser.bodies[0].faces, 6);
    assert_eq!(super::super::direction::preview_depth(world), CAPS);
}

#[given("the user makes Sketch2 visible and starts Extrude with its circle selected")]
fn start_from_circle(world: &mut ApplicationWorld) {
    let browser = world.state().browser.unwrap();
    let sketch = browser
        .sketches
        .iter()
        .find(|row| row.id == SKETCH)
        .unwrap();
    assert_eq!(sketch.name, "Sketch2");
    if !sketch.visible {
        world.right_click(diagnostics::pointer_pixels(world.run("Sketch2").center()));
        world.click_text("Visibility");
    }
    world.click_text("Sketch2");
    world.click_text("Extrude");
    world.pump(Vec::new());
    let draft = world.state().extrude_draft_preview.unwrap_or_else(|| {
        panic!(
            "expected a numeric preview: {:?}",
            world.app().window.native.command_notice
        );
    });
    assert_eq!((draft.sketch, draft.profile), (SKETCH, 0));
    world.run_in_window("Extrude", "1Profile");
}

#[given("the profile plane lies between the block's top and bottom planar faces")]
fn profile_on_midplane(world: &mut ApplicationWorld) {
    let (center, _) = profile_geometry(world);
    assert_eq!(center[2], 1.);
}

#[given("the projected profile lies inside both face boundaries")]
fn projection_inside_faces(world: &mut ApplicationWorld) {
    let (center, radius) = profile_geometry(world);
    let scene = &world.app().window.scene_triangles;
    for cap in CAPS {
        for axis in 0..2 {
            let (min, max) = scene
                .iter()
                .filter(|triangle| triangle.color != crate::part_scene::NEW_SOLID_PREVIEW)
                .flat_map(|triangle| triangle.positions)
                .filter(|point| point[2] == cap)
                .map(|point| f64::from(point[axis]))
                .fold((f64::INFINITY, f64::NEG_INFINITY), |(min, max), value| {
                    (min.min(value), max.max(value))
                });
            assert!(min < center[axis] - radius && center[axis] + radius < max);
        }
    }
}

#[when("the user selects the first face as From and the second as To")]
fn supply_faces(world: &mut ApplicationWorld) {
    activate_from(world);
    assert!(!world.run_is_enabled("OK"));
    let [from, _] = request_rows(world)
        .try_into()
        .expect("two unset references");
    super::super::viewport_interaction::click_at(world, diagnostics::pointer_pixels(from));
    let from_key = pick_cap(world, 0., -20.);
    assert!(world.state().extrude_draft_preview.is_none());
    let [to] = request_rows(world).try_into().expect("one unset reference");
    super::super::viewport_interaction::click_at(world, diagnostics::pointer_pixels(to));
    let to_key = pick_cap(world, 2., 20.);
    assert_ne!(from_key, to_key);
    let draft = world.state().extrude_draft_preview.unwrap_or_else(|| {
        panic!(
            "expected a bounded preview: {:?}",
            world.app().window.native.command_notice
        );
    });
    assert_eq!(
        draft.reference_extent,
        Some(GuiExtrudeReferenceExtent::FromTo {
            from: from_key,
            to: to_key,
        })
    );
}

#[then("the extrusion preview starts on the first face and terminates on the second")]
fn spans_the_two_caps(world: &mut ApplicationWorld) {
    assert!(world.state().extrude_preview_active);
    assert!(world.run_is_enabled("OK"));
    assert_eq!(feature_count(world), 2, "preview must not author a feature");
    let mesh = preview_mesh(world);
    assert!(
        mesh.iter()
            .flat_map(|triangle| triangle.positions)
            .all(|point| { (CAPS[0]..=CAPS[1]).contains(&point[2]) })
    );
    for cap in CAPS {
        assert!(
            mesh.iter()
                .any(|triangle| triangle.positions.iter().all(|point| point[2] == cap))
        );
    }
}

#[then("neither cap lies on the original sketch plane")]
fn independent_of_sketch_plane(world: &mut ApplicationWorld) {
    let (center, _) = profile_geometry(world);
    assert!(preview_mesh(world).iter().all(|triangle| {
        !triangle
            .positions
            .iter()
            .all(|point| f64::from(point[2]) == center[2])
    }));
}

fn profile_geometry(world: &mut ApplicationWorld) -> ([f64; 3], f64) {
    let document = world.active_id();
    let workspace = world.app().owner.documents().workspace();
    let snapshot = workspace.session(document).unwrap().snapshot();
    let sketch = snapshot.sketches.get(SKETCH).unwrap().geometry();
    let [circle] = sketch.circles() else {
        panic!("expected one circle on the middle plane");
    };
    let point = sketch.point(circle.center).unwrap().position;
    let plane = snapshot.resolve_sketch_plane(SKETCH).unwrap();
    (plane.to_model(point).unwrap(), circle.radius)
}

fn pick_cap(world: &mut ApplicationWorld, z: f64, offset: f64) -> String {
    let ([x, y, _], _) = profile_geometry(world);
    let camera = ViewportCamera {
        eye: [x, y, z + offset],
        target: [x, y, z],
        scale: 2.,
        ..Default::default()
    };
    world.app().window.set_view_camera(camera).unwrap();
    world.pump(Vec::new());
    let pixel = world
        .app()
        .window
        .native
        .view
        .model_pixel([x, y, z])
        .unwrap();
    super::super::viewport_interaction::click_at(world, pixel);
    world.pump(Vec::new());
    let state = world.state();
    let [(key, _)] = state.selected_face_appearances.as_slice() else {
        panic!(
            "expected one face at Z={z}: {:?}",
            state.selected_face_appearances
        );
    };
    key.clone()
}

fn preview_mesh(world: &mut ApplicationWorld) -> Vec<oblikovati_viewport::ViewportTriangle> {
    let triangles: Vec<_> = world
        .app()
        .window
        .scene_triangles
        .iter()
        .filter(|triangle| triangle.color == crate::part_scene::NEW_SOLID_PREVIEW)
        .copied()
        .collect();
    assert!(!triangles.is_empty());
    triangles
}
