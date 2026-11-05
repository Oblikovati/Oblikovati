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
    let (kind, kind_name) = document_kind(kind)?;
    if seed && kind == DOCUMENT_PART {
        return Err(
            "new: --seed requires Martensite extrusion; seeded part creation is not yet ported"
                .into(),
        );
    }
    let path = paths::with_extension(requested, paths::kind_extension(kind)?);
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
fn document_kind(name: &OsString) -> Result<(DocumentType, &'static str), String> {
    match name
        .to_str()
        .map(|value| value.trim().to_lowercase())
        .as_deref()
    {
        Some("part") => Ok((DOCUMENT_PART, "part")),
        Some("assembly") => Ok((DOCUMENT_ASSEMBLY, "assembly")),
        Some("drawing") => Ok((DOCUMENT_DRAWING, "drawing")),
        Some("presentation") => Ok((DOCUMENT_PRESENTATION, "presentation")),
        _ => Err(format!(
            "new: type {name:?}: expected part, assembly, drawing, or presentation"
        )),
    }
}
