// SPDX-License-Identifier: GPL-2.0-only
#[derive(Clone, Copy, Debug, Default, PartialEq, Eq)]
pub struct HeadWorkStatus {
    pub changed: bool,
    pub pending: bool,
    pub wake_after: Option<std::time::Duration>,
}
/// Coalesced model changes from the injected document observers.
pub trait HeadInvalidation {
    /// Consume pending invalidation; example: `factory.take_changed()` after direct UI commands.
    fn take_changed(&self) -> bool;
}
/// Owner-thread services borrow model state only while the window loop polls them.
pub trait HeadWork {
    /// Copy the current transient authoring graph; example: `work.sketch_preview()` before drawing.
    fn sketch_preview(&self) -> Option<oblikovati_viewport::ViewportSketchPreview> {
        None
    }

    /// Count bounded frames only after startup is complete; example: `work.startup_ready()` before smoke accounting.
    fn startup_ready(&self) -> bool {
        true
    }

    /// Complete the requesting document's path choice; example: `work.complete_save_path(id, Some(path))?`.
    fn complete_save_path(
        &mut self,
        document: u64,
        selected: Option<&std::path::Path>,
    ) -> Result<bool, String> {
        if selected.is_none() {
            return Ok(false);
        }
        Err(format!(
            "save path for document {document}, selection {selected:?}: expected a document-save owner"
        ))
    }

    /// Copy bounded command metadata on the owner; example: `work.gui_command_state()?` after polling.
    fn gui_command_state(&self) -> Result<oblikovati_gui::GuiCommandState, String> {
        Ok(Default::default())
    }

    /// Apply native commands on the document owner; example: `work.process_inputs(&events)?` before polling.
    fn process_inputs(&mut self, _: &[oblikovati_viewport::ViewportEvent]) -> Result<bool, String> {
        Ok(false)
    }
    /// Route owner commands with the current window projection; example: `work.process_window_inputs(&events, window)?`.
    fn process_window_inputs(
        &mut self,
        inputs: &[oblikovati_viewport::ViewportEvent],
        _: &dyn crate::HeadWindow,
    ) -> Result<bool, String> {
        self.process_inputs(inputs)
    }
    /// Consume the latest command result for display; example: `work.take_command_notice()` after Save.
    fn take_command_notice(&mut self) -> Option<crate::DesktopCommandNotice> {
        None
    }

    /// Consume a complete scene update after all owner services; example: `work.take_scene()?` before presentation.
    fn take_scene(&mut self) -> Result<Option<crate::HeadSceneUpdate>, String> {
        Ok(None)
    }

    /// Drain one bounded request batch; example: `work.poll(32)?` before presentation.
    fn poll(&mut self, budget: usize) -> Result<HeadWorkStatus, String>;
    /// Check close without discarding documents; example: `work.close_allowed()?` after a native close request.
    fn close_allowed(&mut self) -> Result<bool, String>;
    /// Stop admission before window retirement; example: `work.finish()?` after the owner loop ends.
    fn finish(&mut self) -> Result<(), String>;
}
pub(crate) struct IdleHeadWork;
impl HeadWork for IdleHeadWork {
    fn poll(&mut self, _: usize) -> Result<HeadWorkStatus, String> {
        Ok(HeadWorkStatus::default())
    }
    fn close_allowed(&mut self) -> Result<bool, String> {
        Ok(true)
    }
    fn finish(&mut self) -> Result<(), String> {
        Ok(())
    }
}
