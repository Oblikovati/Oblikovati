// SPDX-License-Identifier: GPL-2.0-only
use super::*;
pub(super) struct ScriptArguments {
    pub script: PathBuf,
    pub document: Option<PathBuf>,
    pub save: Option<PathBuf>,
}
pub(super) fn parse(args: &[OsString]) -> Result<ScriptArguments, String> {
    if args.first().and_then(|arg| arg.to_str()) != Some("run") {
        return Err(format!(
            "script operands {args:?}: expected run <file.lua> [--doc <path>] [--save <path>]"
        ));
    }
    let script = args
        .get(1)
        .filter(|path| !path.to_string_lossy().starts_with('-'))
        .ok_or_else(|| format!("script operands {args:?}: expected source file first"))?;
    let mut parsed = ScriptArguments {
        script: script.into(),
        document: None,
        save: None,
    };
    parse_options(&mut parsed, &args[2..])?;
    Ok(parsed)
}
fn parse_options(parsed: &mut ScriptArguments, mut args: &[OsString]) -> Result<(), String> {
    while let Some((flag, remaining)) = args.split_first() {
        if flag == "--" && remaining.is_empty() {
            return Ok(());
        }
        let (name, value, inline) = split_option(flag, remaining)?;
        assign_option(parsed, name, value)?;
        args = if inline { remaining } else { &remaining[1..] };
    }
    Ok(())
}
fn assign_option(
    parsed: &mut ScriptArguments,
    name: Option<&str>,
    value: OsString,
) -> Result<(), String> {
    let path = if value.is_empty() {
        None
    } else {
        Some(PathBuf::from(value))
    };
    match name {
        Some("--doc" | "-doc") => parsed.document = path,
        Some("--save" | "-save") => parsed.save = path,
        _ => {
            return Err(format!(
                "script flag {name:?}: expected --doc or --save with a path"
            ));
        }
    }
    Ok(())
}

fn split_option<'a>(
    flag: &'a OsString,
    remaining: &[OsString],
) -> Result<(Option<&'a str>, OsString, bool), String> {
    let text = flag
        .to_str()
        .ok_or_else(|| format!("script flag {flag:?}: expected a UTF-8 option name"))?;
    if let Some((name, value)) = text.split_once('=') {
        return Ok((Some(name), value.into(), true));
    }
    let value = remaining
        .first()
        .cloned()
        .ok_or_else(|| format!("script flag {flag:?}: expected a path value"))?;
    Ok((Some(text), value, false))
}
