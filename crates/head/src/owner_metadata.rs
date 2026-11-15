// SPDX-License-Identifier: GPL-2.0-only
//! Guard text-only editor requests against concurrent changes to document properties.
use oblikovati_api::{types::VariantValue, wire::SetPropertyArgs};
use oblikovati_model::property::DocumentProperties;

pub(super) fn validate_text_property(
    properties: &DocumentProperties,
    request: &SetPropertyArgs,
) -> Result<(), String> {
    if !matches!(request.value.value, VariantValue::String(_)) {
        return Err(format!(
            "property {:?}/{:?}: expected a text value",
            request.set, request.name
        ));
    }
    let Some(property) = properties.get(&request.set, &request.name) else {
        return Ok(());
    };
    if !property.from_parameter.is_empty() || !matches!(property.value, VariantValue::String(_)) {
        return Err(format!(
            "property {:?}/{:?}: expected an editable text property",
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
    pub(super) fn set_text_document_property(
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
        validate_text_property(&session.snapshot().properties, request)?;
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
        assert!(validate_text_property(&properties, &request).is_err());
    }
}
