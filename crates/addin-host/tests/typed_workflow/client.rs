// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) struct WorkflowReport {
    pub(super) created: u64,
    pub(super) reopened: u64,
    pub(super) radius_bits: u64,
    pub(super) parameter: String,
}
pub(super) fn execute<T: Caller>(client: Client<T>, path: &Path) -> WorkflowReport {
    let created = create_part(&client);
    let circle = create_constrained_circle(&client);
    let after = edit_with_history(&client, circle);
    let key = circle_info(&client, circle).reference_key;
    let reopened = save_reopen(&client, created, path);
    assert_eq!(circle_radius(&client, circle).to_bits(), after.to_bits());
    assert_eq!(circle_info(&client, circle).reference_key, key);
    assert_width_expression(&client);
    WorkflowReport {
        created,
        reopened,
        radius_bits: after.to_bits(),
        parameter: "width".into(),
    }
}
fn parameter_input(name: &str, expression: &str) -> ParameterSetArgs {
    ParameterSetArgs {
        name: name.into(),
        expression: expression.into(),
        ..Default::default()
    }
}
fn create_constrained_circle<T: Caller>(client: &Client<T>) -> u64 {
    let circle = add_circle(client);
    ground_circle(client, circle);
    let dimension = client
        .dimension()
        .add(&AddDimensionArgs {
            kind: "radius".into(),
            entities: Some(vec![circle]),
            expression: "width".into(),
            ..Default::default()
        })
        .unwrap();
    assert_eq!(dimension.dof, 0);
    circle
}
fn circle_radius<T: Caller>(client: &Client<T>, circle: u64) -> f64 {
    let status = client
        .sketch()
        .constraint_status(&Default::default())
        .unwrap();
    assert_eq!(status.dof, 0);
    circle_info(client, circle).radius
}
fn save_reopen<T: Caller>(client: &Client<T>, document: u64, path: &Path) -> u64 {
    let path = path.to_str().unwrap();
    save_and_close(client, document, path);
    let reopened = client
        .documents()
        .open(&OpenDocumentArgs {
            full_document_name: path.into(),
            visible: true,
            defer_content: false,
        })
        .unwrap();
    assert!(!reopened.dirty);
    reopened.id
}

fn create_part<T: Caller>(client: &Client<T>) -> u64 {
    let created = client
        .documents()
        .create(&CreateDocumentArgs {
            type_: "part".into(),
            name: "Native part".into(),
            ..Default::default()
        })
        .unwrap();
    client
        .parameters()
        .add(&parameter_input("width", "20 mm"))
        .unwrap();
    client.sketch().create(&Default::default()).unwrap();
    created.id
}
fn edit_with_history<T: Caller>(client: &Client<T>, circle: u64) -> f64 {
    let before = circle_radius(client, circle);
    client
        .parameters()
        .set(&parameter_input("width", "35 mm"))
        .unwrap();
    let after = circle_radius(client, circle);
    assert!(after > before);
    client.transactions().undo().unwrap();
    assert_eq!(circle_radius(client, circle).to_bits(), before.to_bits());
    client.transactions().redo().unwrap();
    assert_eq!(circle_radius(client, circle).to_bits(), after.to_bits());
    after
}
fn add_circle<T: Caller>(client: &Client<T>) -> u64 {
    client
        .sketch()
        .add_entity(&AddSketchEntityArgs {
            kind: "circle".into(),
            points: Some(vec![Some(vec![0.0, 0.0])]),
            radius: "width".into(),
            ..Default::default()
        })
        .unwrap()
        .entity_id
}
fn ground_circle<T: Caller>(client: &Client<T>, circle: u64) {
    client
        .constrain()
        .add(&AddConstraintArgs {
            kind: "ground".into(),
            entities: Some(vec![circle]),
            ..Default::default()
        })
        .unwrap();
}
fn save_and_close<T: Caller>(client: &Client<T>, document: u64, path: &str) {
    client
        .documents()
        .save_as(&SaveDocumentAsArgs {
            document,
            new_full_document_name: path.into(),
        })
        .unwrap();
    client
        .documents()
        .close(&CloseDocumentArgs {
            id: document,
            force: false,
        })
        .unwrap();
}

fn circle_info<T: Caller>(client: &Client<T>, circle: u64) -> SketchEntityInfo {
    client
        .sketch()
        .entities(&Default::default())
        .unwrap()
        .entities
        .unwrap()
        .into_iter()
        .find(|entry| entry.id == circle)
        .unwrap()
}
fn assert_width_expression<T: Caller>(client: &Client<T>) {
    let width = client
        .parameters()
        .get(&ParameterNameArgs {
            name: "width".into(),
        })
        .unwrap();
    assert_eq!(width.expression, "35 mm");
}
