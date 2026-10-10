// SPDX-License-Identifier: GPL-2.0-only

use super::*;
use oblikovati_persistence::load::read_package;
use std::path::Path;

pub(super) fn inspect(
    arguments: &[OsString],
    reader: &mut dyn PackageFileReader,
    output: &mut dyn Write,
) -> Result<(), String> {
    let (path, json) = info_arguments(arguments)?;
    let package = read_package(path, reader).map_err(|error| format!("info: {error}"))?;
    let manifest = package.manifest();
    let kind = manifest.type_name();
    let text = if json {
        serde_json::json!({"schemaVersion": manifest.schema_version, "type": kind, "name": manifest.display_name}).to_string()
    } else {
        format!(
            "schema version: {}\ntype: {kind}\nname: {}",
            manifest.schema_version, manifest.display_name
        )
    };
    write_line(output, &text)
}

fn info_arguments(arguments: &[OsString]) -> Result<(&Path, bool), String> {
    let mut path = None;
    let mut json = false;
    for argument in arguments {
        match argument.to_str() {
            Some("--json" | "-json" | "--json=true" | "-json=true") => json = true,
            Some("--json=false" | "-json=false") => json = false,
            Some(value) if value.starts_with('-') => {
                return Err(format!(
                    "info: flag {value:?}: expected --json or one package path"
                ));
            }
            _ if path.is_none() => path = Some(Path::new(argument)),
            _ => {
                return Err(format!(
                    "info: expected one package path, got {arguments:?}"
                ));
            }
        }
    }
    path.map(|path| (path, json))
        .ok_or_else(|| format!("info: expected <path>, got {arguments:?}"))
}
