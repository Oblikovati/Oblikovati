// SPDX-License-Identifier: GPL-2.0-only
//! Keep filenames readable as the recent list resizes.
use super::super::recent_view::column_width;
use super::{ACTION_GAP, CHECK_COLUMN, KIND_COLUMN, SortField, Ui};

const MINIMUM_NAME_WIDTH: f32 = 160.;
type RecentColumnWidths = [(SortField, f32); SortField::MENU.len()];

#[derive(Clone, Copy)]
pub(super) struct RecentListLayout {
    widths: RecentColumnWidths,
    pub(super) spacing: f32,
    pub(super) frame_padding: f32,
}

impl RecentListLayout {
    pub(super) fn for_columns(ui: &Ui, columns: &[SortField]) -> Self {
        let style = ui.clone_style();
        let spacing = style.item_spacing[0];
        let frame_padding = 2. * style.frame_padding[0];
        let actions = ui.calc_text_size("...")[0] + frame_padding + spacing + ACTION_GAP;
        let available = ui.content_region_avail()[0] - CHECK_COLUMN - KIND_COLUMN - actions;
        let marks = ui.calc_text_size(" v")[0].max(ui.calc_text_size(" ^")[0]);
        let minimum = SortField::MENU.map(|field| {
            let width = ui.calc_text_size(field.label())[0] + marks + spacing + frame_padding;
            (field, width)
        });
        Self {
            widths: fit_recent_columns(minimum, columns, available),
            spacing,
            frame_padding,
        }
    }

    pub(super) fn width(self, column: SortField) -> f32 {
        self.widths
            .iter()
            .find(|(field, _)| *field == column)
            .unwrap_or_else(|| panic!("recent column {column:?}: expected a measured width"))
            .1
    }
}

fn fit_recent_columns(
    minima: RecentColumnWidths,
    shown: &[SortField],
    available: f32,
) -> RecentColumnWidths {
    let minima = visible_recent_headings(minima, shown, available);
    let (minimum, extra) = minima.iter().filter(|(_, width)| *width > 0.).fold(
        (0., 0.),
        |(minimum, extra), &(field, width)| {
            (
                minimum + width,
                extra + extra_recent_column_width(field, width),
            )
        },
    );
    // Reserve the heading widths, then give filenames first claim on the flexible space.
    let heading_scale = (available / minimum).clamp(0., 1.);
    let name_minimum = minima[0].1;
    let name_target = if extra > 0. {
        (available * 0.34).max(220.)
    } else {
        available
    };
    let name_extra = (name_target - name_minimum).clamp(0., (available - minimum).max(0.));
    let extra_scale = if extra > 0. {
        ((available - minimum - name_extra) / extra).clamp(0., 1.)
    } else {
        0.
    };
    minima.map(|(field, width)| {
        let growth = if width == 0. {
            0.
        } else if field == SortField::Name {
            name_extra
        } else {
            extra_scale * extra_recent_column_width(field, width)
        };
        (field, heading_scale * width + growth)
    })
}

fn visible_recent_headings(
    mut minima: RecentColumnWidths,
    shown: &[SortField],
    available: f32,
) -> RecentColumnWidths {
    for (field, width) in &mut minima {
        if *field != SortField::Name && !shown.contains(field) {
            *width = 0.;
        }
    }
    let mut needed = minima.iter().map(|(_, width)| width).sum::<f32>();
    needed += (MINIMUM_NAME_WIDTH - minima[0].1).max(0.);
    for field in [
        SortField::Size,
        SortField::Modified,
        SortField::Opened,
        SortField::Location,
    ] {
        if needed <= available {
            break;
        }
        if let Some((_, width)) = minima.iter_mut().find(|(column, _)| *column == field) {
            needed -= std::mem::take(width);
        }
    }
    minima
}

fn extra_recent_column_width(field: SortField, minimum: f32) -> f32 {
    if field == SortField::Name {
        return 0.;
    }
    (column_width(field) - minimum).max(0.)
}

#[cfg(test)]
#[test]
fn compact_recent_columns_keep_width_for_size_and_its_sort_mark() {
    let shown = &super::super::recent_view::ADJUSTABLE;
    let minimum =
        SortField::MENU.map(|field| (field, if field == SortField::Size { 62. } else { 108. }));
    let widths = fit_recent_columns(minimum, shown, 560.);
    let size = widths
        .into_iter()
        .find(|(field, _)| *field == SortField::Size)
        .unwrap()
        .1;
    assert!(size >= 62.);
    let compact = fit_recent_columns(minimum, shown, 320.);
    assert_eq!(compact[0].0, SortField::Name);
    assert!(compact[0].1 >= MINIMUM_NAME_WIDTH);
    assert!(
        compact
            .iter()
            .any(|&(field, width)| field == SortField::Size && width == 0.)
    );
    for width in [560., 320.] {
        let widths = fit_recent_columns(minimum, shown, width);
        let visible = widths
            .iter()
            .filter(|(field, _)| *field == SortField::Name || shown.contains(field));
        assert!((visible.map(|(_, width)| width).sum::<f32>() - width).abs() < 0.01);
    }
}
