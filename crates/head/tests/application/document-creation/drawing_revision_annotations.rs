// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    json::EmptyObject,
    wire::{
        AddDatumFeatureArgs, AddRevisionCloudArgs, AddRevisionTableArgs, AddRevisionTagArgs,
        AddSurfaceTextureArgs, DeleteAnnotationArgs, DrawingAnnotationInfo, RevisionTableRow,
    },
};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};
use oblikovati_model::drawing::DrawingAnnotationDefinition;

#[test]
fn drawing_revision_table_and_tag_have_independent_history_and_persist() {
    let (mut owner, drawing) = drawing_owner();
    let client = Client::new(owner.document_host());
    let [table, tag] = add_revision_annotations(&client);
    client.transactions().undo().unwrap();
    assert_eq!(listed(&client).as_slice(), std::slice::from_ref(&table));
    client.transactions().redo().unwrap();
    assert_eq!(listed(&client), [table.clone(), tag.clone()]);
    let reopened =
        super::drawing_custom_table::reopen(&client, drawing, "/memory/revision-annotations.odd");
    assert_eq!(listed(&client), [table, tag]);
    drop(client);
    let state = owner.gui_command_state().unwrap();
    assert!(state.drawing_projections[&reopened].error.is_none());
    let sheet = &state.drawing_documents[&reopened].sheets()[0];
    assert_presentations(&sheet.annotations);
}

fn drawing_owner() -> (TestOwner, u64) {
    let (mut owner, _) = owner();
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    (owner, drawing)
}

#[test]
fn drawing_surface_texture_retains_all_material_removal_variants() {
    let (mut owner, drawing) = drawing_owner();
    let client = Client::new(owner.document_host());
    let added = add_surface_symbols(&client);
    client.transactions().undo().unwrap();
    assert_eq!(listed(&client), added[..2]);
    client.transactions().redo().unwrap();
    assert_eq!(listed(&client), added);
    let reopened =
        super::drawing_custom_table::reopen(&client, drawing, "/memory/surface-texture.odd");
    assert_eq!(listed(&client), added);
    drop(client);
    let state = owner.gui_command_state().unwrap();
    assert!(state.drawing_projections[&reopened].error.is_none());
    let sheet = &state.drawing_documents[&reopened].sheets()[0];
    assert_finish_geometry(&sheet.annotations);
    let client = Client::new(owner.document_host());
    let deleted = client
        .drawing_annotations()
        .delete(&DeleteAnnotationArgs {
            name: added[2].name.clone(),
        })
        .unwrap();
    assert_eq!(deleted.annotations.as_deref(), Some(&added[..2]));
}

fn add_surface_symbols(client: &Client<impl Caller>) -> [DrawingAnnotationInfo; 3] {
    let annotations = client.drawing_annotations();
    assert!(
        annotations
            .add_surface_texture(&AddSurfaceTextureArgs {
                material_removal: "unknown".into(),
                ..Default::default()
            })
            .is_err()
    );
    [
        ("", "", 3),
        ("required", "Ra 1.6", 4),
        ("prohibited", "Ra 3.2", 27),
    ]
    .map(|(material_removal, text, curves)| {
        let info = annotations
            .add_surface_texture(&AddSurfaceTextureArgs {
                xmm: 60.,
                ymm: 80.,
                roughness: text.into(),
                material_removal: material_removal.into(),
                ..Default::default()
            })
            .unwrap()
            .annotation;
        assert_eq!(
            (info.kind.as_str(), info.tag.as_str(), info.curve_count),
            ("surfaceTexture", text, curves)
        );
        info
    })
}

fn assert_finish_geometry(annotations: &[DrawingAnnotationDefinition]) {
    let basic = annotations[0].surface_texture_presentation().unwrap();
    let required = annotations[1].surface_texture_presentation().unwrap();
    let prohibited = annotations[2].surface_texture_presentation().unwrap();
    let base = [
        [[60., 80.], [57., 84.]],
        [[60., 80.], [65., 88.8]],
        [[65., 88.8], [77., 88.8]],
    ];
    assert_eq!(basic.segments_mm, base);
    assert!(basic.roughness.is_empty());
    assert_eq!(required.segments_mm[..3], base);
    assert_eq!(required.segments_mm[3], [[57., 84.], [65., 88.8]]);
    assert_eq!(required.text_center_mm[0], 71.);
    assert!((required.text_center_mm[1] - 91.2).abs() < 1e-10);
    assert_eq!(prohibited.segments_mm[..3], base);
    let circle = &prohibited.segments_mm[3..];
    assert_eq!(circle.len(), 24);
    assert_eq!(circle[0][0], circle[23][1]);
    assert_eq!(circle[0][0], [61.8, 81.8]);
}

#[test]
fn drawing_datum_feature_round_trips_box_stem_and_filled_triangle() {
    let (mut owner, drawing) = drawing_owner();
    let client = Client::new(owner.document_host());
    let datum = add_datum(&client);
    client.transactions().undo().unwrap();
    assert!(listed(&client).is_empty());
    client.transactions().redo().unwrap();
    assert_eq!(listed(&client).as_slice(), std::slice::from_ref(&datum));
    let reopened =
        super::drawing_custom_table::reopen(&client, drawing, "/memory/datum-feature.odd");
    assert_eq!(listed(&client).as_slice(), std::slice::from_ref(&datum));
    drop(client);
    let state = owner.gui_command_state().unwrap();
    assert!(state.drawing_projections[&reopened].error.is_none());
    let sheet = &state.drawing_documents[&reopened].sheets()[0];
    assert_datum_presentation(&sheet.annotations[0]);
    let client = Client::new(owner.document_host());
    let deleted = client
        .drawing_annotations()
        .delete(&DeleteAnnotationArgs { name: datum.name })
        .unwrap();
    assert!(deleted.annotations.is_none());
}

fn add_datum(client: &Client<impl Caller>) -> DrawingAnnotationInfo {
    let annotations = client.drawing_annotations();
    let request = AddDatumFeatureArgs {
        name: "Reference plane".into(),
        xmm: 60.,
        ymm: 80.,
        letter: "A".into(),
    };
    assert!(
        annotations
            .add_datum_feature(&AddDatumFeatureArgs {
                letter: String::new(),
                ..request.clone()
            })
            .is_err()
    );
    let info = annotations.add_datum_feature(&request).unwrap().annotation;
    assert_eq!(
        (info.kind.as_str(), info.tag.as_str(), info.curve_count),
        ("datumFeature", "A", 10)
    );
    info
}

fn assert_datum_presentation(annotation: &DrawingAnnotationDefinition) {
    let datum = annotation.datum_feature_presentation().unwrap();
    assert_eq!(datum.letter, "A");
    assert_eq!(datum.text_center_mm, [64., 76.]);
    assert_eq!(
        datum.frame_and_stem_mm,
        [
            [[60., 80.], [68., 80.]],
            [[68., 80.], [68., 72.]],
            [[68., 72.], [60., 72.]],
            [[60., 72.], [60., 80.]],
            [[64., 72.], [64., 68.]],
        ]
    );
    assert_eq!(datum.triangle_mm, [[64., 68.], [66.5, 63.], [61.5, 63.]]);
}

#[test]
fn drawing_revision_cloud_has_outward_scallops_and_round_trips() {
    let (mut owner, drawing) = drawing_owner();
    let client = Client::new(owner.document_host());
    let cloud = add_cloud(&client);
    client.transactions().undo().unwrap();
    assert!(listed(&client).is_empty());
    client.transactions().redo().unwrap();
    assert_eq!(listed(&client).as_slice(), std::slice::from_ref(&cloud));
    let reopened =
        super::drawing_custom_table::reopen(&client, drawing, "/memory/revision-cloud.odd");
    assert_eq!(listed(&client).as_slice(), std::slice::from_ref(&cloud));
    drop(client);
    let state = owner.gui_command_state().unwrap();
    assert!(state.drawing_projections[&reopened].error.is_none());
    let sheet = &state.drawing_documents[&reopened].sheets()[0];
    assert_cloud_presentation(&sheet.annotations[0]);
    let client = Client::new(owner.document_host());
    assert!(
        client
            .drawing_annotations()
            .delete(&DeleteAnnotationArgs { name: cloud.name })
            .unwrap()
            .annotations
            .is_none()
    );
}

fn add_cloud(client: &Client<impl Caller>) -> DrawingAnnotationInfo {
    let annotations = client.drawing_annotations();
    let request = AddRevisionCloudArgs {
        xmm: 60.,
        ymm: 80.,
        width_mm: 60.,
        height_mm: 40.,
        tag: "A".into(),
        ..Default::default()
    };
    for width_mm in [0., -1., 1e30] {
        assert!(
            annotations
                .add_revision_cloud(&AddRevisionCloudArgs {
                    width_mm,
                    ..request.clone()
                })
                .is_err()
        );
    }
    let cloud = annotations.add_revision_cloud(&request).unwrap().annotation;
    assert_eq!(
        (
            cloud.name.as_str(),
            cloud.kind.as_str(),
            cloud.tag.as_str(),
            cloud.curve_count
        ),
        ("NOTE:1", "revisionCloud", "A", 204)
    );
    cloud
}

fn assert_cloud_presentation(annotation: &DrawingAnnotationDefinition) {
    let cloud = annotation.revision_cloud_presentation().unwrap();
    assert_eq!(cloud.tag, "A");
    assert_eq!(cloud.label_center_mm(), [90., 126.]);
    let outline = cloud.visible_segments([420., 297.], 0.25);
    assert_eq!(outline.len(), 204);
    assert_eq!(outline[0][0], [60., 80.]);
    assert!((outline[2][1][0] - 63.).abs() < 1e-10);
    assert_eq!(outline[2][1][1], 77.);
    assert!(
        outline[60..102]
            .iter()
            .flatten()
            .all(|point| point[0] >= 120.)
    );
    assert_eq!(cloud.visible_segments([420., 297.], 10.).len(), 4);
    assert_eq!(cloud.visible_segments([90., 90.], 0.25).len(), 84);
}

fn add_revision_annotations(client: &Client<impl Caller>) -> [DrawingAnnotationInfo; 2] {
    let annotations = client.drawing_annotations();
    assert!(
        annotations
            .add_revision_table(&AddRevisionTableArgs::default())
            .is_err()
    );
    assert!(
        annotations
            .add_revision_tag(&AddRevisionTagArgs::default())
            .is_err()
    );
    let table = annotations
        .add_revision_table(&AddRevisionTableArgs {
            name: "Changes".into(),
            xmm: 50.,
            ymm: 180.,
            rows: Some(vec![
                RevisionTableRow {
                    revision: "A".into(),
                    date: "2026-11-26".into(),
                    description: "Released".into(),
                },
                RevisionTableRow {
                    revision: "B".into(),
                    description: "Bore enlarged".into(),
                    ..Default::default()
                },
            ]),
        })
        .unwrap()
        .annotation;
    let tag = annotations
        .add_revision_tag(&AddRevisionTagArgs {
            name: "B Marker".into(),
            xmm: 190.,
            ymm: 125.,
            revision: "B".into(),
        })
        .unwrap()
        .annotation;
    assert_eq!(
        (table.kind.as_str(), table.row_count, table.curve_count),
        ("revisionTable", 2, 8)
    );
    assert_eq!(
        (tag.kind.as_str(), tag.tag.as_str(), tag.curve_count),
        ("revisionTag", "B", 3)
    );
    [table, tag]
}

fn listed(client: &Client<impl Caller>) -> Vec<DrawingAnnotationInfo> {
    client
        .drawing_annotations()
        .list(&EmptyObject {})
        .unwrap()
        .annotations
        .unwrap_or_default()
}

fn assert_presentations(annotations: &[DrawingAnnotationDefinition]) {
    let table = annotations[0].authored_table_presentation().unwrap();
    assert_eq!(table.lines[0].start, [50., 180.]);
    assert_eq!(table.lines[0].end, [152., 180.]);
    let labels: Vec<_> = table.labels.iter().map(|cell| cell.text.as_str()).collect();
    assert_eq!(
        labels,
        [
            "REV",
            "DATE",
            "DESCRIPTION",
            "A",
            "2026-11-26",
            "Released",
            "B",
            "",
            "Bore enlarged"
        ]
    );
    let tag = annotations[1].revision_tag_presentation().unwrap();
    assert_eq!(tag.text, "B");
    assert_eq!(tag.text_center_mm, [190., 123.]);
    assert_eq!(
        tag.segments_mm,
        [
            [[190., 131.], [184., 119.]],
            [[184., 119.], [196., 119.]],
            [[196., 119.], [190., 131.]],
        ]
    );
}
