// SPDX-License-Identifier: GPL-2.0-only
//! Facet and accent colors from the supplied design tokens.

pub(super) const PAINTS: &[(&str, [&str; 3])] = &[
    ("accent-brand", ["#8172FA", "#5B46E8", "#A599FF"]),
    (
        "accent-brand-subtle",
        [
            "rgba(129, 114, 250, 0.16)",
            "rgba(91, 70, 232, 0.10)",
            "rgba(165, 153, 255, 0.24)",
        ],
    ),
    ("accent-selection-bright", ["#2DD4BF", "#14B8A6", "#5EEAD4"]),
    (
        "accent-selection-subtle",
        [
            "rgba(20, 184, 166, 0.20)",
            "rgba(13, 148, 136, 0.14)",
            "rgba(45, 212, 191, 0.28)",
        ],
    ),
    ("accent-work", ["#F59E0B", "#D97706", "#FBBF24"]),
    ("bg-surface-base", ["#121317", "#F6F7F9", "#000000"]),
    ("icon-body-front", ["#9EA6B8", "#C5CCD9", "#C8CEDC"]),
    ("icon-body-side", ["#677084", "#8C95A8", "#8E96A8"]),
    ("icon-body-stroke", ["#181B22", "#282E3A", "#000000"]),
    ("icon-body-top", ["#D2D7E2", "#EEF1F6", "#FFFFFF"]),
    ("icon-cutout", ["#121317", "#F6F7F9", "#000000"]),
    ("icon-line-active", ["#8172FA", "#5B46E8", "#A599FF"]),
    ("icon-line-center", ["#38BDF8", "#0284C7", "#7DD3FC"]),
    ("icon-line-general", ["#D2D7E2", "#3B4252", "#FFFFFF"]),
    ("icon-mod-front", ["#8172FA", "#6754E8", "#A599FF"]),
    ("icon-mod-side", ["#5948D4", "#4936C4", "#7B6BF5"]),
    ("icon-mod-stroke", ["#1D1840", "#241A63", "#000000"]),
    ("icon-mod-top", ["#B4AAFF", "#9E91FF", "#CFC8FF"]),
    ("icon-sel-front", ["#14B8A6", "#0D9488", "#2DD4BF"]),
    ("icon-sel-side", ["#0D877A", "#0F766E", "#14B8A6"]),
    ("icon-sel-stroke", ["#0B2926", "#0A3330", "#000000"]),
    ("icon-sel-top", ["#74EFE0", "#5EEAD4", "#99F6E4"]),
    ("icon-surf-front", ["#F59E0B", "#D97706", "#FBBF24"]),
    ("icon-surf-stroke", ["#2E1D05", "#452406", "#000000"]),
    ("icon-surf-top", ["#FDE08A", "#FDE68A", "#FEF08A"]),
    ("icon-void-front", ["#5B6375", "#7E8799", "#6B7282"]),
    ("icon-void-side", ["#3A404F", "#555E6F", "#444956"]),
    ("icon-void-top", ["#858DA0", "#A6AFBF", "#9EA6B8"]),
    ("status-success", ["#22C55E", "#16A34A", "#4ADE80"]),
];
