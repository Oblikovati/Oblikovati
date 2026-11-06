// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use super::{
    geometric_tools::{geometric_owner, geometry_for_relation},
    line_tools::line_key,
    slot_input::deliver,
};
use crate::placement_test_inputs::*;
use SketchCurveFormat::{Centerline, Construction};
use oblikovati_gui::{GuiSketchConstraint, GuiSketchTool};
use oblikovati_model::{
    document_sketch::DocumentSketchId,
    sketch::{Sketch2D, SketchCurveFormat},
};
use oblikovati_viewport::ViewportPhysicalKey;
fn format_session(
    owner: &TestOwner,
    document: u64,
) -> &oblikovati_session::DocumentSession<DesktopObservations> {
    owner.files.workspace().session(document).unwrap()
}

fn assert_format_selection(owner: &mut TestOwner, expected: i64) {
    assert_eq!(
        Client::new(owner.document_host())
            .model()
            .selection()
            .unwrap()
            .count,
        expected
    );
}

#[test]
fn arming_creation_formats_has_no_history_or_generation_change() {
    let (mut owner, document, _) = geometric_owner(&[], false);
    let generation = format_session(&owner, document).generation();
    let before = format_session(&owner, document).history_entries();
    owner.toggle_sketch_format_from_gui(Construction).unwrap();
    owner.toggle_sketch_format_from_gui(Centerline).unwrap();
    assert_eq!(format_session(&owner, document).generation(), generation);
    assert_eq!(format_session(&owner, document).history_entries(), before);
    let state = owner.gui_command_state().unwrap();
    assert!(state.sketch_formats.construction && state.sketch_formats.centerline);
}
#[test]
fn line_preview_and_commit_share_creation_roles_and_mid_tool_changes() {
    for format in [Construction, Centerline] {
        verify_owned_line_format(format);
    }
}
fn verify_owned_line_format(format: SketchCurveFormat) {
    let (mut owner, document, sketch) = geometric_owner(&[], false);
    owner.toggle_sketch_format_from_gui(format).unwrap();
    owner
        .place_sketch_tool_from_gui(GuiSketchTool::Line)
        .unwrap();
    verify_line_preview_format_cycle(&mut owner, format);
    deliver(
        &mut owner,
        &[
            button([100, 20], true),
            line_key(ViewportPhysicalKey::Enter),
        ],
    );
    let geometry = geometry_for_relation(&owner, document, sketch).clone();
    let line = geometry.lines()[0].id;
    assert!(geometry.is_construction(line).unwrap());
    assert_eq!(geometry.is_centerline(line).unwrap(), format == Centerline);
    assert_line_format_history(&mut owner, document, sketch, &geometry);
}

fn verify_line_preview_format_cycle(owner: &mut TestOwner, format: SketchCurveFormat) {
    deliver(owner, &[button([0, 0], true), motion([100, 20])]);
    assert_preview_format(owner, format, true);
    owner.toggle_sketch_format_from_gui(format).unwrap();
    assert_preview_format(owner, format, false);
    owner.toggle_sketch_format_from_gui(format).unwrap();
}

fn assert_preview_format(owner: &TestOwner, format: SketchCurveFormat, armed: bool) {
    let preview = owner.placement_preview().unwrap();
    for member in &preview.members {
        assert_eq!(member.construction, armed && format == Construction);
        assert_eq!(member.centerline, armed && format == Centerline);
    }
}
fn undo_format_edit(owner: &mut TestOwner) {
    Client::new(owner.document_host())
        .transactions()
        .undo()
        .unwrap();
}

fn assert_line_format_history(
    owner: &mut TestOwner,
    document: u64,
    sketch: DocumentSketchId,
    expected: &Sketch2D,
) {
    undo_format_edit(owner);
    assert!(
        geometry_for_relation(owner, document, sketch)
            .lines()
            .is_empty()
    );
    Client::new(owner.document_host())
        .transactions()
        .redo()
        .unwrap();
    assert!(geometry_for_relation(owner, document, sketch).same_recipe(expected));
}
#[test]
fn construction_circle_preview_and_commit_share_roles() {
    let (mut owner, document, sketch) = geometric_owner(&[], false);
    owner.toggle_sketch_format_from_gui(Construction).unwrap();
    owner
        .place_sketch_tool_from_gui(GuiSketchTool::Circle)
        .unwrap();
    deliver(
        &mut owner,
        &[
            button([0, 0], true),
            button([0, 0], false),
            motion([100, 0]),
        ],
    );
    assert!(owner.placement_preview().unwrap().members[0].construction);
    deliver(&mut owner, &[button([100, 0], true)]);
    let geometry = geometry_for_relation(&owner, document, sketch).clone();
    assert!(geometry.is_construction(geometry.circles()[0].id).unwrap());
}
#[test]
fn selected_conversion_keeps_picks_and_does_not_arm_creation_mode() {
    let (mut owner, document, sketch) = geometric_owner(&[[0., 0.], [100., 20.]], true);
    deliver(&mut owner, &[button([50, 10], true)]);
    let before = geometry_for_relation(&owner, document, sketch).clone();
    owner.toggle_sketch_format_from_gui(Centerline).unwrap();
    let converted = geometry_for_relation(&owner, document, sketch).clone();
    assert_eq!(before.points(), converted.points());
    assert!(converted.is_centerline(converted.lines()[0].id).unwrap());
    assert_format_selection(&mut owner, 1);
    assert!(!owner.gui_command_state().unwrap().sketch_formats.centerline);
    let generation = format_session(&owner, document).generation();
    owner.toggle_sketch_format_from_gui(Construction).unwrap();
    assert_eq!(format_session(&owner, document).generation(), generation);
    assert!(converted.same_recipe(geometry_for_relation(&owner, document, sketch)));
}
#[test]
fn unsupported_construction_point_is_refused_before_retiring_the_current_tool() {
    let (mut owner, _, _) = geometric_owner(&[], false);
    owner.toggle_sketch_format_from_gui(Construction).unwrap();
    owner
        .place_sketch_tool_from_gui(GuiSketchTool::Circle)
        .unwrap();
    assert!(
        owner
            .place_sketch_tool_from_gui(GuiSketchTool::Point)
            .unwrap_err()
            .contains("construction points are unsupported")
    );
    assert_eq!(
        owner.placement_state().unwrap().tool,
        crate::SketchPlacementTool::Circle
    );
}
#[test]
fn construction_mode_cannot_be_armed_for_an_active_point_tool() {
    let (mut owner, _, _) = geometric_owner(&[], false);
    owner
        .place_sketch_tool_from_gui(GuiSketchTool::Point)
        .unwrap();
    assert!(
        owner
            .toggle_sketch_format_from_gui(Construction)
            .unwrap_err()
            .contains("active Point")
    );
    assert!(
        !owner
            .gui_command_state()
            .unwrap()
            .sketch_formats
            .construction
    );
}
#[test]
fn format_commands_refuse_caller_transactions_and_non_placement_tools() {
    let (mut owner, _, _) = geometric_owner(&[], false);
    Client::new(owner.document_host())
        .transactions()
        .begin(&TransactionBeginArgs {
            label: "Caller".into(),
        })
        .unwrap();
    assert!(owner.toggle_sketch_format_from_gui(Construction).is_err());
    Client::new(owner.document_host())
        .transactions()
        .abort()
        .unwrap();
}
#[test]
fn format_commands_require_a_placement_or_idle_sketch_editor() {
    let (mut owner, _, _) = geometric_owner(&[], false);
    owner
        .constrain_sketch_from_gui(GuiSketchConstraint::Coincident)
        .unwrap();
    assert!(!owner.gui_command_state().unwrap().sketch_format_available);
    assert!(owner.toggle_sketch_format_from_gui(Centerline).is_err());
    owner.finish_sketch_edit().unwrap();
    assert!(
        owner
            .toggle_sketch_format_from_gui(Construction)
            .unwrap_err()
            .contains("active sketch edit")
    );
}

#[test]
fn format_arming_after_undo_uses_the_live_empty_selection_without_rebinding_stale_picks() {
    let (mut owner, document, sketch) = geometric_owner(&[[0., 0.], [100., 20.]], true);
    deliver(&mut owner, &[button([50, 10], true)]);
    owner.toggle_sketch_format_from_gui(Centerline).unwrap();
    undo_format_edit(&mut owner);
    assert_format_selection(&mut owner, 0);
    let before = geometry_for_relation(&owner, document, sketch).clone();
    owner.toggle_sketch_format_from_gui(Construction).unwrap();
    assert!(
        owner
            .gui_command_state()
            .unwrap()
            .sketch_formats
            .construction
    );
    assert!(before.same_recipe(geometry_for_relation(&owner, document, sketch)));
    assert_format_selection(&mut owner, 0);
}
