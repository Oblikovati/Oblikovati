// SPDX-License-Identifier: GPL-2.0-only
//! Drive block placement through the native browser and sketch viewport.
use super::{ApplicationWorld, diagnostics, sketch_constrain::segment_color_changed};
use crate::HeadSketchWork;
use crate::HeadWindow;
use oblikovati_viewport::{ViewportKey, ViewportPhysicalKey};

#[test]
fn member_color_is_drawn_after_block_explosion() {
    use oblikovati_api::types::{Color, OVERRIDE_COLOR_SOURCE, SketchEntityFormat};
    let mut world = ApplicationWorld::with_asset_tools_closed();
    world.start_empty_sketch();
    world.draw_rectangle();
    world.escape();
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let line = world.edited_geometry().lines()[0].id;
    world
        .app()
        .owner
        .document_commands(edit.document)
        .run("Prepare red block", |snapshot, _| {
            let mut geometry = snapshot
                .sketches
                .get(edit.sketch)
                .unwrap()
                .geometry()
                .clone();
            let color = Color {
                r: 255,
                g: 0,
                b: 0,
                opacity: 1.0,
                source: OVERRIDE_COLOR_SOURCE,
            };
            geometry
                .set_entity_format(
                    line,
                    SketchEntityFormat {
                        override_color: color,
                        ..Default::default()
                    },
                )
                .unwrap();
            snapshot
                .sketches
                .replace_geometry(edit.sketch, geometry)
                .unwrap();
            snapshot
                .sketches
                .capture_block_definition("Red", edit.sketch, &[line])
                .unwrap();
            Ok::<_, String>(())
        })
        .unwrap();
    world.pump(Vec::new());
    let name = world.run("Red").center();
    world.right_click(diagnostics::pointer_pixels(name));
    world.click_text("Place Block");
    world.click([1100, 550]);
    world.escape();
    let instance = world.state().browser.unwrap().block_instances[0].id.0;
    let cyan = Color {
        r: 0,
        g: 255,
        b: 255,
        opacity: 1.0,
        source: OVERRIDE_COLOR_SOURCE,
    };
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .sketch()
        .set_block_instance_color(&oblikovati_api::wire::SetBlockInstanceColorArgs {
            sketch_index: 0,
            instance_id: instance,
            color: Some(Box::new(cyan)),
        })
        .unwrap();
    world.pump(Vec::new());
    let row = world.run("Red:1").center();
    world.right_click(diagnostics::pointer_pixels(row));
    world.click_text("Explode");
    world.pump(Vec::new());
    let geometry = world.edited_geometry();
    let exploded_line = geometry.lines().last().unwrap();
    let exploded = exploded_line.id;
    assert_eq!(
        geometry
            .entity_format(exploded)
            .unwrap()
            .unwrap()
            .override_color
            .r,
        255
    );
    let points = exploded_line
        .endpoints
        .map(|id| geometry.point(id).unwrap().position);
    let midpoint = [
        (points[0][0] + points[1][0]) / 2.0,
        (points[0][1] + points[1][1]) / 2.0,
    ];
    let pixel = world
        .app()
        .window
        .sketch_pixel(edit.plane, midpoint)
        .unwrap();
    let image = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(
        red_pixel_near(&image, pixel),
        "expected a red exploded member near {pixel:?}"
    );
    world.command_key(ViewportPhysicalKey::Z, ViewportKey::Undo);
    let image = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(
        cyan_pixel_near(&image, pixel),
        "expected the restored cyan instance near {pixel:?}"
    );
    world.command_key(ViewportPhysicalKey::Y, ViewportKey::Redo);
    let image = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(
        red_pixel_near(&image, pixel),
        "expected the red member after redo"
    );
    world.close();
}

fn cyan_pixel_near(image: &oblikovati_viewport::ViewportCapture, pixel: [i32; 2]) -> bool {
    (-3..=3).any(|dy| {
        (-3..=3).any(|dx| {
            let x = (pixel[0] + dx) as usize;
            let y = (pixel[1] + dy) as usize;
            let index = (y * image.extent[0] as usize + x) * 4;
            image.rgba.get(index..index + 3).is_some_and(|rgb| {
                rgb[1] > 150
                    && rgb[2] > 150
                    && u16::from(rgb[1]) > u16::from(rgb[0]) * 2
                    && u16::from(rgb[2]) > u16::from(rgb[0]) * 2
            })
        })
    })
}

fn red_pixel_near(image: &oblikovati_viewport::ViewportCapture, pixel: [i32; 2]) -> bool {
    (-3..=3).any(|dy| {
        (-3..=3).any(|dx| {
            let x = (pixel[0] + dx) as usize;
            let y = (pixel[1] + dy) as usize;
            let index = (y * image.extent[0] as usize + x) * 4;
            image.rgba.get(index..index + 3).is_some_and(|rgb| {
                rgb[0] > 150
                    && u16::from(rgb[0]) > u16::from(rgb[1]) * 2
                    && u16::from(rgb[0]) > u16::from(rgb[2]) * 2
            })
        })
    })
}

#[test]
fn explode_instance_from_browser_menu_and_undo_redo() {
    let mut world = ApplicationWorld::new();
    world.start_empty_sketch();
    world.draw_rectangle();
    world.escape();
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let line = world.edited_geometry().lines()[0].id;
    world
        .app()
        .owner
        .document_commands(edit.document)
        .run("Capture block", |snapshot, _| {
            snapshot
                .sketches
                .capture_block_definition("Rib", edit.sketch, &[line])
        })
        .unwrap();
    world.pump(Vec::new());
    start_block_from_browser(&mut world);
    world.click([1100, 550]);
    world.escape();
    let before = world.edited_geometry().lines().len();
    let row = world.run("Rib:1").center();
    world.right_click(diagnostics::pointer_pixels(row));
    world.click_text("Explode");
    world.pump(Vec::new());
    world.expect_no_text("Rib:1");
    assert_eq!(world.edited_geometry().lines().len(), before + 1);
    assert_eq!(world.state().sketch_block_names, ["Rib"]);
    world.command_key(ViewportPhysicalKey::Z, ViewportKey::Undo);
    world.expect_text("Rib:1");
    assert_eq!(world.edited_geometry().lines().len(), before);
    world.command_key(ViewportPhysicalKey::Y, ViewportKey::Redo);
    world.expect_no_text("Rib:1");
    assert_eq!(world.edited_geometry().lines().len(), before + 1);
    world.close();
}

#[test]
fn exploded_edges_select_independently_of_a_second_grouped_instance() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    world.start_empty_sketch();
    world.draw_rectangle();
    world.escape();
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let mut entities: Vec<u64> = world
        .edited_geometry()
        .lines()
        .iter()
        .map(|line| line.id.0)
        .collect();
    entities.extend(add_two_test_circles(&mut world, edit));
    let args = oblikovati_api::wire::CreateBlockDefinitionArgs {
        name: "Rib".into(),
        source_sketch_index: 0,
        entity_refs: Some(entities),
    };
    oblikovati_api::client::Client::new(world.app().owner.document_host())
        .sketch()
        .create_block_definition(&args)
        .unwrap();
    world.pump(Vec::new());
    start_block_from_browser(&mut world);
    world.click([1100, 550]);
    world.escape();
    let second = world.state().browser.unwrap().block_instances[1].id;
    let row = world.run("Rib:1").center();
    world.right_click(diagnostics::pointer_pixels(row));
    world.click_text("Explode");
    world.pump(Vec::new());
    world.expect_no_text("Rib:1");
    world.expect_text("Rib:2");
    let geometry = world.edited_geometry();
    assert_eq!((geometry.lines().len(), geometry.circles().len()), (4, 2));
    let line = geometry.lines()[0].id;
    let segment = [world.line_point(0, 0.35), world.line_point(0, 0.65)];
    let midpoint = std::array::from_fn(|axis| (segment[0][axis] + segment[1][axis]) / 2);
    world.click(midpoint);
    assert_eq!(
        world.state().sketch_selection.entities,
        [(edit.sketch, line)]
    );
    world.click([1300, 700]);
    let [group_edge, first_circle, second_circle] = grouped_samples(&mut world, edit, second);
    let unselected = super::super::capture_framebuffer(&mut world.app().window.native);
    world.click_text("Rib:2");
    assert_eq!(
        world.state().sketch_selection.entities,
        [(edit.sketch, second)]
    );
    let selected = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(segment_color_changed(&unselected, &selected, group_edge));
    assert!(segment_color_changed(&unselected, &selected, first_circle));
    assert!(segment_color_changed(&unselected, &selected, second_circle));
    assert!(
        !segment_color_changed(&unselected, &selected, segment),
        "selecting the second instance changed the exploded edge"
    );
    world.close();
}

fn grouped_samples(
    world: &mut ApplicationWorld,
    edit: crate::HeadSketchEdit,
    id: oblikovati_model::sketch::SketchEntityId,
) -> [[[i32; 2]; 2]; 3] {
    let snapshot = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap()
        .snapshot();
    let instance = snapshot
        .sketches
        .get(edit.sketch)
        .unwrap()
        .block_instances()
        .iter()
        .find(|instance| instance.id() == id)
        .unwrap()
        .clone();
    let geometry = snapshot
        .sketches
        .block_definitions()
        .iter()
        .find(|definition| definition.name() == instance.definition())
        .unwrap()
        .geometry()
        .clone();
    let endpoints = geometry.lines()[0]
        .endpoints
        .map(|id| geometry.point(id).unwrap().position);
    let edge = endpoints.map(|point| instance.to_sketch(point).unwrap());
    let circles = geometry
        .circles()
        .iter()
        .map(|circle| {
            let center = geometry.point(circle.center).unwrap().position;
            instance
                .to_sketch([center[0] + circle.radius, center[1]])
                .unwrap()
        })
        .collect::<Vec<_>>();
    let pixel = |world: &mut ApplicationWorld, point| {
        world.app().window.sketch_pixel(edit.plane, point).unwrap()
    };
    let a = pixel(world, edge[0]);
    let b = pixel(world, edge[1]);
    let c = pixel(world, circles[0]);
    let d = pixel(world, circles[1]);
    [[a, b], [c, c], [d, d]]
}

fn add_two_test_circles(world: &mut ApplicationWorld, edit: crate::HeadSketchEdit) -> Vec<u64> {
    world
        .app()
        .owner
        .document_commands(edit.document)
        .run("Prepare two circles", |snapshot, ids| {
            let mut geometry = snapshot
                .sketches
                .get(edit.sketch)
                .unwrap()
                .geometry()
                .clone();
            let points: Vec<_> = geometry
                .lines()
                .iter()
                .flat_map(|line| line.endpoints)
                .map(|id| geometry.point(id).unwrap().position)
                .collect();
            let bounds = [0, 1].map(|axis| {
                points
                    .iter()
                    .map(|point| point[axis])
                    .fold([f64::INFINITY, f64::NEG_INFINITY], |[lo, hi], value| {
                        [lo.min(value), hi.max(value)]
                    })
            });
            let radius = (bounds[0][1] - bounds[0][0]).min(bounds[1][1] - bounds[1][0]) / 8.0;
            let mut circles = Vec::new();
            for fraction in [1.0 / 3.0, 2.0 / 3.0] {
                let center = ids.allocate_sketch_entity().unwrap();
                let circle = ids.allocate_sketch_entity().unwrap();
                geometry
                    .add_point(
                        center,
                        [
                            bounds[0][0] + fraction * (bounds[0][1] - bounds[0][0]),
                            (bounds[1][0] + bounds[1][1]) / 2.0,
                        ],
                    )
                    .unwrap();
                geometry.add_circle(circle, center, radius).unwrap();
                circles.push(circle.0);
            }
            snapshot
                .sketches
                .replace_geometry(edit.sketch, geometry)
                .unwrap();
            Ok::<_, String>(circles)
        })
        .unwrap()
}

#[test]
fn place_block_context_menu_repeats_and_escape_cancels() {
    let mut world = ApplicationWorld::with_asset_tools_closed();
    world.start_empty_sketch();
    world.draw_rectangle();
    world.escape();
    let edit = world.app().owner.active_sketch_edit().unwrap();
    let line = world.edited_geometry().lines()[0].id;
    world
        .app()
        .owner
        .document_commands(edit.document)
        .run("Capture block", |snapshot, _| {
            snapshot
                .sketches
                .capture_block_definition("Rib", edit.sketch, &[line])
        })
        .unwrap();
    world.pump(Vec::new());
    start_block_from_browser(&mut world);
    assert!(!world.state().sketch_at_ready);
    verify_preview(&mut world, edit);
    let positions = [[1100, 550], [1150, 600]].map(|pixel| {
        world
            .app()
            .window
            .sketch_point(pixel, edit.plane)
            .unwrap()
            .unwrap()
    });
    world.click([1100, 550]);
    world.expect_text("Rib:1");
    world.click([1150, 600]);
    world.expect_text("Rib:2");
    let instances = world
        .app()
        .owner
        .documents()
        .workspace()
        .session(edit.document)
        .unwrap()
        .snapshot()
        .sketches
        .get(edit.sketch)
        .unwrap()
        .block_instances()
        .to_vec();
    for (instance, expected) in instances.iter().zip(positions) {
        for (actual, expected) in instance.position().into_iter().zip(expected) {
            assert!((actual - expected).abs() < 1e-9);
        }
    }
    world.escape();
    assert!(world.state().sketch_at_ready);
    assert!(world.app().owner.sketch_preview().is_none());
    world.command_key(ViewportPhysicalKey::Z, ViewportKey::Undo);
    world.expect_no_text("Rib:2");
    world.command_key(ViewportPhysicalKey::Y, ViewportKey::Redo);
    world.expect_text("Rib:2");
    start_block_from_browser(&mut world);
    world.escape();
    assert_eq!(world.state().browser.unwrap().block_instances.len(), 2);
    world.close();
}
fn verify_preview(world: &mut ApplicationWorld, edit: crate::HeadSketchEdit) {
    let before = super::super::capture_framebuffer(&mut world.app().window.native);
    world.hover([1100, 550]);
    let preview = world.app().owner.sketch_preview().unwrap();
    assert_eq!(preview.member_count(), 1);
    let oblikovati_viewport::ViewportSketchPreview::Resolved { geometry, .. } = preview else {
        panic!("expected placed block preview geometry");
    };
    let segment = geometry.lines()[0].endpoints.map(|id| {
        let point = geometry.point(id).unwrap().position;
        world.app().window.sketch_pixel(edit.plane, point).unwrap()
    });
    let after = super::super::capture_framebuffer(&mut world.app().window.native);
    assert!(segment_color_changed(&before, &after, segment));
}
fn start_block_from_browser(world: &mut ApplicationWorld) {
    let name = world.run("Rib").center();
    world.right_click(diagnostics::pointer_pixels(name));
    world.click_text("Place Block");
    world.pump(Vec::new());
}
