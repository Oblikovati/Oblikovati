// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use crate::{GuiParameterLinkParameter, GuiParameterLinkSource};
#[test]
fn linked_health_is_explicit_and_keeps_source_failure_reasons() {
    assert_eq!(linked_health_label(""), "OK");
    assert_eq!(
        linked_health_label("source unavailable"),
        "source unavailable"
    );
}
#[test]
fn link_picker_emits_one_owned_selection_and_retires_after_matching_result() {
    let snapshot = GuiParameterSnapshot {
        document: 7,
        rows: Vec::new(),
        linked_tables: Vec::new(),
        groups: Vec::new(),
        link_sources: vec![GuiParameterLinkSource {
            document: 8,
            source_document: "source.opd".into(),
            display_name: "Source".into(),
            parameters: vec![GuiParameterLinkParameter {
                parameter: 3,
                name: "width".into(),
            }],
        }],
    };
    let mut picker = LinkPicker::default();
    picker.open();
    picker.source = Some(8);
    picker.checked.insert(3);
    let mut actions = Vec::new();
    picker.enqueue(&snapshot, &mut actions);
    let [GuiParameterAction::Link(edit)] = actions.as_slice() else {
        panic!("expected one link action, received {actions:?}");
    };
    assert_eq!(edit.document, 7);
    assert_eq!(edit.source_document, 8);
    assert_eq!(edit.linked, vec![3]);
    picker.acknowledge(edit, Ok(()));
    assert!(!picker.open);
}
#[test]
fn canonical_linked_tables_build_owned_draws_in_the_parameter_panel() {
    let mut engine = crate::GuiFrameEngine::new().unwrap();
    let snapshot = GuiParameterSnapshot {
        document: 7,
        rows: Vec::new(),
        linked_tables: vec![DerivedParameterTableInfo {
            id: 1,
            source_document: "source.opd".into(),
            linked: Some(vec!["width".into()]),
            ..Default::default()
        }],
        link_sources: Vec::new(),
        groups: Vec::new(),
    };
    engine.set_parameter_snapshot(Some(snapshot)).unwrap();
    engine.set_parameters_open(true);
    let request = crate::GuiFrameRequest {
        extent: [1000, 700],
        events: Vec::new(),
        commands: Default::default(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    };
    engine.frame(&request).unwrap();
    let frame = engine.frame(&request).unwrap();
    assert!(!frame.draws.batches.is_empty());
    assert!(frame.parameter_actions.is_empty());
}
struct LinkedTableWidgetFixture {
    context: imgui::Context,
    delete_position: [f32; 2],
}
impl LinkedTableWidgetFixture {
    fn new() -> Self {
        let mut context = imgui::Context::create();
        context.set_ini_filename(None);
        context.io_mut().display_size = [900., 300.];
        context.io_mut().delta_time = 1. / 60.;
        context.fonts().build_rgba32_texture();
        Self {
            context,
            delete_position: [0.; 2],
        }
    }
    fn frame(&mut self, events: &[crate::GuiInputEvent]) -> Vec<GuiParameterAction> {
        for event in events {
            crate::input::apply_input(self.context.io_mut(), event, [1.; 2]);
        }
        let ui = self.context.frame();
        let mut actions = Vec::new();
        ui.window("Linked-table button probe")
            .position([8., 8.], Condition::Always)
            .size([800., 200.], Condition::Always)
            .build(|| {
                let Some(_table) = ui.begin_table("##linked-table-probe", 4) else {
                    return;
                };
                for label in ["Source", "Linked", "Health", ""] {
                    ui.table_setup_column(label);
                }
                ui.table_headers_row();
                let table = DerivedParameterTableInfo {
                    id: 11,
                    source_document: "source.opd".into(),
                    ..Default::default()
                };
                draw_linked_table_row(ui, 7, &table, &mut actions);
                let minimum = ui.item_rect_min();
                let maximum = ui.item_rect_max();
                self.delete_position =
                    std::array::from_fn(|axis| (minimum[axis] + maximum[axis]) / 2.);
            });
        self.context.render();
        actions
    }
}
#[test]
fn real_linked_table_delete_button_emits_the_scoped_document_and_table_identity() {
    use crate::{GuiInputEvent, GuiPointerButton};
    let mut fixture = LinkedTableWidgetFixture::new();
    fixture.frame(&[]);
    fixture.frame(&[]);
    fixture.frame(&[GuiInputEvent::PointerMoved(fixture.delete_position)]);
    assert!(
        fixture
            .frame(&[GuiInputEvent::PointerButton {
                button: GuiPointerButton::Left,
                pressed: true
            }])
            .is_empty()
    );
    assert_eq!(
        fixture.frame(&[GuiInputEvent::PointerButton {
            button: GuiPointerButton::Left,
            pressed: false
        }]),
        vec![GuiParameterAction::DeleteLinkedTable {
            document: 7,
            table: 11
        }]
    );
    assert!(fixture.frame(&[]).is_empty());
}
