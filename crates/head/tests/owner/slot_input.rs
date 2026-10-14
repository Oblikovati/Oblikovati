// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{HeadPresentation, HeadWindow, placement_test_inputs::*};
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::SketchPlane};
use oblikovati_viewport::{ViewportButton, ViewportEvent, ViewportInput, ViewportKey};
#[derive(Default)]
pub(super) struct PlanarInputWindow {
    pub(super) fail: bool,
    pub(super) edge_on: bool,
}
impl HeadWindow for PlanarInputWindow {
    fn sketch_point(&self, pixel: [i32; 2], _: SketchPlane) -> Result<Option<[f64; 2]>, String> {
        if self.fail {
            return Err("named projection failure".into());
        }
        Ok((!self.edge_on).then_some(pixel.map(f64::from)))
    }
    fn events(&mut self, _: usize) -> Result<Vec<ViewportEvent>, String> {
        Ok(Vec::new())
    }
    fn present(&mut self) -> Result<HeadPresentation, String> {
        Ok(HeadPresentation::Completed)
    }
    fn wait(&mut self, _: Option<Duration>) -> Result<bool, String> {
        Ok(false)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
fn slot_owner() -> (TestOwner, u64, DocumentSketchId) {
    let (mut owner, awake) = owner();
    let document = create(&mut owner, &awake);
    Client::new(owner.document_host())
        .sketch()
        .create(&CreateSketchArgs {
            plane: "XY".into(),
            ..Default::default()
        })
        .unwrap();
    let sketches = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .sketches();
    assert_eq!(sketches.len(), 1);
    let sketch = sketches[0].id();
    owner.start_slot_placement(document, sketch, 2.).unwrap();
    (owner, document, sketch)
}
fn count(owner: &TestOwner, document: u64, sketch: DocumentSketchId) -> usize {
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
        .len()
}
pub(super) fn deliver(owner: &mut TestOwner, events: &[ViewportEvent]) {
    owner
        .process_window_inputs(events, &PlanarInputWindow::default())
        .unwrap();
}
#[test]
fn slot_click_click_and_drag_release_share_geometry_and_one_history_command() {
    for drag in [false, true] {
        let (mut owner, document, sketch) = slot_owner();
        let before = count(&owner, document, sketch);
        let history = owner.gui_command_state().unwrap();
        deliver(&mut owner, &[button([0, 0], true), motion([0, 0])]);
        assert!(owner.placement_preview().is_none());
        if !drag {
            deliver(&mut owner, &[button([0, 0], false)]);
        }
        deliver(&mut owner, &[motion([10, 0])]);
        assert_eq!(count(&owner, document, sketch), before);
        assert_eq!(owner.gui_command_state().unwrap(), history);
        let preview = owner.placement_preview().unwrap().clone();
        deliver(&mut owner, &[button([10, 0], !drag)]);
        assert_eq!(count(&owner, document, sketch), before + 6);
        assert!(owner.placement_active());
        assert!(owner.placement_preview().is_none());
        verify_slot_centers(&owner, document, sketch, &preview);
        deliver(&mut owner, &[key(ViewportKey::Undo, true)]);
        assert_eq!(count(&owner, document, sketch), before);
        assert!(!owner.placement_active());
    }
}
fn verify_slot_centers(
    owner: &TestOwner,
    document: u64,
    sketch: DocumentSketchId,
    preview: &oblikovati_model::sketch::SketchPlacementRecipe,
) {
    let geometry = owner
        .documents()
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry();
    let points = &geometry.points()[geometry.points().len() - preview.points.len()..];
    for (actual, expected) in points.iter().zip(&preview.points) {
        for (a, b) in actual.position.into_iter().zip(expected) {
            assert!((a - b).abs() < 1e-8);
        }
    }
}
#[test]
fn slot_sub_slop_release_does_not_commit_and_release_itself_can_start_a_drag() {
    let (mut owner, document, sketch) = slot_owner();
    let before = count(&owner, document, sketch);
    deliver(&mut owner, &[button([0, 0], true), button([4, 0], false)]);
    assert_eq!(count(&owner, document, sketch), before);
    assert!(owner.placement_preview().is_none());
    deliver(&mut owner, &[button([10, 0], true), button([20, 0], false)]);
    assert_eq!(count(&owner, document, sketch), before + 6);
    deliver(&mut owner, &[button([30, 0], true), button([40, 0], false)]);
    assert_eq!(count(&owner, document, sketch), before + 12);
}
#[test]
fn slot_cancel_lifecycle_and_navigation_never_turn_release_into_a_commit() {
    for event in [
        key(ViewportKey::Cancel, true),
        ViewportEvent::Focus(false),
        ViewportEvent::Mapped(false),
        ViewportEvent::CloseRequested,
    ] {
        let (mut owner, document, sketch) = slot_owner();
        let before = count(&owner, document, sketch);
        deliver(
            &mut owner,
            &[
                button([0, 0], true),
                motion([10, 0]),
                event,
                button([10, 0], false),
            ],
        );
        assert!(!owner.placement_active());
        assert_eq!(count(&owner, document, sketch), before);
    }
    verify_navigation_disarms_slot();
}
fn verify_navigation_disarms_slot() {
    for navigation in [
        ViewportKey::Pan,
        ViewportKey::Zoom,
        ViewportKey::Orbit,
        ViewportKey::Home,
        ViewportKey::Previous,
    ] {
        let (mut owner, document, sketch) = slot_owner();
        let before = count(&owner, document, sketch);
        deliver(
            &mut owner,
            &[
                button([0, 0], true),
                key(navigation, true),
                motion([10, 0]),
                button([10, 0], false),
                key(navigation, false),
            ],
        );
        assert_eq!(count(&owner, document, sketch), before);
        deliver(&mut owner, &[button([10, 0], true)]);
        assert_eq!(count(&owner, document, sketch), before + 6);
    }
}
#[test]
fn slot_projection_failure_and_invalid_second_point_keep_a_retryable_first_point() {
    let (mut owner, document, sketch) = slot_owner();
    let before = count(&owner, document, sketch);
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
    let failed = PlanarInputWindow {
        fail: true,
        edge_on: false,
    };
    owner
        .process_window_inputs(&[motion([10, 0])], &failed)
        .unwrap();
    deliver(&mut owner, &[button([10, 0], false)]);
    assert_eq!(count(&owner, document, sketch), before);
    deliver(&mut owner, &[button([10, 0], true)]);
    assert_eq!(count(&owner, document, sketch), before + 6);
}
#[test]
fn slot_rejects_wrong_document_bad_width_or_transaction_and_cancels_on_external_edit() {
    let (mut owner, document, sketch) = slot_owner();
    for width in [0., -1., f64::NAN, f64::INFINITY] {
        assert!(owner.start_slot_placement(document, sketch, width).is_err());
    }
    assert!(
        owner
            .start_slot_placement(document + 1, sketch, 2.)
            .is_err()
    );
    assert!(
        owner
            .start_slot_placement(document, DocumentSketchId(999), 2.)
            .is_err()
    );
    deliver(&mut owner, &[button([0, 0], true), motion([10, 0])]);
    super::commands::dirty_property(&mut owner, document);
    assert!(owner.placement_preview().is_none());
    assert!(!owner.placement_active());
    deliver(&mut owner, &[]);
    assert!(!owner.placement_active());
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "External".into(),
        })
        .unwrap();
    assert!(owner.start_slot_placement(document, sketch, 2.).is_err());
}
#[test]
fn inactive_owner_and_edge_on_planes_do_not_create_geometry() {
    let (mut owner, document, sketch) = slot_owner();
    let before = count(&owner, document, sketch);
    let edge = PlanarInputWindow {
        edge_on: true,
        ..Default::default()
    };
    owner
        .process_window_inputs(&[button([0, 0], true), button([10, 0], false)], &edge)
        .unwrap();
    assert!(owner.placement_preview().is_none());
    deliver(
        &mut owner,
        &[
            ViewportEvent::Expose,
            ViewportEvent::Focus(true),
            ViewportEvent::Mapped(true),
            key(ViewportKey::Cancel, false),
        ],
    );
    deliver(
        &mut owner,
        &[
            ViewportEvent::Input(ViewportInput::Button {
                button: ViewportButton::Middle,
                pressed: true,
                position: [0, 0],
                modifiers: Default::default(),
            }),
            ViewportEvent::Input(ViewportInput::Wheel {
                notches: 1,
                position: [0, 0],
            }),
        ],
    );
    deliver(
        &mut owner,
        &[
            key(ViewportKey::Cancel, true),
            button([0, 0], true),
            button([10, 0], true),
        ],
    );
    assert_eq!(count(&owner, document, sketch), before);
}

#[test]
fn slot_input_is_forwarded_by_script_owner_and_projectionless_windows_report_failure() {
    let (mut owner, document, sketch) = slot_owner();
    owner
        .process_window_inputs(&[button([0, 0], true)], &ProjectionlessWindow)
        .unwrap();
    assert!(
        owner
            .take_command_notice()
            .unwrap()
            .message()
            .contains("expected a window with sketch projection")
    );
    let before = count(&owner, document, sketch);
    let engine =
        oblikovati_script::IsolatedLuaEngine::new(&std::env::current_exe().unwrap(), Vec::new());
    let mut desktop =
        crate::DesktopScripts::new(owner, engine, oblikovati_script::ScriptLimits::gui());
    desktop
        .process_window_inputs(
            &[button([0, 0], true), button([10, 0], false)],
            &PlanarInputWindow::default(),
        )
        .unwrap();
    let owner = desktop.into_document_owner();
    assert_eq!(count(&owner, document, sketch), before + 6);
}

#[test]
fn preview_packet_contains_current_plane_and_owned_recipe() {
    let (mut owner, _, _) = slot_owner();
    assert!(owner.sketch_preview().is_none());
    owner
        .process_window_inputs(
            &[button([0, 0], true), button([0, 0], false), motion([10, 0])],
            &PlanarInputWindow::default(),
        )
        .unwrap();
    let packet = owner.sketch_preview().unwrap();
    assert_eq!(packet.plane(), SketchPlane::xy());
    let oblikovati_viewport::ViewportSketchPreview::Recipe { ref recipe, .. } = packet else {
        panic!("expected standalone recipe");
    };
    assert_eq!(recipe, owner.placement_preview().unwrap());
    owner
        .process_window_inputs(
            &[key(ViewportKey::Cancel, true)],
            &PlanarInputWindow::default(),
        )
        .unwrap();
    assert!(owner.sketch_preview().is_none());
    assert!(packet.member_count() > 0);
}
