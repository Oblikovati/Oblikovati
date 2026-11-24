// SPDX-License-Identifier: GPL-2.0-only
use super::*;

#[test]
fn restored_selection_matches_the_file_in_the_active_catalog() {
    let saved = crate::GuiProjectTemplate {
        path: "/project/English/Standard (in).opd".into(),
        title: "Standard (in)".into(),
        folder: TemplateFolder::Units(GuiTemplateUnits::Inch),
        kind: GuiTemplateKind::Part,
        length_unit: "in",
        size: 42,
        material: None,
    };
    let unrelated = crate::GuiProjectTemplate {
        path: "/other/English/Standard (in).opd".into(),
        ..saved.clone()
    };
    let mut chooser = TemplateChooser::default();
    chooser.remember_project_template(Some(saved.path.clone()));
    chooser.restore_project_selection(&GuiCommandState {
        project_templates: vec![unrelated].into(),
        ..Default::default()
    });
    assert!(chooser.selected.is_none());
    assert_eq!(chooser.project_template_path(), Some(saved.path.as_path()));

    chooser.restore_project_selection(&GuiCommandState {
        project_templates: vec![saved.clone()].into(),
        ..Default::default()
    });
    assert_eq!(chooser.folder, saved.folder);
    assert_eq!(chooser.selected, Some(TemplateChoice::Project(saved)));
}

#[test]
fn a_root_only_catalog_opens_at_its_named_templates() {
    let mut chooser = TemplateChooser::default();
    let root_template = crate::GuiProjectTemplate {
        path: "templates/Standard (mm).oad".into(),
        title: "Standard (mm)".into(),
        folder: TemplateFolder::Root,
        kind: GuiTemplateKind::Assembly,
        length_unit: "mm",
        size: 42,
        material: None,
    };
    let state = GuiCommandState {
        project_templates: vec![root_template.clone()].into(),
        ..Default::default()
    };

    chooser.reveal_project_root(&state);
    assert_eq!(chooser.folder, TemplateFolder::Root);
    assert_eq!(
        chooser.catalog(&state, &[GuiTemplateKind::Assembly]),
        vec![TemplateChoice::Project(root_template)]
    );
    assert!(
        chooser
            .catalog(&GuiCommandState::default(), &[GuiTemplateKind::Assembly])
            .is_empty()
    );
}
#[test]
fn details_columns_reserve_measured_text_and_group_indent() {
    let layout = DetailsLayout::new([200., 60., 90.], 20., 8.);
    assert_eq!(layout.size, 228.);
    assert_eq!(layout.type_column, 296.);
    assert_eq!(layout.width, 394.);
    let scaled = DetailsLayout::new([400., 120., 180.], 40., 16.);
    assert_eq!(scaled.width, 2. * layout.width);
}
#[test]
fn gallery_keeps_space_for_details_in_a_narrow_dialog() {
    assert_eq!(gallery_column_width(820.), GALLERY_COLUMN_WIDTH);
    let narrow = gallery_column_width(460.);
    assert!(narrow > 0. && narrow < GALLERY_COLUMN_WIDTH);
    assert!(460. - FOLDER_COLUMN_WIDTH - narrow > 100.);
}
#[test]
fn a_drawing_cannot_be_created_until_its_template_preview_is_accepted() {
    let state = GuiCommandState::default();
    let mut chooser = TemplateChooser {
        selected: Some(
            GuiDocumentTemplate {
                kind: GuiTemplateKind::Drawing,
                units: GuiTemplateUnits::Metric,
            }
            .into(),
        ),
        ..Default::default()
    };
    assert!(!chooser.can_create(&state));
    chooser.drawing_layout = Some(DrawingLayout::Template);
    assert!(chooser.can_create(&state));
    chooser.drawing_layout = Some(DrawingLayout::Assembly);
    assert!(chooser.can_create(&state));
}
#[test]
fn an_open_transaction_or_no_selection_blocks_creation() {
    let mut chooser = TemplateChooser::default();
    assert!(!chooser.can_create(&GuiCommandState::default()));
    chooser.selected = Some(GuiDocumentTemplate::default().into());
    assert!(chooser.can_create(&GuiCommandState::default()));
    let busy = GuiCommandState {
        transaction_open: true,
        ..Default::default()
    };
    assert!(!chooser.can_create(&busy));
}
#[test]
fn every_view_has_its_label_and_icon_sizes_change_the_tile_height() {
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
    assert!(
        TemplateView::ExtraLargeIcons.tile_size(1.)[1] > TemplateView::MediumIcons.tile_size(1.)[1]
    );
    assert!(TemplateView::MediumIcons.tile_size(1.)[1] > TemplateView::SmallIcons.tile_size(1.)[1]);
    assert!(!TemplateView::List.is_icon_view());
    assert!(!TemplateView::Details.is_icon_view());
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
fn template_sort_switches_columns_and_preserves_selection_order_for_ties() {
    use super::template_sort::{SortOrder, TemplateSort, ordered_kinds};
    use GuiTemplateKind::{Part, SheetMetal};
    let state = GuiCommandState {
        template_sizes: vec![(Part, 10), (SheetMetal, 20)],
        ..Default::default()
    };
    let units = GuiTemplateUnits::Metric;
    let kinds = [Part, SheetMetal];
    let sort = TemplateSort::default();
    assert_eq!(
        ordered_kinds(&kinds, sort, &state, units),
        [SheetMetal, Part]
    );
    let sort = sort.toggle_name();
    assert_eq!(
        ordered_kinds(&kinds, sort, &state, units),
        [Part, SheetMetal]
    );
    assert_eq!(
        sort.toggle_name(),
        TemplateSort::Name(SortOrder::Increasing)
    );
    let sort = sort.toggle_size();
    assert_eq!(
        ordered_kinds(&kinds, sort, &state, units),
        [Part, SheetMetal]
    );
    assert_eq!(
        ordered_kinds(&kinds, sort.toggle_size(), &state, units),
        [SheetMetal, Part]
    );
    assert_eq!(
        ordered_kinds(&kinds, sort, &GuiCommandState::default(), units),
        kinds
    );
}
#[test]
fn an_unmeasured_template_says_so() {
    assert_eq!(size_text(None), "-");
    assert_eq!(size_text(Some(120)), "120 B");
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

#[test]
fn selecting_a_file_menu_kind_preserves_units_and_clears_drawing_preview() {
    let mut chooser = TemplateChooser::default();
    chooser.go_to(TemplateFolder::Units(GuiTemplateUnits::Inch));
    chooser.drawing_layout = Some(DrawingLayout::Assembly);
    chooser.request_kind(GuiTemplateKind::Drawing);
    assert_eq!(
        chooser.selected,
        Some(
            GuiDocumentTemplate {
                kind: GuiTemplateKind::Drawing,
                units: GuiTemplateUnits::Inch,
            }
            .into()
        )
    );
    assert!(chooser.requested);
    assert_eq!(chooser.drawing_layout, None);
    assert!(!chooser.can_create(&GuiCommandState::default()));
}
