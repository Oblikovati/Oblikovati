// SPDX-License-Identifier: GPL-2.0-only
// Ported from Go release/{version,scope}.go at aa8297d1 (#3560).

use crate::process::{BuildResult, Runner};
use chrono::{DateTime, Utc};
use regex::Regex;
use std::path::Path;

#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
enum Scope {
    None,
    Patch,
    Feature,
    Breaking,
}

fn classify(message: &str) -> Scope {
    if message
        .lines()
        .any(|line| line.starts_with("BREAKING CHANGE:") || line.starts_with("BREAKING-CHANGE:"))
    {
        return Scope::Breaking;
    }
    let pattern = Regex::new(r"^([A-Za-z]+)(?:\([^)]*\))?(!)?:").expect("constant commit pattern");
    let Some(captures) = pattern.captures(message.trim()) else {
        return Scope::Patch;
    };
    if captures.get(2).is_some() {
        return Scope::Breaking;
    }
    match captures[1].to_ascii_lowercase().as_str() {
        "feat" => Scope::Feature,
        "docs" | "chore" | "ci" | "test" | "style" | "build" | "refactor" => Scope::None,
        _ => Scope::Patch,
    }
}

pub(crate) fn api_field(api: &str) -> BuildResult<String> {
    let clean = api.trim().trim_start_matches('v');
    let components = clean.split('.').collect::<Vec<_>>();
    if components.len() != 3 {
        return Err(format!("API version {api:?}: expected MAJOR.MINOR.PATCH"));
    }
    components
        .iter()
        .map(|part| {
            let value = number(part)?;
            Ok(format!("{value:02}"))
        })
        .collect()
}

fn number(text: &str) -> BuildResult<u64> {
    if text.is_empty() || !text.bytes().all(|byte| byte.is_ascii_digit()) {
        return Err(format!(
            "version component {text:?}: expected an unsigned decimal integer"
        ));
    }
    text.parse()
        .map_err(|error| format!("version component {text:?}: expected u64: {error}"))
}

fn tag_fields(tag: &str) -> Option<[u64; 4]> {
    // Go's v* tags belong to unrelated history. Only Rust stable tags seed numbering.
    let fields = tag
        .strip_prefix("rust-v")?
        .split('.')
        .map(number)
        .collect::<BuildResult<Vec<_>>>()
        .ok()?;
    fields.try_into().ok()
}

fn base_tag(tags: &str, major: u64, api: &str) -> Option<(String, u64, u64)> {
    let prefix = format!("rust-v{major}.{api}.");
    let valid = tags
        .lines()
        .filter_map(|tag| tag_fields(tag).map(|fields| (tag, fields)));
    let same = valid
        .clone()
        .filter(|(tag, _)| tag.starts_with(&prefix))
        .max_by_key(|(_, fields)| *fields);
    if let Some((tag, fields)) = same {
        return Some((tag.into(), fields[2], fields[3]));
    }
    valid
        .max_by_key(|(_, fields)| *fields)
        .map(|(tag, _)| (tag.into(), 0, 0))
}

fn bump(minor: u64, patch: u64, messages: &str) -> BuildResult<(u64, u64)> {
    let scope = messages
        .split('\0')
        .filter(|message| !message.trim().is_empty())
        .map(classify)
        .max()
        .unwrap_or(Scope::None);
    match scope {
        Scope::None => Some((minor, patch)),
        Scope::Patch => patch.checked_add(1).map(|next| (minor, next)),
        _ => minor.checked_add(1).map(|next| (next, 0)),
    }
    .ok_or_else(|| format!("version {minor}.{patch}: expected a bump within u64"))
}

pub(crate) fn compute(
    root: &Path,
    runner: &dyn Runner,
    major: u64,
    api: &str,
    channel: &str,
    now: DateTime<Utc>,
) -> BuildResult<String> {
    if !["stable", "nightly"].contains(&channel) {
        return Err(format!("channel {channel:?}: expected stable or nightly"));
    }
    let api = api_field(api)?;
    let tags = runner.output(
        root,
        "git",
        &["tag", "--merged", "HEAD", "--list", "rust-v*"],
    )?;
    let (tag, minor, patch) = base_tag(&tags, major, &api).unwrap_or_default();
    let range = if tag.is_empty() {
        "HEAD".into()
    } else {
        format!("{tag}..HEAD")
    };
    let messages = runner.output(
        root,
        "git",
        &["log", "--no-merges", "--format=%B%x00", &range],
    )?;
    let (minor, patch) = bump(minor, patch, &messages)?;
    let version = format!("{major}.{api}.{minor}.{patch}");
    Ok(if channel == "nightly" {
        format!("{version}-nightly.{}", now.format("%y%m%d%H"))
    } else {
        version
    })
}

#[cfg(test)]
#[path = "../tests/version.rs"]
mod tests;
