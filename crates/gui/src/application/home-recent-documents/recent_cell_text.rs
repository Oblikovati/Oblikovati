// SPDX-License-Identifier: GPL-2.0-only
//! Fit cell labels without changing the recent-file names and locations.
use std::borrow::Cow;

const ELLIPSIS: &str = "...";

pub(super) fn fit_recent_cell(
    full: &str,
    width: f32,
    measure: impl Fn(&str) -> f32,
) -> Cow<'_, str> {
    if measure(full) <= width {
        return Cow::Borrowed(full);
    }
    let available = width - measure(ELLIPSIS);
    if available < 0. {
        return Cow::Borrowed("");
    }
    let prefix = fitting_recent_prefix(full, available, measure);
    Cow::Owned(format!("{prefix}{ELLIPSIS}"))
}

fn fitting_recent_prefix(full: &str, width: f32, measure: impl Fn(&str) -> f32) -> &str {
    let (mut lower, mut upper) = (0, full.len());
    // Byte probes preserve UTF-8 boundaries without allocating a character index for every row.
    while lower < upper {
        let probe = lower + (upper - lower).div_ceil(2);
        let prefix = &full[..full.floor_char_boundary(probe)];
        if measure(prefix) <= width {
            lower = probe;
        } else {
            upper = probe - 1;
        }
    }
    &full[..full.floor_char_boundary(lower)]
}

pub(super) fn visible_source_text<'a>(shown: &'a str, full: &str) -> &'a str {
    if shown == full {
        return shown;
    }
    shown.strip_suffix(ELLIPSIS).unwrap_or(shown)
}

pub(super) fn reveal_full_cell(ui: &imgui::Ui, full: &str, shown: &str) {
    if full != shown && ui.is_item_hovered() {
        ui.tooltip_text(full);
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn recent_label_uses_whole_characters_and_keeps_short_text() {
        let width = |text: &str| text.chars().count() as f32;
        let shortened = fit_recent_cell("abcø界🌍é", 6., width);
        assert_eq!(shortened, "abc...");
        let shortened = fit_recent_cell("ø界🌍éabcd", 6., width);
        assert_eq!(shortened, "ø界🌍...");
        assert_eq!(visible_source_text(&shortened, "ø界🌍éabcd"), "ø界🌍");
        let same = fit_recent_cell("A...", 4., width);
        assert!(matches!(same, Cow::Borrowed("A...")));
        assert_eq!(visible_source_text(&same, "A..."), "A...");
        assert_eq!(fit_recent_cell("Long", 2., width), "");
    }
}
