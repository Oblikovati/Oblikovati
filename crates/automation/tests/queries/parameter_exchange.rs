// SPDX-License-Identifier: GPL-2.0-only

use super::parameters::parameter_workspace;
use super::*;
use oblikovati_automation::SessionHost;
use oblikovati_model::parameter::ParameterAnnotations;

#[test]
fn parameter_exchange_round_trips_user_values_metadata_and_undo() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters
        .insert_expression(1, "width", USER_PARAM, "4 cm")
        .unwrap();
    parameters
        .insert_expression(2, "model", MODEL_PARAM, "2 cm")
        .unwrap();
    parameters
        .set_annotations(
            1,
            ParameterAnnotations {
                comment: "outer width".into(),
                is_key: true,
                ..Default::default()
            },
        )
        .unwrap();
    let (mut workspace, id) = parameter_workspace(parameters);
    let xml = {
        let client = Client::new(SessionHost::new(&mut workspace));
        client.parameters().export().unwrap().xml
    };
    assert!(xml.contains("name=\"width\""));
    assert!(!xml.contains("name=\"model\""));
    let imported = concat!(
        "<parameters>",
        "<parameter name=\"width\" expression=\"7 cm\" comment=\"changed\" isKey=\"true\" exposedAsProperty=\"true\"/>",
        "<parameter name=\"material\" valueType=\"text\" text=\"steel\"/>",
        "<parameter name=\"vented\" valueType=\"boolean\" bool=\"true\"/>",
        "</parameters>"
    );
    let result = Client::new(SessionHost::new(&mut workspace))
        .parameters()
        .import(&ParameterImportArgs {
            xml: imported.into(),
        })
        .unwrap();
    assert_eq!((result.added, result.updated), (2, 1));
    let snapshot = workspace.session(id).unwrap().snapshot();
    assert_eq!(snapshot.parameters.get(1).unwrap().source, "7 cm");
    assert_eq!(
        snapshot.parameters.get(1).unwrap().annotations.comment,
        "changed"
    );
    assert!(
        snapshot
            .parameters
            .get(1)
            .unwrap()
            .annotations
            .exposed_as_property
    );
    assert!(
        matches!(snapshot.parameters.get(snapshot.parameters.find("material").unwrap()).unwrap().current(), Some(oblikovati_model::parameter::ParameterValue::Text(value)) if value == "steel")
    );
    workspace.commands(id).unwrap().undo().unwrap();
    assert!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .find("material")
            .is_none()
    );
    assert_eq!(
        workspace
            .session(id)
            .unwrap()
            .snapshot()
            .parameters
            .get(1)
            .unwrap()
            .source,
        "4 cm"
    );
}

#[test]
fn failed_parameter_exchange_keeps_the_document_unchanged() {
    let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
    parameters.insert_text(1, "material", "steel").unwrap();
    let (mut workspace, id) = parameter_workspace(parameters);
    let error = Client::new(SessionHost::new(&mut workspace)).parameters().import(&ParameterImportArgs { xml: "<parameters><parameter name=\"new\" expression=\"2 cm\"/><parameter name=\"material\" expression=\"3 cm\"/></parameters>".into() }).unwrap_err();
    assert!(error.to_string().contains("material"));
    let parameters = &workspace.session(id).unwrap().snapshot().parameters;
    assert!(parameters.find("new").is_none());
    assert_eq!(parameters.text_value(1).unwrap(), "steel");
    assert!(workspace.session(id).unwrap().undo_labels().is_empty());
}
