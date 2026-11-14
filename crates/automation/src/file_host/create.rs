// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::*;
use oblikovati_persistence::package::DocumentPackage;

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileContext<'_, O, W, F> {
    pub(super) fn create(&mut self, request: CreateDocumentArgs) -> Result<Vec<u8>, Error> {
        let package = creation_package(&request)?;
        let observer = open::create_observer(&mut self.observers).map_err(Error)?;
        let registered = self
            .files
            .create_unbound(package, observer, self.version, self.source)
            .map_err(|error| Error(error.to_string()))?;
        let response = self
            .files
            .document_info(registered.session_id)
            .map_err(|error| Error(error.to_string()))?;
        encode(&response)
    }
}

/// Prepare the document recipe used by creation; example: `creation_package(&request)?`.
pub fn creation_package(request: &CreateDocumentArgs) -> Result<DocumentPackage, Error> {
    if request.name.is_empty() {
        return Err(Error(
            "documents.create name \"\": expected a nonempty document name".into(),
        ));
    }
    let kind = creation_kind(&request.type_)?;
    let sheet_metal = kind == DOCUMENT_PART && request.sub_type == SUB_TYPE_SHEET_METAL_PART.0;
    if !request.sub_type.is_empty() && !sheet_metal {
        return Err(Error(format!(
            "documents.create subType {:?}: expected empty, or the sheet-metal flavor of a part; subtype registration and flavor setup are not yet ported",
            request.sub_type
        )));
    }
    let package = DocumentPackage::empty(kind, &request.name);
    Ok(if sheet_metal {
        package.with_sub_type(&request.sub_type)
    } else {
        package
    })
}

fn creation_kind(name: &str) -> Result<DocumentType, Error> {
    let kind = match name.trim().to_lowercase().as_str() {
        "part" => DOCUMENT_PART,
        "assembly" => DOCUMENT_ASSEMBLY,
        "drawing" => DOCUMENT_DRAWING,
        "presentation" => DOCUMENT_PRESENTATION,
        _ => {
            return Err(Error(format!(
                "documents.create type {:?}: expected part, assembly, drawing, or presentation",
                name
            )));
        }
    };
    Ok(kind)
}
