// SPDX-License-Identifier: GPL-2.0-only
use super::*;
#[test]
fn a_drawing_cannot_be_created_until_its_template_preview_is_accepted() {
    let state = GuiCommandState::default();
    let mut chooser = TemplateChooser {
        selected: Some(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        }),
        ..Default::default()
    };
    assert!(!chooser.can_create(&state));
    chooser.use_template = true;
    assert!(chooser.can_create(&state));
}
#[test]
fn an_open_transaction_or_no_selection_blocks_creation() {
    let mut chooser = TemplateChooser::default();
    assert!(!chooser.can_create(&GuiCommandState::default()));
    chooser.selected = Some(GuiDocumentTemplate::default());
    assert!(chooser.can_create(&GuiCommandState::default()));
    let busy = GuiCommandState {
        transaction_open: true,
        ..Default::default()
    };
    assert!(!chooser.can_create(&busy));
}
#[test]
fn every_view_has_its_label_and_only_icon_sizes_change_the_row_height() {
    let labels: Vec<&str> = TemplateView::ALL.map(TemplateView::label).into();
    assert_eq!(
        labels,
        [
            "Extra Large Icons",
            "Medium Icons",
            "Small Icons",
            "List",
            "Details"
        ]
    );
    assert!(TemplateView::ExtraLargeIcons.row_height() > TemplateView::MediumIcons.row_height());
    assert!(TemplateView::MediumIcons.row_height() > TemplateView::SmallIcons.row_height());
    assert_eq!(TemplateView::List.row_height(), 0.);
    assert_eq!(TemplateView::Details.row_height(), 0.);
}
#[test]
fn back_and_forward_walk_the_folders_visited() {
    let metric = TemplateFolder::Units(GuiTemplateUnits::Metric);
    let inch = TemplateFolder::Units(GuiTemplateUnits::Inch);
    let mut chooser = TemplateChooser::default();
    chooser.go_to(inch);
    chooser.go_to(metric);
    chooser.back();
    assert_eq!(chooser.folder, inch);
    chooser.back();
    assert_eq!(chooser.folder, metric);
    chooser.back();
    assert_eq!(chooser.folder, metric);
    chooser.forward();
    assert_eq!(chooser.folder, inch);
}
#[test]
fn a_new_folder_ends_the_path_forward() {
    let inch = TemplateFolder::Units(GuiTemplateUnits::Inch);
    let mut chooser = TemplateChooser::default();
    chooser.go_to(inch);
    chooser.back();
    chooser.go_to(TemplateFolder::Root);
    chooser.forward();
    assert_eq!(chooser.folder, TemplateFolder::Root);
}
#[test]
fn choosing_the_current_folder_adds_no_history() {
    let mut chooser = TemplateChooser::default();
    chooser.go_to(TemplateFolder::default());
    assert!(chooser.folders_behind.is_empty());
}
#[test]
fn opening_the_chooser_does_not_create_a_document_and_escape_cancels() {
    let mut engine = crate::GuiFrameEngine::new().unwrap();
    let mut request = crate::GuiFrameRequest {
        extent: [1000, 700],
        events: Vec::new(),
        commands: GuiCommandState::default(),
        status: None,
        delta_seconds: 1. / 60.,
        framebuffer_scale: [1.; 2],
    };
    engine.frame(&request).unwrap();
    request.events = vec![crate::GuiInputEvent::Modifiers(crate::GuiModifiers {
        control: true,
        ..Default::default()
    })];
    engine.frame(&request).unwrap();
    request.events = vec![crate::GuiInputEvent::Key {
        key: crate::GuiKey::N,
        pressed: true,
    }];
    assert!(engine.frame(&request).unwrap().actions.is_empty());
    request.events = vec![
        crate::GuiInputEvent::Key {
            key: crate::GuiKey::N,
            pressed: false,
        },
        crate::GuiInputEvent::Modifiers(Default::default()),
    ];
    engine.frame(&request).unwrap();
    request.events = vec![crate::GuiInputEvent::Key {
        key: crate::GuiKey::Escape,
        pressed: true,
    }];
    assert!(engine.frame(&request).unwrap().actions.is_empty());
}
