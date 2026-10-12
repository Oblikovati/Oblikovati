// SPDX-License-Identifier: GPL-2.0-only
use std::ffi::OsString;
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct HeadOptions {
    pub frames: Option<u64>,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum HeadCommand {
    Help,
    Version,
    Window(HeadOptions),
}
/// Validate before opening a native window; example: `parse_arguments(&["--frames".into(), "3".into()])?`.
pub fn parse_arguments(arguments: &[OsString]) -> Result<HeadCommand, String> {
    match arguments {
        [] => Ok(HeadCommand::Window(HeadOptions::default())),
        [flag] if ["--help", "-h"].contains(&flag.to_str().unwrap_or("")) => Ok(HeadCommand::Help),
        [flag] if ["--version", "version"].contains(&flag.to_str().unwrap_or("")) => {
            Ok(HeadCommand::Version)
        }
        [flag, count] if ["--frames", "-frames"].contains(&flag.to_str().unwrap_or("")) => {
            parse_frames(count)
        }
        _ => Err(format!(
            "head arguments {arguments:?}: expected --frames N, --help, --version, or no arguments"
        )),
    }
}
fn parse_frames(count: &OsString) -> Result<HeadCommand, String> {
    let value = count
        .to_str()
        .filter(|value| !value.is_empty() && value.bytes().all(|byte| byte.is_ascii_digit()))
        .and_then(|value| value.parse::<u64>().ok())
        .ok_or_else(|| {
            format!("head frame count {count:?}: expected an unsigned 64-bit decimal integer")
        })?;
    Ok(HeadCommand::Window(HeadOptions {
        frames: (value != 0).then_some(value),
    }))
}
