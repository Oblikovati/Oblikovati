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
fn continuous_line_keeps_preview_private_and_commits_shared_endpoints_once() {
    let (mut owner, document, sketch) = line_owner();
    let marks = owner
        .files
        .workspace()
        .session(document)
        .unwrap()
        .history()
        .sketch_reservations();
    assert!(owner.gui_command_state().unwrap().transaction_open);
    assert!(!owner.close_allowed().unwrap());
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([50, 20], true),
            button([80, 90], true),
            motion([120, 150]),
        ],
    );
    assert!(owner.placement_active());
    assert!(
        line_tool_geometry(&owner, document, sketch)
            .lines()
            .is_empty()
    );
    assert_eq!(
        owner
            .files
            .workspace()
            .session(document)
            .unwrap()
            .history()
            .sketch_reservations(),
        marks
    );
    let oblikovati_viewport::ViewportSketchPreview::Resolved {
        geometry: preview,
        members,
        ..
    } = owner.sketch_preview().unwrap()
    else {
        panic!("expected inferred Line preview")
    };
    assert_eq!(members.len(), 7);
    assert_eq!(preview.lines().len(), 3);
    assert_eq!(preview.points().len(), 4);
    assert_eq!(preview.points()[3].position, [120., 150.]);
    deliver(&mut owner, &[line_key(ViewportPhysicalKey::Enter)]);
    assert!(!owner.placement_active());
    assert!(!owner.gui_command_state().unwrap().transaction_open);
    let geometry = line_tool_geometry(&owner, document, sketch);
    assert_eq!(geometry.lines().len(), 2);
    assert_eq!(geometry.points().len(), 3);
    assert_eq!(
        geometry.lines()[0].endpoints[1],
        geometry.lines()[1].endpoints[0]
    );
    assert_eq!(
        owner.gui_command_state().unwrap().undo_label.as_deref(),
        Some("Line")
    );
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
    assert!(
        line_tool_geometry(&owner, document, sketch)
            .lines()
            .is_empty()
    );
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert_eq!(line_tool_geometry(&owner, document, sketch), geometry);
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
fn backspace_changes_only_the_draft_and_close_uses_the_first_shared_vertex() {
    let (mut owner, document, sketch) = line_owner();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([50, 20], true),
            button([80, 90], true),
            line_key(ViewportPhysicalKey::Backspace),
            button([100, 80], true),
        ],
    );
    deliver(
        &mut owner,
        &[ViewportEvent::Input(ViewportInput::Text('c'))],
    );
    let geometry = line_tool_geometry(&owner, document, sketch);
    assert_eq!(geometry.lines().len(), 3);
    assert_eq!(geometry.points().len(), 3);
    assert_eq!(
        geometry.lines()[2].endpoints[1],
        geometry.lines()[0].endpoints[0]
    );
    assert_eq!(geometry.points()[2].position, [100., 80.]);
}

#[test]
fn inferred_line_preview_matches_the_committed_graph() {
    let (mut owner, document, sketch) = line_owner();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([100, 1], true),
            button([100, 99], true),
        ],
    );
    let oblikovati_viewport::ViewportSketchPreview::Resolved {
        geometry: preview, ..
    } = owner.sketch_preview().unwrap()
    else {
        panic!("expected inferred Line preview")
    };
    assert!(!preview.constraints().is_empty());
    deliver(&mut owner, &[line_key(ViewportPhysicalKey::Enter)]);
    assert_eq!(line_tool_geometry(&owner, document, sketch), *preview);
}
