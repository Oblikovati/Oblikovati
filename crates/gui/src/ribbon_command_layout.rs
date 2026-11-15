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
    pub(crate) fn cell(self, index: usize) -> ([f32; 2], [f32; 2]) {
        let (rows, size, stride) = match self {
            Self::Normal => (2, [108., 26.], [116., 30.]),
            Self::TextOff => (2, [26., 26.], [30., 30.]),
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
