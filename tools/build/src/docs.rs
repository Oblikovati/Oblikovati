// SPDX-License-Identifier: GPL-2.0-only

use crate::{
    process::BuildResult,
    tree::{read_text, source_files},
};
use regex::Regex;
use std::{collections::BTreeMap, path::Path};

pub(crate) fn check(root: &Path) -> BuildResult<()> {
    for path in source_files(root)? {
        if path.extension().is_some_and(|ext| ext == "rs") {
            check_source(&path)?;
        }
        if path.extension().is_some_and(|ext| ext == "md") {
            check_markdown(&path)?;
        }
    }
    if !root.join("LICENSE").is_file() {
        return Err("LICENSE: required GPL-2.0 text is missing".into());
    }
    Ok(())
}

fn check_source(path: &Path) -> BuildResult<()> {
    let text = read_text(path)?;
    if !text
        .lines()
        .take(5)
        .any(|line| line.contains("SPDX-License-Identifier: GPL-2.0-only"))
    {
        return Err(format!(
            "{}: expected GPL-2.0-only SPDX header",
            path.display()
        ));
    }
    if text.lines().count() >= 500 {
        return Err(format!("{}: expected fewer than 500 lines", path.display()));
    }
    Ok(())
}

fn anchors(text: &str) -> Vec<String> {
    let mut counts = BTreeMap::<String, usize>::new();
    text.lines()
        .filter_map(|line| line.strip_prefix('#'))
        .map(|heading| {
            let heading = heading.trim_start_matches('#').trim().to_lowercase();
            let slug = heading
                .chars()
                .filter(|character| {
                    character.is_alphanumeric() || [' ', '-', '_'].contains(character)
                })
                .map(|character| if character == ' ' { '-' } else { character })
                .collect::<String>();
            let count = counts.entry(slug.clone()).or_default();
            let anchor = if *count == 0 {
                slug
            } else {
                format!("{slug}-{count}")
            };
            *count += 1;
            anchor
        })
        .collect()
}

fn check_markdown(path: &Path) -> BuildResult<()> {
    let text = read_text(path)?;
    let pattern = Regex::new(r"\[[^\]]*\]\(([^)\s]+)\)").expect("constant Markdown link pattern");
    for matched in pattern.captures_iter(&text) {
        let link = matched[1].trim_matches(['<', '>']);
        if link.contains("://") || link.starts_with("mailto:") {
            continue;
        }
        check_link(path, link)?;
    }
    if text.lines().count() >= 500 {
        return Err(format!("{}: expected fewer than 500 lines", path.display()));
    }
    Ok(())
}

fn check_link(source: &Path, link: &str) -> BuildResult<()> {
    let (filename, anchor) = link.split_once('#').unwrap_or((link, ""));
    let target = if filename.is_empty() {
        source.to_path_buf()
    } else {
        source.parent().unwrap_or(Path::new(".")).join(filename)
    };
    if !target.exists() {
        return Err(format!(
            "{} link {link:?}: target {} is missing",
            source.display(),
            target.display()
        ));
    }
    if !anchor.is_empty()
        && target.extension().is_some_and(|ext| ext == "md")
        && !anchors(&read_text(&target)?).contains(&anchor.into())
    {
        return Err(format!(
            "{} link {link:?}: expected an existing Markdown heading",
            source.display()
        ));
    }
    Ok(())
}

#[cfg(test)]
#[path = "../tests/docs.rs"]
mod tests;
