// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_api::{Error, wire::*};
use oblikovati_model::parameter::{ParameterCollection, ParameterGroup, dimension_display_name};

pub(super) fn settings(
    parameters: &ParameterCollection<u64>,
) -> Result<ParameterSettingsInfo, Error> {
    let settings = parameters.settings();
    let mode = dimension_display_name(settings.dimension_display_type).ok_or_else(|| {
        Error(format!(
            "dimension display {:?}: expected a supported API display mode",
            settings.dimension_display_type
        ))
    })?;
    Ok(ParameterSettingsInfo {
        linear_standard_tolerance: settings.linear_standard_tolerance.clone(),
        angular_standard_tolerance: settings.angular_standard_tolerance.clone(),
        use_standard_tolerances: settings.use_standard_tolerances,
        export_standard_tolerances: settings.export_standard_tolerances,
        linear_dimension_precision: settings.linear_dimension_precision,
        angular_dimension_precision: settings.angular_dimension_precision,
        dimension_display_type: mode.into(),
        display_parameter_as_expression: settings.display_parameter_as_expression,
    })
}

pub(super) fn groups(
    parameters: &ParameterCollection<u64>,
) -> Result<ListParameterGroupsResult, Error> {
    let groups = parameters
        .groups()
        .iter()
        .map(|group| group_info(parameters, group))
        .collect::<Result<_, _>>()?;
    Ok(ListParameterGroupsResult {
        groups: Some(groups),
    })
}

pub(super) fn group_info(
    parameters: &ParameterCollection<u64>,
    group: &ParameterGroup<u64>,
) -> Result<ParameterGroupInfo, Error> {
    let ids = parameters
        .group_members(&group.key.0)
        .map_err(|error| Error(error.to_string()))?;
    Ok(ParameterGroupInfo {
        internal_name: group.key.0.to_string(),
        display_name: group.display_name.clone(),
        client_id: group.client_id.clone(),
        members: Some(crate::parameters::names(parameters, &ids)?),
    })
}
