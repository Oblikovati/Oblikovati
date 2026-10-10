// SPDX-License-Identifier: GPL-2.0-only

mod query;
mod target;
use oblikovati_api::{
    Error,
    types::{Variant, VariantValue},
    wire::*,
};
use oblikovati_model::attribute::{DocumentAttributes, ObjectAttribute};
pub(super) use query::dispatch;

fn info(set: &str, target: &str, attribute: &ObjectAttribute) -> AttributeInfo {
    AttributeInfo {
        set: set.into(),
        target: target.into(),
        name: attribute.name.clone(),
        value: Variant {
            value: attribute.value.clone(),
            unit: String::new(),
        },
    }
}

fn get(
    attributes: &DocumentAttributes,
    request: GetAttributeArgs,
) -> Result<AttributeResult, Error> {
    let key = target::key(&request.target)?;
    let attribute = attributes.get(&key, &request.set, &request.name);
    Ok(AttributeResult {
        found: attribute.is_some(),
        attribute: attribute
            .map(|attribute| info(&request.set, &request.target, attribute))
            .unwrap_or(AttributeInfo {
                set: request.set,
                name: request.name,
                target: request.target,
                value: Variant {
                    value: VariantValue::String(String::new()),
                    unit: String::new(),
                },
            }),
    })
}

pub(super) fn set(
    attributes: &mut DocumentAttributes,
    request: SetAttributeArgs,
) -> Result<AttributeResult, Error> {
    let key = target::key(&request.target)?;
    let value = match request.value.value {
        VariantValue::Uninitialized => VariantValue::String(String::new()),
        value => value,
    };
    attributes
        .put(&key, &request.set, &request.name, value)
        .map_err(|error| Error(error.to_string()))?;
    get(
        attributes,
        GetAttributeArgs {
            document: request.document,
            set: request.set,
            name: request.name,
            target: request.target,
        },
    )
}

pub(super) fn delete(
    attributes: &mut DocumentAttributes,
    request: DeleteAttributeArgs,
) -> Result<DeleteAttributeResult, Error> {
    let key = target::key(&request.target)?;
    if !request.name.is_empty() {
        return Ok(DeleteAttributeResult {
            removed: i64::from(attributes.remove(&key, &request.set, &request.name)),
        });
    }
    let count = attributes
        .set(&key, &request.set)
        .map_or(0, |set| set.attributes().len());
    let removed = i64::try_from(count)
        .map_err(|_| Error(format!("attribute count {count}: expected an i64 count")))?;
    attributes.remove_set(&key, &request.set);
    Ok(DeleteAttributeResult { removed })
}
