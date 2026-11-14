// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Client,
    types,
    wire::{CloseDocumentArgs, CreateDocumentArgs},
};

#[test]
fn zero_document_chooser_creates_each_document_kind() {
    for (template, kind, tab) in [
        ("Standard metric part", types::DOCUMENT_PART, "3D Model"),
        (
            "Standard metric assembly",
            types::DOCUMENT_ASSEMBLY,
            "Assemble",
        ),
        (
            "Metric drawing with title block",
            types::DOCUMENT_DRAWING,
            "Place Views",
        ),
        (
            "Standard metric presentation",
            types::DOCUMENT_PRESENTATION,
            "Presentation",
        ),
    ] {
        let preferences = tempfile::tempdir().unwrap();
        let mut native = NativeHeadWindow::open().unwrap();
        native.ribbon_settings_directory = Some(preferences.path().to_owned());
        let mut owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
        let mut window = scripted_window(native);
        crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner)
            .unwrap();
        application_bdd::rendered_label_center("Get Started");
        create::metric_template(&mut window, &mut owner, template);
        assert_eq!(
            owner.gui_command_state().unwrap().document_type,
            Some(kind),
            "template {template:?}; actions {:?}; notice {:?}",
            window.gui_actions,
            window.native.command_notice
        );
        application_bdd::rendered_label_center(tab);
        owner.finish().unwrap();
        window.finish().unwrap();
    }
}

#[test]
fn native_document_tabs_switch_the_visible_ribbon() {
    let preferences = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    native.ribbon_settings_directory = Some(preferences.path().to_owned());
    let mut owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
    let mut window = scripted_window(native);
    let kinds = [
        ("part", "Part ribbon", types::DOCUMENT_PART, "3D Model"),
        (
            "assembly",
            "Assembly ribbon",
            types::DOCUMENT_ASSEMBLY,
            "Assemble",
        ),
        (
            "drawing",
            "Drawing ribbon",
            types::DOCUMENT_DRAWING,
            "Place Views",
        ),
        (
            "presentation",
            "Presentation ribbon",
            types::DOCUMENT_PRESENTATION,
            "Presentation",
        ),
    ];
    let documents: Vec<_> = kinds
        .iter()
        .map(|(kind, name, _, _)| {
            Client::new(owner.document_host())
                .documents()
                .create(&CreateDocumentArgs {
                    type_: (*kind).into(),
                    name: (*name).into(),
                    ..Default::default()
                })
                .unwrap()
                .id
        })
        .collect();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    for ((_, name, kind, tab), document) in kinds.iter().zip(documents) {
        click_shell_position(
            &mut window,
            &mut owner,
            application_bdd::rendered_label_start(name),
        );
        let state = owner.gui_command_state().unwrap();
        assert_eq!(state.document, Some(document));
        assert_eq!(state.document_type, Some(*kind));
        application_bdd::rendered_label_center(tab);
    }
    owner.finish().unwrap();
    window.finish().unwrap();
}

#[test]
fn closing_last_document_restores_the_zero_document_ribbon() {
    let preferences = tempfile::tempdir().unwrap();
    let mut native = NativeHeadWindow::open().unwrap();
    native.ribbon_settings_directory = Some(preferences.path().to_owned());
    let mut owner = native_owner_at(native.wake.clone(), preferences.path()).unwrap();
    let mut window = scripted_window(native);
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    create::metric_part(&mut window, &mut owner);
    let document = owner.gui_command_state().unwrap().document.unwrap();
    Client::new(owner.document_host())
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: true,
        })
        .unwrap();
    crate::frames::run_frames(&mut window, HeadOptions { frames: Some(3) }, &mut owner).unwrap();
    let state = owner.gui_command_state().unwrap();
    assert!(state.documents.is_empty());
    assert!(state.document_type.is_none());
    application_bdd::rendered_label_center("Get Started");
    owner.finish().unwrap();
    window.finish().unwrap();
}
