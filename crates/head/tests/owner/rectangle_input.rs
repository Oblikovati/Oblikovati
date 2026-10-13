// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input::deliver;
use super::*;
use crate::{SketchPlacementTool, placement_test_inputs::*};
use oblikovati_model::{
    document_sketch::SketchPlacement, history::DocumentSketchCreation, sketch::SketchPlane,
};
fn rectangle_owner(
    kind: SketchPlacementTool,
) -> (
    TestOwner,
    u64,
    oblikovati_model::document_sketch::DocumentSketchId,
) {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    let sketch = owner
        .create_sketch_for_edit(
            document,
            &DocumentSketchCreation {
                name: "Rectangle".into(),
                placement: SketchPlacement::Fixed(SketchPlane::xy()),
                orientation: None,
                project_origin: false,
            },
        )
        .unwrap();
    owner
        .start_sketch_placement(document, sketch, kind)
        .unwrap();
    (owner, document, sketch)
}
#[test]
fn rectangle_click_and_drag_use_one_commit_and_restart_for_every_kind() {
    for kind in [
        SketchPlacementTool::Rectangle,
        SketchPlacementTool::CenterRectangle,
        SketchPlacementTool::ThreePointRectangle,
    ] {
        let clicked = place_rectangle(kind, false);
        let dragged = place_rectangle(kind, true);
        assert!(clicked.same_recipe(&dragged));
    }
}
fn place_rectangle(
    kind: SketchPlacementTool,
    drag: bool,
) -> oblikovati_model::document_sketch::DocumentSketches {
    let (mut owner, document, sketch) = rectangle_owner(kind);
    let before = owner.gui_command_state().unwrap();
    deliver(&mut owner, &[button([0, 0], true)]);
    if !drag {
        deliver(&mut owner, &[button([0, 0], false)]);
    }
    let endpoint = if kind == SketchPlacementTool::ThreePointRectangle {
        [10, 0]
    } else {
        [10, 5]
    };
    deliver(&mut owner, &[motion(endpoint)]);
    assert_eq!(owner.gui_command_state().unwrap(), before);
    if kind == SketchPlacementTool::ThreePointRectangle {
        assert!(owner.placement_preview().is_none());
    } else {
        assert_eq!(
            owner.placement_preview().unwrap().points.len(),
            if kind == SketchPlacementTool::CenterRectangle {
                5
            } else {
                4
            }
        );
    }
    deliver(&mut owner, &[button(endpoint, !drag)]);
    if kind == SketchPlacementTool::ThreePointRectangle {
        assert_eq!(owner.gui_command_state().unwrap(), before);
        deliver(&mut owner, &[motion([3, 5]), button([3, 5], true)]);
    }
    let snapshot = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot();
    let sketches = snapshot.sketches.clone();
    assert_eq!(
        sketches.get(sketch).unwrap().geometry().lines().len(),
        if kind == SketchPlacementTool::CenterRectangle {
            6
        } else {
            4
        }
    );
    assert!(owner.placement_active());
    assert!(owner.placement_preview().is_none());
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .get(sketch)
            .unwrap()
            .geometry()
            .points()
            .is_empty()
    );
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .snapshot()
            .sketches
            .same_recipe(&sketches)
    );
    sketches
}
#[test]
fn rectangle_failed_points_keep_the_current_stage_and_escape_cancels() {
    let (mut owner, document, _) = rectangle_owner(SketchPlacementTool::ThreePointRectangle);
    let before = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .history_entries();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            button([0, 0], true),
        ],
    );
    assert!(matches!(
        owner.take_command_notice(),
        Some(DesktopCommandNotice::PlacementFailed { .. })
    ));
    deliver(
        &mut owner,
        &[button([10, 0], true), motion([5, 0]), button([5, 0], true)],
    );
    assert!(matches!(
        owner.take_command_notice(),
        Some(DesktopCommandNotice::PlacementFailed { .. })
    ));
    assert_eq!(
        owner
            .documents()
            .workspace()
            .session(document)
            .unwrap()
            .history_entries(),
        before
    );
    deliver(&mut owner, &[motion([5, 4])]);
    assert_eq!(
        owner.placement_preview().unwrap().points,
        vec![[0., 0.], [10., 0.], [10., 4.], [0., 4.]]
    );
    deliver(
        &mut owner,
        &[key(oblikovati_viewport::ViewportKey::Cancel, true)],
    );
    assert!(!owner.placement_active());
    assert!(owner.placement_preview().is_none());
}
