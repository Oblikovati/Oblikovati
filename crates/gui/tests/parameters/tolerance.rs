// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn tolerance_row() -> GuiParameterRow {
    GuiParameterRow {
        tolerance_text: String::new(),
        has_groups: false,
        parameter: 0,
        detail: oblikovati_api::wire::ParameterDetail {
            units: "mm".into(),
            model_value_type: "upper".into(),
            tolerance: Some(Box::new(oblikovati_api::wire::ToleranceInfo {
                upper: 0.2,
                lower: -0.1,
                ..Default::default()
            })),
            ..Default::default()
        },
    }
}
#[test]
fn tolerance_editor_seeds_working_deviations_and_each_canonical_model_value() {
    let mut editor = ParameterToleranceEditor::default();
    editor.open(7, &tolerance_row()).unwrap();
    assert_eq!((editor.upper, editor.lower), (0.2, -0.1));
    assert_eq!(editor.selection, Some(MODEL_VALUE_UPPER));
    for (selection, name, label) in MODEL_SELECTIONS {
        assert_eq!(selection_from_name(name).unwrap(), selection);
        assert_eq!(selection_label(selection).unwrap(), label);
        editor.selection = Some(selection);
        let edit = editor.submit().unwrap().unwrap();
        edit.validate().unwrap();
        assert_eq!(edit.selection, selection);
        assert_eq!(edit.tolerance, deviation_band(0.2, -0.1));
        assert!(editor.submit().unwrap().is_none());
        assert_eq!(editor.acknowledge(&edit, Ok(())), GuiParameterAck::Applied);
        editor.open(7, &tolerance_row()).unwrap();
    }
    assert_eq!(selection_from_name("").unwrap(), MODEL_VALUE_NOMINAL);
}
#[test]
fn failed_tolerance_edits_keep_fields_and_signed_zero_results_must_match_exactly() {
    let mut editor = ParameterToleranceEditor::default();
    editor.open(7, &tolerance_row()).unwrap();
    editor.upper = 0.;
    editor.lower = -0.;
    let edit = editor.submit().unwrap().unwrap();
    let mut forged = edit.clone();
    forged.tolerance.lower = 0.;
    assert_ne!(edit, forged);
    assert_eq!(
        editor.acknowledge(&forged, Ok(())),
        GuiParameterAck::Obsolete
    );
    assert!(editor.request.pending.is_some());
    assert_eq!(
        editor.acknowledge(
            &edit,
            Err("tolerance: expected an editable numeric parameter".into())
        ),
        GuiParameterAck::Failed
    );
    assert!(editor.request.issue.is_some());
    assert_eq!(editor.lower.to_bits(), (-0_f64).to_bits());
    editor.open(8, &tolerance_row()).unwrap();
    let current = editor.submit().unwrap().unwrap();
    assert!(current.request > edit.request);
    assert_eq!(editor.acknowledge(&edit, Ok(())), GuiParameterAck::Obsolete);
    assert!(editor.target.is_some());
}
#[test]
fn invalid_tolerances_refuse_nonfinite_reversed_and_unsupported_values_before_submission() {
    let mut editor = ParameterToleranceEditor::default();
    assert!(editor.submit().is_err());
    editor.open(7, &tolerance_row()).unwrap();
    for (upper, lower) in [(f64::NAN, 0.), (1., f64::INFINITY), (-1., 1.)] {
        editor.upper = upper;
        editor.lower = lower;
        assert!(editor.submit().is_err());
        assert!(editor.request.pending.is_none());
    }
    editor.upper = 0.2;
    editor.lower = -0.1;
    editor.selection = Some(ModelValueType(-1));
    assert!(editor.submit().is_err());
    editor.selection = None;
    assert!(editor.submit().is_err());
    editor.selection = Some(MODEL_VALUE_NOMINAL);
    editor.request.last_request = u64::MAX;
    assert!(editor.submit().is_err());
}
#[test]
fn tolerance_open_validation_is_atomic_and_document_changes_retire_stale_targets() {
    let mut editor = ParameterToleranceEditor::default();
    let snapshot = GuiParameterSnapshot {
        linked_tables: Vec::new(),
        link_sources: Vec::new(),
        groups: Vec::new(),
        document: 7,
        rows: vec![tolerance_row()],
    };
    editor.open(7, &snapshot.rows[0]).unwrap();
    for units in ["text", "boolean"] {
        let mut invalid = tolerance_row();
        invalid.detail.units = units.into();
        assert!(editor.open(8, &invalid).is_err());
        assert_eq!(editor.target.unwrap().document, 7);
    }
    let mut invalid = tolerance_row();
    invalid.detail.model_value_type = "unknown".into();
    assert!(editor.open(8, &invalid).is_err());
    assert!(selection_label(ModelValueType(-1)).is_err());
    editor.synchronize(Some(&snapshot));
    assert!(editor.target.is_some());
    editor.synchronize(Some(&GuiParameterSnapshot {
        linked_tables: Vec::new(),
        link_sources: Vec::new(),
        groups: Vec::new(),
        document: 7,
        rows: Vec::new(),
    }));
    assert!(editor.target.is_none());
    editor.open(7, &tolerance_row()).unwrap();
    editor.synchronize(None);
    assert!(editor.target.is_none());
}
#[test]
fn tolerance_packets_validate_identity_deviation_shape_and_fit_annotations() {
    let mut editor = ParameterToleranceEditor::default();
    editor.open(7, &tolerance_row()).unwrap();
    let edit = editor.submit().unwrap().unwrap();
    for (document, field, request) in [
        (0, GuiParameterTextField::Equation, 1),
        (7, GuiParameterTextField::Name, 1),
        (7, GuiParameterTextField::Equation, 0),
    ] {
        let mut invalid = edit.clone();
        invalid.target.document = document;
        invalid.target.field = field;
        invalid.request = request;
        assert!(invalid.validate().is_err());
    }
    let mut invalid = edit.clone();
    invalid.tolerance.type_ = oblikovati_api::types::ToleranceType(-1);
    assert!(invalid.validate().is_err());
    let mut invalid = edit;
    invalid.tolerance.hole_tolerance = "H7".into();
    assert!(invalid.validate().is_err());
}
#[test]
fn tolerance_enqueue_keeps_declines_local_and_publishes_one_owned_edit() {
    let mut editor = ParameterToleranceEditor::default();
    let mut actions = Vec::new();
    editor.enqueue(&mut actions);
    assert!(editor.request.issue.is_some());
    assert!(actions.is_empty());
    editor.open(7, &tolerance_row()).unwrap();
    editor.enqueue(&mut actions);
    editor.enqueue(&mut actions);
    assert!(matches!(
        actions.as_slice(),
        [GuiParameterAction::Tolerance(_)]
    ));
}
struct ToleranceScalarWidgetFixture {
    context: imgui::Context,
    editor: ParameterToleranceEditor,
    ok_position: [f32; 2],
}
impl ToleranceScalarWidgetFixture {
    fn new() -> Self {
        let mut context = imgui::Context::create();
        context.set_ini_filename(None);
        context.io_mut().display_size = [500., 350.];
        context.io_mut().delta_time = 1. / 60.;
        context.fonts().build_rgba32_texture();
        let mut editor = ParameterToleranceEditor::default();
        editor.open(7, &tolerance_row()).unwrap();
        Self {
            context,
            editor,
            ok_position: [0.; 2],
        }
    }
    fn frame(
        &mut self,
        focus: Option<usize>,
        events: &[crate::GuiInputEvent],
    ) -> Vec<GuiParameterAction> {
        for event in events {
            crate::input::apply_input(self.context.io_mut(), event, [1.; 2]);
        }
        let spacing = self.context.style().item_spacing[0];
        let padding = self.context.style().frame_padding[0];
        let ui = self.context.frame();
        let mut actions = Vec::new();
        ui.window("Tolerance widget probe")
            .position([8., 8.], Condition::Always)
            .size([420., 280.], Condition::Always)
            .build(|| {
                if let Some(offset) = focus {
                    ui.set_keyboard_focus_here_with_offset(imgui::FocusedWidget::Offset(
                        offset as u32,
                    ));
                }
                self.editor.draw_body(ui, &mut actions);
                let minimum = ui.item_rect_min();
                let maximum = ui.item_rect_max();
                self.ok_position = [
                    minimum[0] - spacing - (ui.calc_text_size("OK")[0] + 2. * padding) / 2.,
                    (minimum[1] + maximum[1]) / 2.,
                ];
            });
        self.context.render();
        actions
    }
    fn type_scalar(&mut self, offset: usize, source: &str) {
        self.frame(Some(offset), &[]);
        self.frame(None, &[]);
        self.frame(None, &[crate::GuiInputEvent::Text(source.into())]);
    }
    fn click_ok(&mut self) -> Vec<GuiParameterAction> {
        use crate::{GuiInputEvent, GuiPointerButton};
        self.frame(None, &[GuiInputEvent::PointerMoved(self.ok_position)]);
        self.frame(
            None,
            &[GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed: true,
            }],
        );
        self.frame(
            None,
            &[GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed: false,
            }],
        )
    }
}
#[test]
fn real_double_precision_tolerance_inputs_preserve_typed_bits_and_emit_one_owned_edit() {
    let mut fixture = ToleranceScalarWidgetFixture::new();
    fixture.frame(None, &[]);
    fixture.type_scalar(0, "0.12345678901234567");
    fixture.type_scalar(1, "-0");
    assert_eq!(
        fixture.editor.upper.to_bits(),
        0.12345678901234567_f64.to_bits()
    );
    assert_eq!(fixture.editor.lower.to_bits(), (-0_f64).to_bits());
    let actions = fixture.click_ok();
    let [GuiParameterAction::Tolerance(edit)] = actions.as_slice() else {
        panic!("expected one tolerance edit: {actions:?}");
    };
    assert_eq!(
        edit.tolerance.upper.to_bits(),
        fixture.editor.upper.to_bits()
    );
    assert_eq!(edit.tolerance.lower.to_bits(), (-0_f64).to_bits());
    assert!(fixture.click_ok().is_empty());
}
#[test]
fn switching_tolerance_documents_does_not_reuse_the_old_active_scalar_input() {
    let mut fixture = ToleranceScalarWidgetFixture::new();
    fixture.frame(None, &[]);
    fixture.type_scalar(0, "1.25");
    assert_eq!(fixture.editor.upper, 1.25);
    fixture.editor.open(8, &tolerance_row()).unwrap();
    assert_eq!(fixture.editor.upper, 0.2);
    fixture.frame(None, &[crate::GuiInputEvent::Text("9".into())]);
    assert_eq!(fixture.editor.upper, 0.2);
    assert_eq!(fixture.editor.lower, -0.1);
}
#[test]
fn switching_tolerance_parameters_does_not_reuse_the_old_active_scalar_input() {
    let mut fixture = ToleranceScalarWidgetFixture::new();
    fixture.frame(None, &[]);
    fixture.type_scalar(0, "1.25");
    let mut replacement = tolerance_row();
    replacement.parameter = 11;
    fixture.editor.open(7, &replacement).unwrap();
    fixture.frame(None, &[crate::GuiInputEvent::Text("9".into())]);
    assert_eq!(fixture.editor.upper, 0.2);
    assert_eq!(fixture.editor.lower, -0.1);
}
