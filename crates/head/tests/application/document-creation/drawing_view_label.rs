// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::wire::{DrawingViewInfo, SetViewLabelArgs};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};

#[test]
fn drawing_view_label_edits_preserve_omissions_and_round_trip() {
    let (mut owner, _) = owner();
    super::super::part_scene::open_prism(&mut owner, false);
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    let client = Client::new(owner.document_host());
    let default = super::drawing_view_rotation::add_top_view(&client);
    assert_eq!(default.label, "Top  SCALE 1:1");
    let edit = |args| {
        client
            .drawing_views()
            .set_label(&SetViewLabelArgs {
                name: "Top".into(),
                ..args
            })
            .unwrap()
            .views
            .unwrap()
            .remove(0)
    };
    let custom = edit(SetViewLabelArgs {
        text: Some("COVER PLAN".to_owned().into()),
        show_name: Some(false.into()),
        label_xmm: Some(40_f64.into()),
        label_ymm: Some(65_f64.into()),
        ..Default::default()
    });
    assert_eq!(
        custom,
        DrawingViewInfo {
            label: "COVER PLAN".into(),
            show_name: false,
            label_xmm: 40.,
            label_ymm: 65.,
            ..default
        }
    );
    assert!(
        client
            .drawing_views()
            .set_label(&SetViewLabelArgs {
                name: "Missing".into(),
                ..Default::default()
            })
            .is_err()
    );
    let hidden = edit(SetViewLabelArgs {
        show_label: Some(false.into()),
        label_xmm: Some(50_f64.into()),
        ..Default::default()
    });
    assert_eq!(
        hidden,
        DrawingViewInfo {
            label: String::new(),
            show_label: false,
            ..custom.clone()
        }
    );
    let listed = || {
        client
            .drawing_views()
            .list(&oblikovati_api::json::EmptyObject {})
            .unwrap()
            .views
            .unwrap()
            .remove(0)
    };
    client.transactions().undo().unwrap();
    assert_eq!(listed(), custom);
    client.transactions().redo().unwrap();
    assert_eq!(listed(), hidden);
    let path = "/memory/view-label.odd";
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: drawing,
            new_full_document_name: path.into(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: drawing,
            force: true,
        })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(listed(), hidden);
    assert_eq!(
        edit(SetViewLabelArgs {
            show_label: Some(true.into()),
            ..Default::default()
        }),
        custom
    );
    let restored = edit(SetViewLabelArgs {
        text: Some(String::new().into()),
        label_xmm: Some(0_f64.into()),
        label_ymm: Some(0_f64.into()),
        ..Default::default()
    });
    assert_eq!(
        restored,
        DrawingViewInfo {
            label: "SCALE 1:1".into(),
            label_xmm: 0.,
            label_ymm: 0.,
            ..custom
        }
    );
    assert!(
        edit(SetViewLabelArgs {
            show_scale: Some(false.into()),
            ..Default::default()
        })
        .label
        .is_empty()
    );
}
