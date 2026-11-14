// SPDX-License-Identifier: GPL-2.0-only

//! Parameter-set XML exchange kept behind the automation boundary.

use oblikovati_api::{
    Error,
    types::USER_PARAM,
    wire::{ParameterExportResult, ParameterImportArgs, ParameterImportResult},
};
use oblikovati_model::{
    history::RecipeReservations,
    parameter::{ParameterCollection, ParameterState, ParameterValue},
};
use quick_xml::{Reader, de::from_str, events::Event, se::to_string};
use serde::{Deserialize, Serialize};
use std::collections::BTreeSet;

#[derive(Debug, Deserialize, Serialize)]
#[serde(rename = "parameters")]
struct ExchangeSet {
    #[serde(rename = "parameter", default)]
    parameters: Vec<ExchangeParameter>,
}

#[derive(Clone, Debug, Deserialize, Serialize)]
struct ExchangeParameter {
    #[serde(rename = "@name")]
    name: String,
    #[serde(
        rename = "@valueType",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    value_type: String,
    #[serde(
        rename = "@expression",
        default,
        skip_serializing_if = "String::is_empty"
    )]
    expression: String,
    #[serde(rename = "@text", default, skip_serializing_if = "String::is_empty")]
    text: String,
    #[serde(rename = "@bool", default, skip_serializing_if = "is_false")]
    boolean: bool,
    #[serde(rename = "@comment", default, skip_serializing_if = "String::is_empty")]
    comment: String,
    #[serde(rename = "@isKey", default, skip_serializing_if = "is_false")]
    is_key: bool,
    #[serde(
        rename = "@exposedAsProperty",
        default,
        skip_serializing_if = "is_false"
    )]
    exposed_as_property: bool,
}

/// Render user parameters in exchange XML; example: `export(&parameters)?`.
pub(super) fn export(
    parameters: &ParameterCollection<u64>,
) -> Result<ParameterExportResult, Error> {
    let parameters = parameters
        .ids()
        .iter()
        .filter_map(|id| parameters.get(*id))
        .filter(|state| state.kind == USER_PARAM)
        .map(export_parameter)
        .collect();
    let xml = to_string(&ExchangeSet { parameters })
        .map_err(|error| Error(format!("parameters.export: XML encoding failed: {error}")))?;
    Ok(ParameterExportResult { xml })
}

/// Apply an XML document; example: `import(parameters, ids, request)?`.
pub(super) fn import(
    parameters: &mut ParameterCollection<u64>,
    reservations: &mut RecipeReservations<'_>,
    request: ParameterImportArgs,
) -> Result<ParameterImportResult, Error> {
    let set = parse(&request.xml)?;
    let mut result = ParameterImportResult::default();
    for entry in set.parameters {
        apply_entry(parameters, reservations, entry, &mut result)?;
    }
    Ok(result)
}

fn export_parameter(state: &ParameterState<u64>) -> ExchangeParameter {
    let mut entry = ExchangeParameter {
        name: state.name.clone(),
        value_type: String::new(),
        expression: state.source.clone(),
        text: String::new(),
        boolean: false,
        comment: state.annotations.comment.clone(),
        is_key: state.annotations.is_key,
        exposed_as_property: state.annotations.exposed_as_property,
    };
    match &state.last_good_value {
        Some(ParameterValue::Text(text)) => set_text(&mut entry, text),
        Some(ParameterValue::Boolean(value)) => set_boolean(&mut entry, *value),
        Some(ParameterValue::Numeric(_)) | None => {}
    }
    entry
}

fn set_text(entry: &mut ExchangeParameter, text: &str) {
    entry.value_type = "text".into();
    entry.expression.clear();
    entry.text = text.into();
}

fn set_boolean(entry: &mut ExchangeParameter, value: bool) {
    entry.value_type = "boolean".into();
    entry.expression.clear();
    entry.boolean = value;
}

fn parse(xml: &str) -> Result<ExchangeSet, Error> {
    validate_root(xml)?;
    let set: ExchangeSet = from_str(xml).map_err(|error| {
        Error(format!(
            "parameters.import XML {xml:?}: expected a parameter-set document: {error}"
        ))
    })?;
    validate(&set)?;
    Ok(set)
}

fn validate_root(xml: &str) -> Result<(), Error> {
    let mut reader = Reader::from_str(xml);
    loop {
        match reader.read_event() {
            Ok(Event::Start(tag)) | Ok(Event::Empty(tag))
                if tag.name().as_ref() == b"parameters" =>
            {
                return Ok(());
            }
            Ok(Event::Start(tag)) | Ok(Event::Empty(tag)) => {
                return Err(Error(format!(
                    "parameters.import root {:?}: expected <parameters>",
                    String::from_utf8_lossy(tag.name().as_ref())
                )));
            }
            Ok(Event::Eof) => {
                return Err(Error(
                    "parameters.import XML: expected a <parameters> root".into(),
                ));
            }
            Ok(_) => {}
            Err(error) => {
                return Err(Error(format!(
                    "parameters.import XML {xml:?}: expected a parameter-set document: {error}"
                )));
            }
        }
    }
}

fn validate(set: &ExchangeSet) -> Result<(), Error> {
    let mut names = BTreeSet::new();
    for entry in &set.parameters {
        validate_entry(entry)?;
        if !names.insert(&entry.name) {
            return Err(Error(format!(
                "parameters.import name {:?}: expected one entry",
                entry.name
            )));
        }
    }
    Ok(())
}

fn validate_entry(entry: &ExchangeParameter) -> Result<(), Error> {
    if entry.name.is_empty() {
        return Err(Error(format!(
            "parameters.import expression {:?}: expected an entry name",
            entry.expression
        )));
    }
    match entry.value_type.as_str() {
        "" if !entry.expression.is_empty() => Ok(()),
        "" => Err(Error(format!(
            "parameters.import parameter {:?}: expected a numeric expression",
            entry.name
        ))),
        "text" | "boolean" => Ok(()),
        value_type => Err(Error(format!(
            "parameters.import valueType {value_type:?}: expected text, boolean, or empty for numeric"
        ))),
    }
}

fn apply_entry(
    parameters: &mut ParameterCollection<u64>,
    reservations: &mut RecipeReservations<'_>,
    entry: ExchangeParameter,
    result: &mut ParameterImportResult,
) -> Result<(), Error> {
    let id = parameters.find(&entry.name);
    match id {
        Some(id) => update(parameters, id, &entry)?,
        None => create(parameters, reservations, &entry)?,
    }
    set_metadata(parameters, &entry)?;
    if id.is_some() {
        result.updated += 1;
    } else {
        result.added += 1;
    }
    Ok(())
}

fn create(
    parameters: &mut ParameterCollection<u64>,
    reservations: &mut RecipeReservations<'_>,
    entry: &ExchangeParameter,
) -> Result<(), Error> {
    let id = reservations
        .allocate()
        .map_err(|error| Error(error.to_string()))?;
    let result = match entry.value_type.as_str() {
        "text" => parameters.insert_text(id, &entry.name, &entry.text),
        "boolean" => parameters.insert_boolean(id, &entry.name, entry.boolean),
        "" => parameters.insert_expression(id, &entry.name, USER_PARAM, &entry.expression),
        _ => unreachable!("validated value type"),
    };
    result.map_err(|error| {
        Error(format!(
            "parameters.import parameter {:?}: {error}",
            entry.name
        ))
    })
}

fn update(
    parameters: &mut ParameterCollection<u64>,
    id: u64,
    entry: &ExchangeParameter,
) -> Result<(), Error> {
    let result = match entry.value_type.as_str() {
        "text" => parameters.set_text(id, &entry.text),
        "boolean" => parameters.set_boolean(id, entry.boolean),
        "" => parameters.set_expression(id, &entry.expression),
        _ => unreachable!("validated value type"),
    };
    result.map_err(|error| {
        Error(format!(
            "parameters.import parameter {:?}: {error}",
            entry.name
        ))
    })
}

fn set_metadata(
    parameters: &mut ParameterCollection<u64>,
    entry: &ExchangeParameter,
) -> Result<(), Error> {
    let id = parameters
        .find(&entry.name)
        .expect("created or resolved parameter");
    let mut annotations = parameters
        .get(id)
        .expect("resolved parameter")
        .annotations
        .clone();
    annotations.comment = entry.comment.clone();
    annotations.is_key = entry.is_key;
    annotations.exposed_as_property = entry.exposed_as_property;
    parameters
        .set_annotations(id, annotations)
        .map_err(|error| Error(error.to_string()))
}

fn is_false(value: &bool) -> bool {
    !*value
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_model::units::UnitsOfMeasure;

    #[test]
    fn parse_rejects_duplicate_and_incomplete_numeric_entries() {
        for xml in [
            "<parameters><parameter name=\"a\"/></parameters>",
            "<parameters><parameter name=\"a\" expression=\"1\"/><parameter name=\"a\" expression=\"2\"/></parameters>",
            "<other><parameter name=\"a\" expression=\"1\"/></other>",
        ] {
            assert!(parse(xml).is_err(), "{xml}");
        }
    }

    #[test]
    fn export_preserves_user_literals() {
        let mut parameters = ParameterCollection::new(UnitsOfMeasure::default());
        parameters.insert_text(1, "material", "steel").unwrap();
        parameters
            .insert_expression(2, "depth", USER_PARAM, "2 cm")
            .unwrap();
        let xml = export(&parameters).unwrap().xml;
        assert!(xml.contains("name=\"material\""));
        assert!(xml.contains("valueType=\"text\""));
        assert!(xml.contains("name=\"depth\""));
    }
}
