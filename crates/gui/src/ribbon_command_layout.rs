// SPDX-License-Identifier: GPL-2.0-only
//! Placement policy for a sequence of equal ribbon command cells.

#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub(crate) enum RibbonAppearance {
    #[default]
    Normal,
    TextOff,
    Small,
    Compact,
    Large,
}
impl RibbonAppearance {
    pub(crate) const ALL: [Self; 5] = [
        Self::Normal,
        Self::TextOff,
        Self::Small,
        Self::Compact,
        Self::Large,
    ];
    pub(crate) fn label(self) -> &'static str {
        match self {
            Self::Normal => "Normal",
            Self::TextOff => "Text Off",
            Self::Small => "Small",
            Self::Compact => "Compact",
            Self::Large => "Large",
        }
    }
    pub(crate) fn id(self) -> &'static str {
        match self {
            Self::Normal => "normal",
            Self::TextOff => "text-off",
            Self::Small => "small",
            Self::Compact => "compact",
            Self::Large => "large",
        }
    }
    pub(crate) fn from_id(value: &str) -> Result<Self, String> {
        Self::ALL
            .into_iter()
            .find(|mode| mode.id() == value)
            .ok_or_else(|| format!("ribbon appearance {value:?}: expected a known appearance"))
    }
    pub(crate) fn single_cell(self) -> [f32; 2] {
        if self == Self::Normal {
            return [108., 58.];
        }
        self.cell(0).1
    }
    pub(crate) fn flyout_cell(self, index: usize) -> ([f32; 2], [f32; 2]) {
        let (mut position, size) = self.cell(index);
        let rows = if self == Self::Compact { 1 } else { 2 };
        position[0] += (index / rows) as f32 * 20.;
        (position, size)
    }
    pub(crate) fn command_height(self) -> f32 {
        match self {
            Self::Normal => 60.,
            Self::TextOff => 92.,
            Self::Small => 44.,
            Self::Compact => 20.,
            Self::Large => 92.,
        }
    }
    pub(crate) fn tab_command_height(self, tab: crate::ribbon::RibbonTab) -> f32 {
        if self == Self::Normal && tab == crate::ribbon::RibbonTab::Sketch {
            // The Sketch Create grid has a fourth row for Line and Project Geometry.
            88.
        } else {
            self.command_height()
        }
    }
    pub(crate) fn cell(self, index: usize) -> ([f32; 2], [f32; 2]) {
        let (rows, size, stride) = match self {
            Self::Normal => (2, [108., 26.], [116., 30.]),
            Self::TextOff => (2, [44., 44.], [48., 48.]),
            Self::Small => (2, [20., 20.], [24., 24.]),
            Self::Compact => (1, [20., 20.], [24., 24.]),
            Self::Large => (2, [44., 44.], [48., 48.]),
        };
        let position = [
            (index / rows) as f32 * stride[0],
            (index % rows) as f32 * stride[1],
        ];
        (position, size)
    }
}

#[cfg(test)]
mod tests {
    use super::RibbonAppearance;

    #[test]
    fn menu_labels_keep_the_documented_order() {
        assert_eq!(
            RibbonAppearance::ALL.map(RibbonAppearance::label),
            ["Normal", "Text Off", "Small", "Compact", "Large"]
        );
    }

    #[test]
    fn normal_cells_preserve_authored_two_row_layout() {
        assert_eq!(RibbonAppearance::Normal.cell(0), ([0., 0.], [108., 26.]));
        assert_eq!(RibbonAppearance::Normal.cell(1), ([0., 30.], [108., 26.]));
        assert_eq!(RibbonAppearance::Normal.cell(2), ([116., 0.], [108., 26.]));
    }

    #[test]
    fn appearance_ids_round_trip_and_reject_unknown_values() {
        for appearance in RibbonAppearance::ALL {
            assert_eq!(
                RibbonAppearance::from_id(appearance.id()).unwrap(),
                appearance
            );
        }
        assert!(RibbonAppearance::from_id("unknown").is_err());
    }

    #[test]
    fn single_commands_keep_the_normal_height() {
        assert_eq!(RibbonAppearance::Normal.single_cell(), [108., 58.]);
        for appearance in RibbonAppearance::ALL.into_iter().skip(1) {
            assert_eq!(appearance.single_cell(), appearance.cell(0).1);
        }
    }

    #[test]
    fn flyout_cells_reserve_space_for_choice_arrows() {
        for appearance in RibbonAppearance::ALL {
            let rows = if appearance == RibbonAppearance::Compact {
                1
            } else {
                2
            };
            let (first, size) = appearance.flyout_cell(0);
            let (next, _) = appearance.flyout_cell(rows);
            assert!(next[0] >= first[0] + size[0] + 16.);
        }
    }

    #[test]
    fn command_height_contains_every_cell() {
        for appearance in RibbonAppearance::ALL {
            for index in 0..8 {
                let (position, size) = appearance.cell(index);
                assert!(position[1] + size[1] <= appearance.command_height());
            }
        }
    }

    #[test]
    fn alternate_cells_do_not_overlap_and_compact_has_one_row() {
        for mode in RibbonAppearance::ALL {
            for left in 0..12 {
                let (a, size) = mode.cell(left);
                for right in left + 1..12 {
                    let (b, _) = mode.cell(right);
                    assert!(
                        a[0] + size[0] <= b[0] || a[1] + size[1] <= b[1] || b[1] + size[1] <= a[1],
                        "{mode:?}: {left} overlaps {right}"
                    );
                }
                if mode == RibbonAppearance::Compact {
                    assert_eq!(a[1], 0.);
                }
            }
        }
    }
}
