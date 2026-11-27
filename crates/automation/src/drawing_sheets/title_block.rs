// SPDX-License-Identifier: GPL-2.0-only
//! Resolve authored title-block bindings from current document property values.
use super::*;
use oblikovati_api::{
    types::VariantValue,
    wire::{TitleBlockField, TitleBlockFieldsArgs, TitleBlockFieldsResult},
};
use oblikovati_model::{drawing::DrawingTitleProperty, property::DocumentProperties};

pub(super) fn fields<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    owner: u64,
    drawing: &DrawingDocument,
    args: TitleBlockFieldsArgs,
) -> Result<Vec<u8>, Error> {
    let index = drawing
        .named_or_active_sheet(&args.sheet)
        .map_err(api_error)?;
    let sheet = &drawing.sheets()[index];
    let Some(bindings) = sheet.title_block_fields().map_err(api_error)? else {
        return crate::encode(&TitleBlockFieldsResult::default());
    };
    let properties = referenced_properties(files, owner, drawing.model_reference())?;
    crate::encode(&TitleBlockFieldsResult {
        definition_name: sheet.title_block.clone(),
        fields: Some(
            bindings
                .iter()
                .map(|binding| resolve(binding, properties))
                .collect(),
        ),
    })
}

fn referenced_properties<'a, O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &'a FileWorkspace<O, W, F>,
    owner: u64,
    name: &str,
) -> Result<Option<&'a DocumentProperties>, Error> {
    let source = match files.drawing_reference(owner, name) {
        Some(source) => Some(source),
        None => files.resolve_parameter_source(name).map_err(api_error)?,
    };
    source
        .map(|source| {
            let session = files.workspace().session(source).map_err(api_error)?;
            Ok(&session.snapshot().properties)
        })
        .transpose()
}

fn resolve(
    binding: &DrawingTitleProperty,
    properties: Option<&DocumentProperties>,
) -> TitleBlockField {
    let value = properties
        .and_then(|properties| properties.get(binding.property_set, binding.property_name))
        .map(|property| property_text(&property.value))
        .unwrap_or_default();
    TitleBlockField {
        name: binding.label.into(),
        value,
        source: format!("{}:{}", binding.property_set, binding.property_name),
    }
}

fn property_text(value: &VariantValue) -> String {
    match value {
        VariantValue::String(value) => value.clone(),
        VariantValue::Integer(value) => value.to_string(),
        VariantValue::Double(value) => decimal_text(*value),
        VariantValue::Boolean(value) => value.to_string(),
        VariantValue::Bytes(_) | VariantValue::Uninitialized => String::new(),
    }
}

fn decimal_text(value: f64) -> String {
    let scientific = format!("{value:e}");
    if let Some((mantissa, exponent)) = scientific.split_once('e')
        && let Ok(exponent) = exponent.parse::<i32>()
        && !(-4..6).contains(&exponent)
    {
        // Match the Go title block's shortest general format for numeric properties.
        return format!("{mantissa}e{exponent:+03}");
    }
    value.to_string()
}
