// SPDX-License-Identifier: GPL-2.0-only
use super::*;
use oblikovati_gui::{GuiSelectionFilter, GuiSelectionFilterEntry, GuiSelectionKind};
use oblikovati_session::selection::{AmbientSelectionFilter, SelectionKind};

impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    /// Attach the loaded application preference owner; example: `owner.with_selection_preferences(store)`.
    pub fn with_selection_preferences(
        mut self,
        store: Box<dyn crate::SelectionPreferenceAccess>,
    ) -> Self {
        self.ambient_selection = store.selection_filter().clone();
        self.selection_mode = store.selection_mode();
        self.selection_store = Some(store);
        self
    }
    pub(super) fn publish_selection_filter(
        &mut self,
        filter: AmbientSelectionFilter,
        mode: oblikovati_gui::GuiSelectionMode,
    ) -> Result<bool, String> {
        if self.ambient_selection == filter && self.selection_mode == mode {
            return Ok(false);
        }
        if let Some(store) = self.selection_store.as_mut() {
            store.set_selection_filter(&filter, mode)?;
        }
        self.ambient_selection = filter;
        self.selection_mode = mode;
        Ok(true)
    }
}
impl<
    O: TransactionObserver,
    W: WorkspaceObserver,
    F: DocumentFiles,
    G: GuidSource,
    R: DocumentObserverFactory<Observer = O> + HeadInvalidation,
    S: DispatchWakeup,
> DesktopOwner<O, W, F, G, R, S>
{
    pub(super) fn set_selection_mode(
        &mut self,
        mode: oblikovati_gui::GuiSelectionMode,
    ) -> Result<bool, String> {
        use oblikovati_gui::GuiSelectionMode;
        use oblikovati_session::selection::SelectionPriority;
        let state = self.gui_command_snapshot()?;
        if !oblikovati_gui::GuiShellAction::SetSelectionMode(mode).enabled(&state) {
            return Err(format!(
                "selection mode {mode:?}: expected an idle part or assembly"
            ));
        }
        let priority = match mode {
            GuiSelectionMode::All => SelectionPriority::General,
            GuiSelectionMode::Faces => SelectionPriority::Face,
            GuiSelectionMode::Components => SelectionPriority::Part,
            GuiSelectionMode::Custom => {
                return Err("selection mode Custom: expected a concrete preset".into());
            }
        };
        let mut filter = self.ambient_selection.clone();
        filter.apply_priority(priority);
        self.publish_selection_filter(filter, mode)
    }
    pub(super) fn set_selection_enabled(
        &mut self,
        kind: oblikovati_gui::GuiSelectionKind,
        enabled: bool,
    ) -> Result<bool, String> {
        let state = self.gui_command_snapshot()?;
        if !(oblikovati_gui::GuiShellAction::SetSelectionEnabled { kind, enabled }).enabled(&state)
        {
            return Err(format!(
                "selection category {kind:?}: expected an idle part or assembly"
            ));
        }
        let kind = session_kind(kind);
        if self.ambient_selection.enabled(kind) == enabled {
            return Ok(false);
        }
        let mut filter = self.ambient_selection.clone();
        filter.set_enabled(kind, enabled);
        self.publish_selection_filter(filter, oblikovati_gui::GuiSelectionMode::Custom)
    }
    pub(super) fn move_selection_kind(&mut self, from: usize, to: usize) -> Result<bool, String> {
        let state = self.gui_command_snapshot()?;
        if !(oblikovati_gui::GuiShellAction::MoveSelectionKind { from, to }).enabled(&state) {
            return Err(format!(
                "selection priority {from}->{to}: expected an idle part or assembly"
            ));
        }
        let count = self.ambient_selection.order().len();
        if from >= count || to >= count {
            return Err(format!(
                "selection priority {from}->{to}: expected indices below {count}"
            ));
        }
        if from == to {
            return Ok(false);
        }
        let mut filter = self.ambient_selection.clone();
        filter.move_kind(from, to);
        self.publish_selection_filter(filter, self.selection_mode)
    }
    pub(super) fn reset_selection_filter(&mut self) -> Result<bool, String> {
        let state = self.gui_command_snapshot()?;
        if !oblikovati_gui::GuiShellAction::ResetSelectionFilter.enabled(&state) {
            return Err("selection filter reset: expected an idle part or assembly".into());
        }
        self.publish_selection_filter(Default::default(), oblikovati_gui::GuiSelectionMode::All)
    }
}
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
