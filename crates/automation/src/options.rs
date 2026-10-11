// SPDX-License-Identifier: GPL-2.0-only
use crate::{decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_model::application_options::{SketchApplicationOptions, SketchPreferenceAccess};
/// Only ported groups are advertised; other application groups require their own owners.
pub const SKETCH_OPTION_METHODS: [&str; 3] = [
    METHOD_OPTIONS_LIST_GROUPS,
    METHOD_OPTIONS_GET_GROUP,
    METHOD_OPTIONS_SET_GROUP,
];
pub(super) fn apply(
    preferences: &mut dyn SketchPreferenceAccess,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    match method {
        METHOD_OPTIONS_LIST_GROUPS => encode(&ListOptionGroupsResult {
            groups: Some(vec![OPTION_GROUP_SKETCH.into()]),
        }),
        METHOD_OPTIONS_GET_GROUP => get_group(preferences, decode(method, request)?),
        METHOD_OPTIONS_SET_GROUP => set_group(preferences, decode(method, request)?),
        _ => Err(Error(format!(
            "application option method {method:?}: expected {SKETCH_OPTION_METHODS:?}"
        ))),
    }
}
fn get_group(
    preferences: &dyn SketchPreferenceAccess,
    request: GetOptionGroupArgs,
) -> Result<Vec<u8>, Error> {
    require_sketch_group(&request.group)?;
    encode(&OptionGroupView {
        group: request.group,
        sketch: Some(Box::new(wire_view(preferences.sketch_options()))),
        ..Default::default()
    })
}
fn wire_view(options: &SketchApplicationOptions) -> SketchOptionsView {
    SketchOptionsView {
        grid_spacing_cm: options.grid_spacing_cm,
        grid_visible: options.grid_visible,
        grid_major_every: options.grid_major_every,
        snap_to_points: options.snap_to_points,
        snap_to_grid: options.snap_to_grid,
        auto_project_origin: options.auto_project_origin,
    }
}
fn set_group(
    preferences: &mut dyn SketchPreferenceAccess,
    request: OptionGroupView,
) -> Result<Vec<u8>, Error> {
    require_sketch_group(&request.group)?;
    let view = matching_payload(&request)?;
    let options = merge_view(preferences.sketch_options(), view);
    options.validate().map_err(Error)?;
    let reply = encode(&OKResult { ok: true })?;
    preferences.set_sketch_options(options).map_err(Error)?;
    Ok(reply)
}
fn matching_payload(request: &OptionGroupView) -> Result<&SketchOptionsView, Error> {
    if request.general.is_some()
        || request.display.is_some()
        || request.part.is_some()
        || request.save.is_some()
    {
        return Err(Error(format!(
            "application option group {:?} carries other groups: expected only its sketch payload",
            request.group
        )));
    }
    request.sketch.as_deref().ok_or_else(|| {
        Error(format!(
            "application option group {:?} has no sketch payload: expected exactly the named group",
            request.group
        ))
    })
}
fn merge_view(
    current: &SketchApplicationOptions,
    view: &SketchOptionsView,
) -> SketchApplicationOptions {
    // Go #2016: wire writes must retain the display/input flags omitted from the API view.
    SketchApplicationOptions {
        grid_spacing_cm: view.grid_spacing_cm,
        grid_visible: view.grid_visible,
        grid_major_every: view.grid_major_every,
        snap_to_points: view.snap_to_points,
        snap_to_grid: view.snap_to_grid,
        auto_project_origin: view.auto_project_origin,
        ..current.clone()
    }
}
fn require_sketch_group(group: &str) -> Result<(), Error> {
    if group != OPTION_GROUP_SKETCH {
        return Err(Error(format!(
            "application option group {group:?}: expected supported group {OPTION_GROUP_SKETCH:?}"
        )));
    }
    Ok(())
}
