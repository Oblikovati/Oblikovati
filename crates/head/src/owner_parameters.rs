// SPDX-License-Identifier: GPL-2.0-only
use super::*;
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn gui_parameter_rows(
        &self,
    ) -> Result<Vec<oblikovati_api::wire::ParameterInfo>, String> {
        let query = oblikovati_automation::SessionQueryHost::new(self.files.workspace());
        oblikovati_api::client::Client::new(query)
            .parameters()
            .list()
            .map(|reply| {
                reply
                    .parameters
                    .unwrap_or_default()
                    .into_iter()
                    .map(browser_parameter_display)
                    .collect()
            })
            .map_err(|cause| cause.to_string())
    }
}
// The browser needs displayable text even when the editor reports invalid model text (#3891).
fn browser_parameter_display(
    mut parameter: oblikovati_api::wire::ParameterInfo,
) -> oblikovati_api::wire::ParameterInfo {
    for value in [
        &mut parameter.name,
        &mut parameter.kind,
        &mut parameter.expression,
        &mut parameter.value,
        &mut parameter.health,
    ] {
        if value.contains('\0') {
            *value = value.replace('\0', "\\0");
        }
    }
    parameter
}

#[cfg(test)]
mod browser_display_tests {
    use super::*;

    #[test]
    fn nul_is_visible_in_every_browser_field_without_changing_other_text() {
        let raw = oblikovati_api::wire::ParameterInfo {
            name: "é\0name".into(),
            kind: "kind\0".into(),
            expression: "\0expr".into(),
            value: "a\0b\0c".into(),
            health: "health\0".into(),
        };
        let display = browser_parameter_display(raw.clone());
        assert_eq!(display.name, "é\\0name");
        assert_eq!(display.kind, "kind\\0");
        assert_eq!(display.expression, "\\0expr");
        assert_eq!(display.value, "a\\0b\\0c");
        assert_eq!(display.health, "health\\0");
        assert_eq!(raw.name, "é\0name");
        assert_eq!(browser_parameter_display(display.clone()), display);
    }
}
#[path = "owner_parameters/apply.rs"]
pub(super) mod apply;
#[path = "owner_parameters/create.rs"]
mod create;
#[path = "owner_parameters/derived.rs"]
pub(super) mod derived;
use oblikovati_api::client::Client;
use oblikovati_api::types::Tolerance;
use oblikovati_automation::{FileQueryHost, parameter_detail};
use oblikovati_gui::{
    GuiParameterGroup, GuiParameterLinkParameter, GuiParameterLinkSource, GuiParameterRow,
    GuiParameterSnapshot,
};
use oblikovati_model::{
    parameter::ParameterCollection,
    quantity::{Dimension, Quantity},
};
use oblikovati_persistence::file_access::DocumentFiles;
use oblikovati_session::{TransactionObserver, files::FileWorkspace, workspace::WorkspaceObserver};
pub(super) fn require_group_owner<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    key: &str,
    must_exist: bool,
) -> Result<(), String> {
    if files.workspace().active_document() != Some(document) {
        return Err(format!(
            "GUI parameter group document {document}: expected current document {:?}",
            files.workspace().active_document()
        ));
    }
    if must_exist
        && files
            .workspace()
            .session(document)
            .map_err(|cause| cause.to_string())?
            .snapshot()
            .parameters
            .group(key)
            .is_none()
    {
        return Err(format!(
            "GUI parameter group {key:?} in document {document}: expected an existing group key"
        ));
    }
    Ok(())
}
pub(super) fn group_member_name<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
    parameter: u64,
) -> Result<String, String> {
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| cause.to_string())?;
    session.snapshot().parameters.get(parameter)
        .map(|state| state.name.clone())
        .ok_or_else(|| format!("GUI parameter group member {parameter} in document {document}: expected a registered parameter"))
}
pub(super) fn owner_parameter_snapshot<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
>(
    files: &FileWorkspace<O, W, F>,
) -> Result<Option<GuiParameterSnapshot>, String> {
    let workspace = files.workspace();
    let Some(document) = workspace.active_document() else {
        return Ok(None);
    };
    let session = workspace
        .session(document)
        .map_err(|cause| cause.to_string())?;
    let parameters = &session.snapshot().parameters;
    let feature_consumers = feature_parameter_consumers(session.snapshot());
    let rows = parameters
        .ids()
        .iter()
        .map(|id| {
            projected_parameter_row(
                parameters,
                document,
                *id,
                feature_consumers.get(id).map(Vec::as_slice),
            )
        })
        .collect::<Result<Vec<_>, String>>()?;
    let linked_tables = Client::new(FileQueryHost::new(files))
        .parameters()
        .list_derived_tables()
        .map_err(|cause| cause.to_string())?
        .tables
        .unwrap_or_default();
    let link_sources = owner_link_sources(files, document)?;
    let groups = parameters
        .groups()
        .iter()
        .map(|group| {
            Ok(GuiParameterGroup {
                key: group.key.0.to_string(),
                display_name: group.display_name.clone(),
                client_id: group.client_id.clone(),
                members: parameters
                    .group_members(&group.key.0)
                    .map_err(|cause| cause.to_string())?,
            })
        })
        .collect::<Result<Vec<_>, String>>()?;
    Ok(Some(GuiParameterSnapshot {
        document,
        rows,
        linked_tables,
        link_sources,
        groups,
    }))
}
pub(super) fn owner_link_sources<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    target: u64,
) -> Result<Vec<GuiParameterLinkSource>, String> {
    files
        .workspace()
        .documents()
        .iter()
        .copied()
        .filter(|id| *id != target)
        .map(|id| owner_link_source(files, id))
        .collect::<Result<Vec<_>, _>>()
        .map(|sources| sources.into_iter().flatten().collect())
}
fn owner_link_source<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    document: u64,
) -> Result<Option<GuiParameterLinkSource>, String> {
    let source_document = files
        .parameter_source_name(document)
        .map_err(|cause| cause.to_string())?;
    if !resolves_exact_source(files, &source_document, document)? {
        return Ok(None);
    }
    let session = files
        .workspace()
        .session(document)
        .map_err(|cause| cause.to_string())?;
    let parameters = &session.snapshot().parameters;
    let parameters = parameters
        .ids()
        .iter()
        .filter_map(|id| {
            let state = parameters.get(*id)?;
            (state.kind == oblikovati_api::types::USER_PARAM && state.current_value().is_some())
                .then(|| GuiParameterLinkParameter {
                    parameter: *id,
                    name: state.name.clone(),
                })
        })
        .collect::<Vec<_>>();
    if parameters.is_empty() {
        return Ok(None);
    }
    Ok(Some(GuiParameterLinkSource {
        document,
        source_document,
        display_name: files
            .workspace()
            .display_name(document)
            .map_err(|cause| cause.to_string())?
            .into(),
        parameters,
    }))
}
fn resolves_exact_source<O: TransactionObserver, W: WorkspaceObserver, F: DocumentFiles>(
    files: &FileWorkspace<O, W, F>,
    source_document: &str,
    document: u64,
) -> Result<bool, String> {
    match files.resolve_parameter_source(source_document) {
        Ok(source) => Ok(source == Some(document)),
        Err(oblikovati_session::files::FileWorkspaceError::AmbiguousSource(_)) => Ok(false),
        Err(cause) => Err(cause.to_string()),
    }
}

fn projected_parameter_row(
    parameters: &ParameterCollection<u64>,
    document: u64,
    id: u64,
    feature_consumers: Option<&[(oblikovati_model::feature::FeatureId, String)]>,
) -> Result<GuiParameterRow, String> {
    let detail = parameter_detail(parameters, id)
        .map_err(|cause| format!("GUI parameter ID {id} in document {document}: {cause}"))?;
    Ok(GuiParameterRow {
        parameter: id,
        detail,
        is_length: parameters
            .inspection_value(id)
            .map_err(|cause| cause.to_string())?
            .is_some_and(|value| value.dimension == oblikovati_model::quantity::Dimension::Length),
        feature_consumers: feature_consumers.unwrap_or_default().to_vec(),
        tolerance_text: owner_tolerance_text(parameters, id)?,
        model_value_text: owner_model_value_text(parameters, id)?,
        has_groups: !parameters
            .groups_of(id)
            .map_err(|cause| cause.to_string())?
            .is_empty(),
    })
}

fn feature_parameter_consumers(
    snapshot: &oblikovati_model::history::DocumentSnapshot,
) -> std::collections::BTreeMap<u64, Vec<(oblikovati_model::feature::FeatureId, String)>> {
    use oblikovati_model::feature::FeatureReadKey;
    let mut consumers = std::collections::BTreeMap::<_, Vec<_>>::new();
    for feature in snapshot.work_geometry.features() {
        for key in feature.reads().keys() {
            if let FeatureReadKey::Parameter(parameter) = key {
                consumers
                    .entry(*parameter)
                    .or_default()
                    .push((feature.id(), feature.name().to_owned()));
            }
        }
    }
    consumers
}

fn owner_model_value_text(
    parameters: &ParameterCollection<u64>,
    id: u64,
) -> Result<String, String> {
    let value = parameters
        .inspection_value(id)
        .map_err(|cause| cause.to_string())?;
    value
        .map(|quantity| {
            parameters
                .units()
                .format_precise(quantity)
                .map_err(|cause| cause.to_string())
        })
        .transpose()
        .map(|value| value.unwrap_or_default())
}

fn owner_tolerance_text(parameters: &ParameterCollection<u64>, id: u64) -> Result<String, String> {
    let state = parameters
        .get(id)
        .ok_or_else(|| format!("GUI tolerance parameter {id}: expected a registered parameter"))?;
    if state.tolerance == Tolerance::default() {
        return Ok(String::new());
    }
    let Some(quantity) = parameters
        .inspection_value(id)
        .map_err(|cause| cause.to_string())?
    else {
        return Ok(String::new());
    };
    formatted_tolerance_band(parameters, quantity.dimension, &state.tolerance)
}
fn formatted_tolerance_band(
    parameters: &ParameterCollection<u64>,
    dimension: Dimension,
    band: &Tolerance,
) -> Result<String, String> {
    let upper = parameters
        .units()
        .format_precise(Quantity::new(band.upper, dimension))
        .map_err(|cause| cause.to_string())?;
    let lower = parameters
        .units()
        .format_precise(Quantity::new(band.lower, dimension))
        .map_err(|cause| cause.to_string())?;
    Ok(format!("{upper} / {lower}"))
}

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn apply_parameter_group_management(
        &mut self,
        edit: &oblikovati_gui::GuiParameterGroupAction,
    ) -> Result<bool, String> {
        use oblikovati_gui::GuiParameterGroupAction;
        let exists = !matches!(edit, GuiParameterGroupAction::Create { .. });
        let key = match edit {
            GuiParameterGroupAction::Create { internal_name, .. }
            | GuiParameterGroupAction::Rename { internal_name, .. }
            | GuiParameterGroupAction::Delete { internal_name, .. }
            | GuiParameterGroupAction::RemoveMember { internal_name, .. } => internal_name,
        };
        self::require_group_owner(&self.files, edit.document(), key, exists)?;
        let member_name = match edit {
            GuiParameterGroupAction::RemoveMember { parameter, .. } => Some(
                self::group_member_name(&self.files, edit.document(), *parameter)?,
            ),
            _ => None,
        };
        let client = oblikovati_api::client::Client::new(self.document_host());
        match edit {
            GuiParameterGroupAction::Create {
                internal_name,
                display_name,
                client_id,
                ..
            } => client
                .parameters()
                .add_group(&oblikovati_api::wire::ParameterGroupAddArgs {
                    internal_name: internal_name.clone(),
                    display_name: display_name.clone(),
                    client_id: client_id.clone(),
                }),
            GuiParameterGroupAction::Rename {
                internal_name,
                display_name,
                ..
            } => client.parameters().set_group_display_name(
                &oblikovati_api::wire::ParameterGroupDisplayNameArgs {
                    internal_name: internal_name.clone(),
                    display_name: display_name.clone(),
                },
            ),
            GuiParameterGroupAction::Delete {
                internal_name,
                delete_parameters,
                ..
            } => {
                client
                    .parameters()
                    .delete_group(&oblikovati_api::wire::ParameterGroupDeleteArgs {
                        internal_name: internal_name.clone(),
                        delete_parameters: *delete_parameters,
                    })
                    .map_err(|cause| cause.to_string())?;
                return Ok(true);
            }
            GuiParameterGroupAction::RemoveMember { parameter, .. } => {
                let parameter = member_name.ok_or_else(|| {
                    format!("GUI parameter group member {parameter}: expected a resolved name")
                })?;
                client.parameters().remove_group_member(
                    &oblikovati_api::wire::ParameterGroupMemberArgs {
                        internal_name: key.to_owned(),
                        parameter,
                    },
                )
            }
        }
        .map_err(|cause| cause.to_string())?;
        Ok(true)
    }
}

#[cfg(test)]
#[path = "owner_parameters_model_value_tests.rs"]
mod model_value_tests;
