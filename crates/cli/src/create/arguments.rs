// SPDX-License-Identifier: GPL-2.0-only
use super::*;

pub(super) fn parse(args: &[OsString]) -> Result<CreationRequest, String> {
    let (operands, seed) = partition(args)?;
    let [kind, path] = operands.as_slice() else {
        return Err(format!("new: expected <type> <path>, got {operands:?}"));
    };
    build_request(kind, path, seed)
}
fn build_request(
    kind: &OsString,
    requested: &OsString,
    seed: bool,
) -> Result<CreationRequest, String> {
    let (kind, kind_name, extension) = document_kind(kind)?;
    if seed && kind == DOCUMENT_PART {
        return Err(
            "new: --seed requires Martensite extrusion; seeded part creation is not yet ported"
                .into(),
        );
    }
    let path = with_extension(requested, extension);
    let name = oblikovati_session::files::derive_document_title(&path);
    Ok(CreationRequest {
        kind,
        kind_name,
        path,
        name,
        seed,
    })
}
fn partition(args: &[OsString]) -> Result<(Vec<&OsString>, bool), String> {
    let mut operands = Vec::new();
    let mut seed = false;
    for arg in args {
        match arg.to_str() {
            Some("--seed" | "-seed" | "--seed=true" | "-seed=true") => seed = true,
            Some("--seed=false" | "-seed=false") => seed = false,
            _ if arg.as_encoded_bytes().starts_with(b"-") => {
                return Err(format!(
                    "new: flag {arg:?}: expected --seed or two operands"
                ));
            }
            _ => operands.push(arg),
        }
    }
    Ok((operands, seed))
}
fn document_kind(name: &OsString) -> Result<(DocumentType, &'static str, &'static str), String> {
    match name
        .to_str()
        .map(|value| value.trim().to_lowercase())
        .as_deref()
    {
        Some("part") => Ok((DOCUMENT_PART, "part", PART_FILE_EXTENSION)),
        Some("assembly") => Ok((DOCUMENT_ASSEMBLY, "assembly", ASSEMBLY_FILE_EXTENSION)),
        Some("drawing") => Ok((DOCUMENT_DRAWING, "drawing", DRAWING_FILE_EXTENSION)),
        Some("presentation") => Ok((
            DOCUMENT_PRESENTATION,
            "presentation",
            PRESENTATION_FILE_EXTENSION,
        )),
        _ => Err(format!(
            "new: type {name:?}: expected part, assembly, drawing, or presentation"
        )),
    }
}
fn with_extension(path: &OsString, extension: &str) -> PathBuf {
    let current = PathBuf::from(path);
    if matching_extension(&current, extension) {
        return current;
    }
    let mut stamped = path.clone();
    stamped.push(extension);
    PathBuf::from(stamped)
}
fn matching_extension(path: &std::path::Path, extension: &str) -> bool {
    // Go filepath.Ext includes a leading dot; Path::extension omits dotfile extensions.
    let name = path.file_name().unwrap_or_default().as_encoded_bytes();
    let Some(dot) = name.iter().rposition(|byte| *byte == b'.') else {
        return false;
    };
    name[dot..].eq_ignore_ascii_case(extension.as_bytes())
}
