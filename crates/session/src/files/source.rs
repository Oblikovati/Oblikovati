// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_api::types::{DOCUMENT_ASSEMBLY, DOCUMENT_PART, DocumentType};

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Resolve an open parameter source without I/O or activation; ambiguous names are errors.
    /// Example: `host.resolve_parameter_source("/parts/gear.opd")?`.
    pub fn resolve_parameter_source(&self, name: &str) -> Result<Option<u64>, FileWorkspaceError> {
        if name.is_empty() {
            return Ok(None);
        }
        let mut matching = self
            .retained
            .iter()
            .filter(|(_, file)| source_matches(file, name));
        let Some((&id, file)) = matching.next() else {
            return Ok(None);
        };
        if matching.next().is_some() {
            return Err(FileWorkspaceError::AmbiguousSource(name.into()));
        }
        let kind = DocumentType(file.package.manifest().document_type);
        Ok(matches!(kind, DOCUMENT_PART | DOCUMENT_ASSEMBLY).then_some(id))
    }
}

fn source_matches(file: &RetainedFile, name: &str) -> bool {
    match &file.path {
        Some(path) => path == Path::new(name),
        None => file.display_override.as_deref() == Some(name),
    }
}

impl<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles> FileWorkspace<O, W, F> {
    /// Read healthy numeric user values in the target's working units, without changing either owner.
    /// Example: `host.derived_source_values("/parts/gear.opd", target_id)?` before linking or refreshing.
    pub fn derived_source_values(
        &self,
        name: &str,
        target: u64,
    ) -> Result<Option<BTreeMap<String, oblikovati_model::quantity::Quantity>>, FileWorkspaceError>
    {
        let target = self
            .workspace
            .session(target)
            .map_err(FileWorkspaceError::Workspace)?;
        let Some(source) = self.resolve_parameter_source(name)? else {
            return Ok(None);
        };
        let source = self
            .workspace
            .session(source)
            .map_err(FileWorkspaceError::Workspace)?;
        source_values(
            name,
            &source.snapshot().parameters,
            target.snapshot().parameters.units(),
        )
        .map(Some)
    }
}

fn source_values(
    name: &str,
    parameters: &oblikovati_model::parameter::ParameterCollection<u64>,
    target: &oblikovati_model::units::UnitsOfMeasure,
) -> Result<BTreeMap<String, oblikovati_model::quantity::Quantity>, FileWorkspaceError> {
    parameters
        .ids()
        .iter()
        .filter_map(|id| {
            let parameter = parameters.get(*id)?;
            if parameter.kind != oblikovati_api::types::USER_PARAM {
                return None;
            }
            let quantity = parameter.current_value()?;
            Some(
                parameters
                    .units()
                    .transfer_quantity(quantity, target)
                    .map(|quantity| (parameter.name.clone(), quantity))
                    .map_err(|error| FileWorkspaceError::SourceValue {
                        source: name.into(),
                        parameter: parameter.name.clone(),
                        reason: error.to_string(),
                    }),
            )
        })
        .collect()
}
