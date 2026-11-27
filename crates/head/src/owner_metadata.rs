// SPDX-License-Identifier: GPL-2.0-only
//! Guard typed editor requests against concurrent changes to document properties.
use oblikovati_api::{types::VariantValue, wire::SetPropertyArgs};
use oblikovati_model::property::DocumentProperties;

pub(super) fn validate_typed_property(
    properties: &DocumentProperties,
    request: &SetPropertyArgs,
) -> Result<(), String> {
    match &request.value.value {
        VariantValue::Uninitialized => {
            return Err(format!(
                "property {:?}/{:?}: expected a supported initialized value",
                request.set, request.name
            ));
        }
        VariantValue::Double(value) if !value.is_finite() => {
            return Err(format!(
                "property {:?}/{:?} value {value:?}: expected a finite number",
                request.set, request.name
            ));
        }
        _ => {}
    }
    let Some(property) = properties.get(&request.set, &request.name) else {
        return Ok(());
    };
    if !property.from_parameter.is_empty()
        || std::mem::discriminant(&property.value) != std::mem::discriminant(&request.value.value)
    {
        return Err(format!(
            "property {:?}/{:?}: expected an editable property of the same type",
            request.set, request.name
        ));
    }
    Ok(())
}

impl<
    O: super::TransactionObserver,
    W: super::WorkspaceObserver,
    F: super::DocumentFiles,
    G: super::GuidSource,
    R: super::DocumentObserverFactory<Observer = O> + super::HeadInvalidation,
    S: super::DispatchWakeup,
> super::DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn set_typed_document_property(
        &mut self,
        request: &SetPropertyArgs,
    ) -> Result<bool, String> {
        if self.files.workspace().active_document() != Some(request.document) {
            return Err(format!(
                "document {}: expected the active document for iProperties",
                request.document
            ));
        }
        let session = self
            .files
            .workspace()
            .session(request.document)
            .map_err(|cause| cause.to_string())?;
        validate_typed_property(&session.snapshot().properties, request)?;
        oblikovati_api::client::Client::new(self.document_host())
            .documents()
            .set_property(request)
            .map_err(|cause| format!("property {:?}/{:?}: {cause}", request.set, request.name))?;
        Ok(true)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use oblikovati_api::types::Variant;
    use oblikovati_model::property::SUMMARY_INFORMATION;

    #[test]
    fn typed_edits_refuse_type_changes_parameter_sources_and_nonfinite_values() {
        use oblikovati_api::{json::Bytes, types::Variant};
        use oblikovati_model::property::DocumentProperty;
        let values = [
            VariantValue::Integer(1),
            VariantValue::Double(1.5),
            VariantValue::Boolean(true),
            VariantValue::Bytes(Bytes(Some(vec![0, 255]))),
            VariantValue::String("text".into()),
        ];
        for value in values {
            let mut properties = DocumentProperties::default();
            properties
                .put(SUMMARY_INFORMATION, "value", value.clone())
                .unwrap();
            let request = SetPropertyArgs {
                set: SUMMARY_INFORMATION.into(),
                name: "value".into(),
                value: Variant {
                    value,
                    unit: String::new(),
                },
                ..Default::default()
            };
            assert!(validate_typed_property(&properties, &request).is_ok());
            properties
                .restore_property(
                    SUMMARY_INFORMATION,
                    DocumentProperty {
                        name: "value".into(),
                        value: request.value.value.clone(),
                        from_parameter: "source".into(),
                    },
                )
                .unwrap();
            assert!(validate_typed_property(&properties, &request).is_err());
        }
        for value in [VariantValue::Uninitialized, VariantValue::Double(f64::NAN)] {
            let request = SetPropertyArgs {
                value: Variant {
                    value,
                    unit: String::new(),
                },
                ..Default::default()
            };
            assert!(validate_typed_property(&DocumentProperties::default(), &request).is_err());
        }
    }

    #[test]
    fn stale_text_edit_does_not_replace_a_number() {
        let mut properties = DocumentProperties::default();
        properties
            .put(SUMMARY_INFORMATION, "Title", VariantValue::Integer(1))
            .unwrap();
        let request = SetPropertyArgs {
            set: SUMMARY_INFORMATION.into(),
            name: "Title".into(),
            value: Variant {
                value: VariantValue::String("Motor".into()),
                unit: String::new(),
            },
            ..Default::default()
        };
        assert!(validate_typed_property(&properties, &request).is_err());
    }
}
