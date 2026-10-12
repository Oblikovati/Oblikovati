// SPDX-License-Identifier: GPL-2.0-only
use std::{
    ffi::{OsStr, OsString},
    path::PathBuf,
};
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct HeadOptions {
    pub frames: Option<u64>,
}
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum HeadCommand {
    Help,
    Version,
    Window(HeadOptions),
    OpenDocuments {
        options: HeadOptions,
        documents: Vec<PathBuf>,
    },
}
#[derive(Default)]
struct LaunchArguments {
    options: HeadOptions,
    documents: Vec<PathBuf>,
    literal_paths: bool,
    frames_seen: bool,
}
/// Validate before opening a native window; example: `parse_arguments(&["--frames".into(), "3".into(), "part.opd".into()])?`.
pub fn parse_arguments(arguments: &[OsString]) -> Result<HeadCommand, String> {
    match arguments {
        [flag] if ["--help", "-h"].contains(&flag.to_str().unwrap_or("")) => {
            return Ok(HeadCommand::Help);
        }
        [flag] if ["--version", "version"].contains(&flag.to_str().unwrap_or("")) => {
            return Ok(HeadCommand::Version);
        }
        _ => {}
    }
    let mut launch = LaunchArguments::default();
    let mut remaining = arguments.iter();
    while let Some(argument) = remaining.next() {
        launch.consume(argument, &mut remaining)?;
    }
    Ok(launch.into_command())
}
impl LaunchArguments {
    fn consume(
        &mut self,
        argument: &OsStr,
        remaining: &mut std::slice::Iter<'_, OsString>,
    ) -> Result<(), String> {
        let text = checked_operand(argument)?;
        if self.literal_paths {
            self.documents.push(argument.into());
            return Ok(());
        }
        match text {
            "--" => self.literal_paths = true,
            "--frames" | "-frames" => self.set_frames(
                remaining
                    .next()
                    .ok_or_else(|| format!("head option {argument:?}: expected --frames N"))?,
            )?,
            _ => self.named_or_path(argument, text)?,
        }
        Ok(())
    }
    fn named_or_path(&mut self, argument: &OsStr, text: &str) -> Result<(), String> {
        if let Some(count) = text
            .strip_prefix("--frames=")
            .or_else(|| text.strip_prefix("-frames="))
        {
            return self.set_frames(OsStr::new(count));
        }
        if text.starts_with('-') {
            return Err(format!(
                "head option {argument:?}: expected --frames N or -- before a document path"
            ));
        }
        self.documents.push(argument.into());
        Ok(())
    }
    fn set_frames(&mut self, count: &OsStr) -> Result<(), String> {
        if self.frames_seen {
            return Err(format!(
                "head frame count {count:?}: expected only one --frames option"
            ));
        }
        self.options.frames = parse_frame_count(count)?;
        self.frames_seen = true;
        Ok(())
    }
    fn into_command(self) -> HeadCommand {
        if self.documents.is_empty() {
            return HeadCommand::Window(self.options);
        }
        HeadCommand::OpenDocuments {
            options: self.options,
            documents: self.documents,
        }
    }
}
fn checked_operand(argument: &OsStr) -> Result<&str, String> {
    argument
        .to_str()
        .filter(|value| !value.is_empty())
        .ok_or_else(|| {
            format!("head operand {argument:?}: expected nonempty UTF-8 text for the document API")
        })
}
fn parse_frame_count(count: &OsStr) -> Result<Option<u64>, String> {
    let value = count
        .to_str()
        .filter(|value| !value.is_empty() && value.bytes().all(|byte| byte.is_ascii_digit()))
        .and_then(|value| value.parse::<u64>().ok())
        .ok_or_else(|| {
            format!("head frame count {count:?}: expected an unsigned 64-bit decimal integer")
        })?;
    Ok((value != 0).then_some(value))
}
