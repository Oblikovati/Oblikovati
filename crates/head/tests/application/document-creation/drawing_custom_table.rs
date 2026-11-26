// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_api::{
    client::Caller,
    json::EmptyObject,
    wire::{AddCustomTableArgs, DeleteAnnotationArgs},
};
use oblikovati_gui::{GuiDocumentTemplate, GuiTemplateKind, GuiTemplateUnits};
use oblikovati_model::drawing::DrawingAnnotationDefinition;

#[test]
fn drawing_custom_table_round_trips_ragged_rows() {
    let (mut owner, _) = owner();
    owner
        .create_document_template(GuiDocumentTemplate {
            kind: GuiTemplateKind::Drawing,
            units: GuiTemplateUnits::Metric,
        })
        .unwrap();
    let drawing = owner.documents().workspace().active_document().unwrap();
    let client = Client::new(owner.document_host());
    let request = table_input();
    let annotations = client.drawing_annotations();
    assert!(
        annotations
            .add_custom_table(&AddCustomTableArgs {
                headers: None,
                ..request.clone()
            })
            .is_err()
    );
    let table = annotations.add_custom_table(&request).unwrap().annotation;
    assert_eq!(
        (table.kind.as_str(), table.row_count, table.curve_count),
        ("customTable", 3, 8)
    );
    assert_eq!(table.name, "NOTE:1");
    assert_eq!(
        annotations
            .delete(&DeleteAnnotationArgs {
                name: table.name.clone()
            })
            .unwrap()
            .annotations,
        None
    );
    client.transactions().undo().unwrap();
    assert_eq!(
        annotations
            .list(&EmptyObject {})
            .unwrap()
            .annotations
            .as_deref(),
        Some(std::slice::from_ref(&table))
    );
    let reopened = reopen(&client, drawing, "/memory/custom-table.odd");
    assert_eq!(
        annotations
            .list(&EmptyObject {})
            .unwrap()
            .annotations
            .unwrap(),
        [table]
    );
    drop(client);
    let state = owner.gui_command_state().unwrap();
    assert!(state.drawing_projections[&reopened].error.is_none());
    assert_table_cells(
        &state.drawing_documents[&reopened].sheets()[0].annotations[0],
        &request,
    );
}

fn table_input() -> AddCustomTableArgs {
    AddCustomTableArgs {
        xmm: 80.,
        ymm: 180.,
        headers: Some(vec!["ZONE".into(), "FINISH".into()]),
        rows: Some(vec![
            Some(vec!["A".into(), "ANODIZE".into(), "EXTRA".into()]),
            None,
            Some(vec!["B".into()]),
        ]),
        ..Default::default()
    }
}

fn assert_table_cells(annotation: &DrawingAnnotationDefinition, request: &AddCustomTableArgs) {
    let DrawingAnnotationDefinition::CustomTable { headers, rows, .. } = annotation else {
        panic!("expected a persisted table, got {annotation:?}");
    };
    assert_eq!(Some(headers), request.headers.as_ref());
    assert_eq!(Some(rows), request.rows.as_ref());
    let table = annotation.authored_table_presentation().unwrap();
    assert_eq!(table.lines[0].start, [80., 180.]);
    assert_eq!(table.lines[0].end, [152., 180.]);
    assert_eq!(table.lines.last().unwrap().end, [152., 148.]);
    assert_eq!(table.labels[0].center, [98., 176.]);
    let cells: Vec<_> = table
        .labels
        .iter()
        .map(|label| label.text.as_str())
        .collect();
    assert_eq!(cells, ["ZONE", "FINISH", "A", "ANODIZE", "", "", "B", ""]);
}

pub(super) fn reopen(client: &Client<impl Caller>, id: u64, path: &str) -> u64 {
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document: id,
            new_full_document_name: path.into(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs { id, force: true })
        .unwrap();
    client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            ..Default::default()
        })
        .unwrap()
        .id
}
