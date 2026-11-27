// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    geometric_tools::geometric_owner, line_tools::line_tool_geometry, slot_input::deliver,
};
use crate::placement_test_inputs::*;
#[test]
fn point_tool_preview_uses_the_same_authored_role_and_geometry_as_commit() {
    for center in [false, true] {
        let (mut owner, document, sketch) = geometric_owner(&[], false);
        owner.sketch_memory.center_point = center;
        owner
            .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Point)
            .unwrap();
        deliver(&mut owner, &[motion([30, 40])]);
        assert!(
            line_tool_geometry(&owner, document, sketch)
                .points()
                .is_empty()
        );
        let oblikovati_viewport::ViewportSketchPreview::Resolved {
            geometry, members, ..
        } = owner.sketch_preview().unwrap()
        else {
            panic!("expected Point preview resolved through model authoring");
        };
        assert_eq!(members.len(), 1);
        assert_eq!(geometry.is_center_point(members[0]), center);
        deliver(
            &mut owner,
            &[button([30, 40], true), button([30, 40], false)],
        );
        assert_eq!(*geometry, line_tool_geometry(&owner, document, sketch));
    }
}

#[test]
fn point_format_toggle_refreshes_the_stationary_pointer_preview_before_commit() {
    let (mut owner, document, sketch) = geometric_owner(&[], false);
    owner.sketch_memory.center_point = false;
    owner
        .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Point)
        .unwrap();
    deliver(&mut owner, &[motion([30, 40])]);
    for center in [true, false, true] {
        owner.toggle_center_point_format_from_gui().unwrap();
        let oblikovati_viewport::ViewportSketchPreview::Resolved {
            geometry, members, ..
        } = owner.sketch_preview().unwrap()
        else {
            panic!("expected resolved preview after role toggle");
        };
        assert_eq!(geometry.is_center_point(members[0]), center);
    }
    deliver(
        &mut owner,
        &[button([30, 40], true), button([30, 40], false)],
    );
    let committed = line_tool_geometry(&owner, document, sketch);
    assert!(committed.is_center_point(committed.points()[0].id));
}

#[test]
fn point_tool_preview_uses_the_same_acquired_vertex_as_click() {
    use oblikovati_model::sketch::PlanarConstraint;
    let (mut owner, document, sketch) = geometric_owner(&[[30., 40.]], false);
    let target = line_tool_geometry(&owner, document, sketch).points()[0].id;
    owner
        .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Point)
        .unwrap();
    deliver(&mut owner, &[motion([31, 40])]);
    let oblikovati_viewport::ViewportSketchPreview::Resolved {
        geometry, members, ..
    } = owner.sketch_preview().unwrap()
    else {
        panic!("expected an acquired Point preview");
    };
    assert!(geometry.constraints().iter().any(|item| matches!(item.relation, PlanarConstraint::Coincident { points } if points == [target, members[0]])));
    deliver(
        &mut owner,
        &[button([31, 40], true), button([31, 40], false)],
    );
    assert_eq!(*geometry, line_tool_geometry(&owner, document, sketch));
}
