// SPDX-License-Identifier: GPL-2.0-only
mod go;
mod source;
use crate::{
    process::{BuildResult, Runner},
    release_config::{self, ReleasedDependency},
    tree::{read_text, source_files, write_text},
};
use oblikovati_script::help::{MethodHelp, MethodHelpSnapshot, MethodParameter};
use std::{collections::BTreeMap, path::Path};
const CATALOG: &str = "crates/script/resources/method-help.json";
/// Regenerate pinned API metadata; example: `make script-docs`.
pub(crate) fn write(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    write_text(&root.join(CATALOG), &generated(root, runner)?)
}
/// Refuse stale generated metadata; example: `make gate`.
pub(crate) fn check(root: &Path, runner: &dyn Runner) -> BuildResult<()> {
    if read_text(&root.join(CATALOG))? != generated(root, runner)? {
        return Err(format!(
            "{CATALOG}: expected current pinned API help; run make script-docs"
        ));
    }
    Ok(())
}
fn generated(root: &Path, runner: &dyn Runner) -> BuildResult<String> {
    let pin = release_config::load(root)?.api;
    let api = source::api_source(root, runner, &pin)?;
    let snapshot = collect(&api, &pin)?;
    serde_json::to_string(&snapshot)
        .map(|json| json + "\n")
        .map_err(|error| format!("method help: expected a JSON contract snapshot: {error}"))
}
fn collect(api: &Path, pin: &ReleasedDependency) -> BuildResult<MethodHelpSnapshot> {
    let wire = wire_help(api)?;
    let methods = collect_clients(api, &wire)?;
    if methods.is_empty() {
        return Err(format!(
            "{}: expected scriptable API methods",
            api.display()
        ));
    }
    Ok(MethodHelpSnapshot {
        api_version: pin.version.clone(),
        api_revision: pin.revision.clone(),
        methods: methods.into_values().collect(),
    })
}
fn collect_clients(api: &Path, wire: &go::WireHelp) -> BuildResult<BTreeMap<String, MethodHelp>> {
    let mut methods = BTreeMap::new();
    for file in go_files(&api.join("client"))? {
        let client = go::parse_client(&read_text(&file)?)
            .map_err(|error| format!("{}: {error}", file.display()))?;
        for help in client {
            insert_method(&mut methods, wire, help)?;
        }
    }
    Ok(methods)
}

fn wire_help(api: &Path) -> BuildResult<go::WireHelp> {
    let mut wire = go::WireHelp::default();
    for file in go_files(&api.join("wire"))? {
        go::parse_wire(&read_text(&file)?, &mut wire)
            .map_err(|error| format!("{}: {error}", file.display()))?;
    }
    Ok(wire)
}
fn go_files(directory: &Path) -> BuildResult<Vec<std::path::PathBuf>> {
    Ok(source_files(directory)?
        .into_iter()
        .filter(|path| {
            path.extension().is_some_and(|ext| ext == "go")
                && !path
                    .file_name()
                    .unwrap_or_default()
                    .to_string_lossy()
                    .ends_with("_test.go")
        })
        .collect())
}
fn insert_method(
    methods: &mut BTreeMap<String, MethodHelp>,
    wire: &go::WireHelp,
    client: go::ClientHelp,
) -> BuildResult<()> {
    let name = wire.constants.get(&client.constant).ok_or_else(|| {
        format!(
            "Go method {}: expected a defined wire constant",
            client.constant
        )
    })?;
    let help = MethodHelp {
        wire: name.clone(),
        summary: client.summary,
        parameters: request_parameters(wire, client.request)?,
    };
    if let Some(previous) = methods.get_mut(name) {
        return merge_help(previous, help);
    }
    methods.insert(name.clone(), help);
    Ok(())
}
fn request_parameters(
    wire: &go::WireHelp,
    request: Option<String>,
) -> BuildResult<Vec<MethodParameter>> {
    let Some(request) = request else {
        return Ok(Vec::new());
    };
    wire.requests
        .get(&request)
        .cloned()
        .ok_or_else(|| format!("Go request {request:?}: expected a canonical wire struct"))
}
fn merge_help(previous: &mut MethodHelp, help: MethodHelp) -> BuildResult<()> {
    if previous.summary.is_empty() {
        previous.summary = help.summary;
    }
    for parameter in help.parameters {
        merge_parameter(previous, parameter)?;
    }
    Ok(())
}
fn merge_parameter(previous: &mut MethodHelp, parameter: MethodParameter) -> BuildResult<()> {
    if let Some(existing) = previous
        .parameters
        .iter()
        .find(|existing| existing.name == parameter.name)
    {
        if existing.type_name != parameter.type_name {
            return Err(format!(
                "Go method {} field {} type {:?}: expected {}",
                previous.wire, parameter.name, parameter.type_name, existing.type_name
            ));
        }
        return Ok(());
    }
    previous.parameters.push(parameter);
    Ok(())
}
#[cfg(test)]
#[path = "../tests/script_docs.rs"]
pub(crate) mod tests;
