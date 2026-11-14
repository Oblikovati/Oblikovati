// SPDX-License-Identifier: GPL-2.0-only
use oblikovati_gui::{GuiSelectionFilter, GuiSelectionFilterEntry, GuiSelectionKind};
use oblikovati_session::selection::{AmbientSelectionFilter, SelectionKind};

pub(super) fn gui_selection_filter(
    filter: &AmbientSelectionFilter,
) -> Result<GuiSelectionFilter, String> {
    let entries = filter
        .order()
        .iter()
        .map(|kind| {
            let visible = gui_kind(*kind).ok_or_else(|| {
                format!("selection category {kind:?}: expected a filterable desktop category")
            })?;
            Ok(GuiSelectionFilterEntry {
                kind: visible,
                enabled: filter.enabled(*kind),
            })
        })
        .collect::<Result<Vec<_>, String>>()?;
    Ok(GuiSelectionFilter { entries })
}
fn gui_kind(kind: SelectionKind) -> Option<GuiSelectionKind> {
    Some(match kind {
        SelectionKind::WorkPoint => GuiSelectionKind::WorkPoint,
        SelectionKind::WorkAxis => GuiSelectionKind::WorkAxis,
        SelectionKind::PointCloudPoint => GuiSelectionKind::PointCloudPoint,
        SelectionKind::Vertex => GuiSelectionKind::Vertex,
        SelectionKind::Edge => GuiSelectionKind::Edge,
        SelectionKind::Occurrence => GuiSelectionKind::Occurrence,
        SelectionKind::Body => GuiSelectionKind::Body,
        SelectionKind::Face => GuiSelectionKind::Face,
        SelectionKind::MeshFace => GuiSelectionKind::MeshFace,
        SelectionKind::WorkPlane => GuiSelectionKind::WorkPlane,
        SelectionKind::Profile => GuiSelectionKind::Profile,
        SelectionKind::SketchEntity => GuiSelectionKind::SketchEntity,
        _ => return None,
    })
}
pub(super) fn session_kind(kind: GuiSelectionKind) -> SelectionKind {
    match kind {
        GuiSelectionKind::WorkPoint => SelectionKind::WorkPoint,
        GuiSelectionKind::WorkAxis => SelectionKind::WorkAxis,
        GuiSelectionKind::PointCloudPoint => SelectionKind::PointCloudPoint,
        GuiSelectionKind::Vertex => SelectionKind::Vertex,
        GuiSelectionKind::Edge => SelectionKind::Edge,
        GuiSelectionKind::Occurrence => SelectionKind::Occurrence,
        GuiSelectionKind::Body => SelectionKind::Body,
        GuiSelectionKind::Face => SelectionKind::Face,
        GuiSelectionKind::MeshFace => SelectionKind::MeshFace,
        GuiSelectionKind::WorkPlane => SelectionKind::WorkPlane,
        GuiSelectionKind::Profile => SelectionKind::Profile,
        GuiSelectionKind::SketchEntity => SelectionKind::SketchEntity,
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn every_ordered_category_round_trips_and_retains_its_state() {
        let mut filter = AmbientSelectionFilter::default();
        filter.set_enabled(SelectionKind::Face, false);
        filter.move_kind(7, 0);
        let view = gui_selection_filter(&filter).unwrap();
        assert_eq!(view.entries.len(), 12);
        assert_eq!(view.entries[0].kind, GuiSelectionKind::Face);
        assert!(!view.entries[0].enabled);
        for (entry, source) in view.entries.iter().zip(filter.order()) {
            assert_eq!(session_kind(entry.kind), *source);
            assert_eq!(entry.enabled, filter.enabled(*source));
        }
    }
}
