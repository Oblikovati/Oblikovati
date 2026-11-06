// SPDX-License-Identifier: GPL-2.0-only
use super::slot_input::deliver;
use super::*;
use crate::placement_test_inputs::*;
use oblikovati_model::{document_sketch::DocumentSketchId, sketch::Sketch2D};
use oblikovati_viewport::{ViewportEvent, ViewportInput, ViewportKey, ViewportPhysicalKey};

pub(super) fn line_key(key: ViewportPhysicalKey) -> ViewportEvent {
    ViewportEvent::Input(ViewportInput::PhysicalKey {
        key,
        layout_key: key,
        command: None,
        pressed: true,
        modifiers: Default::default(),
    })
}
pub(super) fn line_tool_geometry(
    owner: &TestOwner,
    document: u64,
    sketch: DocumentSketchId,
) -> Sketch2D {
    owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .snapshot()
        .sketches
        .get(sketch)
        .unwrap()
        .geometry()
        .clone()
}
pub(super) fn line_owner() -> (TestOwner, u64, DocumentSketchId) {
    let (mut owner, document, sketch) = super::geometric_tools::geometric_owner(&[], false);
    owner
        .place_sketch_tool_from_gui(oblikovati_gui::GuiSketchTool::Line)
        .unwrap();
    (owner, document, sketch)
}
#[test]
fn each_accepted_segment_is_one_history_step_and_only_the_pending_one_stays_private() {
    let (mut owner, document, sketch) = line_owner();
    assert!(!owner.gui_command_state().unwrap().transaction_open);
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            button([50, 20], true),
            button([50, 20], false),
            button([80, 90], true),
            button([80, 90], false),
            motion([120, 150]),
        ],
    );
    assert!(owner.placement_active());
    let accepted = line_tool_geometry(&owner, document, sketch);
    assert_eq!(accepted.lines().len(), 2);
    assert_eq!(accepted.points().len(), 3);
    assert_eq!(
        accepted.lines()[0].endpoints[1],
        accepted.lines()[1].endpoints[0],
        "the chain shares its vertex"
    );
    let oblikovati_viewport::ViewportSketchPreview::Resolved {
        geometry: preview, ..
    } = owner.sketch_preview().unwrap()
    else {
        panic!("expected inferred Line preview")
    };
    assert_eq!(preview.lines().len(), 3);
    assert_eq!(preview.points()[3].position, [120., 150.]);
    deliver(&mut owner, &[key(ViewportKey::Cancel, true)]);
    assert!(!owner.placement_active());
    assert_eq!(line_tool_geometry(&owner, document, sketch), accepted);
    assert_eq!(
        owner.gui_command_state().unwrap().undo_label.as_deref(),
        Some("Create Line")
    );
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert_eq!(
        line_tool_geometry(&owner, document, sketch).lines().len(),
        1
    );
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(line_tool_geometry(&owner, document, sketch), accepted);
}
#[test]
fn escape_keeps_segments_but_abandons_an_incomplete_line() {
    for points in [vec![], vec![[0, 0]], vec![[0, 0], [50, 20], [80, 90]]] {
        let (mut owner, document, sketch) = line_owner();
        for point in &points {
            deliver(&mut owner, &[button(*point, true)]);
        }
        deliver(&mut owner, &[key(ViewportKey::Cancel, true)]);
        assert!(!owner.placement_active());
        assert!(
            !owner
                .files
                .workspace()
                .session(document)
                .unwrap()
                .in_transaction()
        );
        assert_eq!(
            line_tool_geometry(&owner, document, sketch).lines().len(),
            points.len().saturating_sub(1)
        );
    }
}
#[test]
fn close_joins_the_first_vertex_and_asks_for_a_new_start() {
    let (mut owner, document, sketch) = line_owner();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            button([50, 20], true),
            button([50, 20], false),
            button([100, 80], true),
            button([100, 80], false),
        ],
    );
    owner
        .line_command_from_gui(oblikovati_gui::GuiLineCommand::Close)
        .unwrap();
    let geometry = line_tool_geometry(&owner, document, sketch);
    assert_eq!(geometry.lines().len(), 3);
    assert_eq!(geometry.points().len(), 3);
    assert_eq!(
        geometry.lines()[2].endpoints[1],
        geometry.lines()[0].endpoints[0]
    );
    assert!(owner.placement_active());
    assert!(
        !owner.gui_line_command_menu().unwrap().restart_enabled,
        "Close returns the command to start-point selection"
    );
}
#[test]
fn inferred_line_preview_matches_the_committed_graph() {
    let (mut owner, document, sketch) = line_owner();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            button([100, 1], true),
            button([100, 1], false),
            motion([100, 99]),
        ],
    );
    let oblikovati_viewport::ViewportSketchPreview::Resolved {
        geometry: preview, ..
    } = owner.sketch_preview().unwrap().clone()
    else {
        panic!("expected inferred Line preview")
    };
    assert!(!preview.constraints().is_empty());
    deliver(&mut owner, &[button([100, 99], true)]);
    assert_eq!(line_tool_geometry(&owner, document, sketch), *preview);
}
