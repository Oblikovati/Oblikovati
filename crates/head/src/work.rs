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
    /// Apply a live custom colour preview; example: `work.edit_theme_color_from_gui(name, token, value)?`.
    fn edit_theme_color_from_gui(
        &mut self,
        name: &str,
        token: &oblikovati_api::types::ThemeToken,
        _value: oblikovati_api::types::Rgba,
    ) -> Result<bool, String> {
        Err(format!(
            "theme {name:?}, token {token:?}: expected an editable application theme owner"
        ))
    }
    /// Copy the active theme; example: `work.duplicate_theme_from_gui("Dark", "Workshop")?`.
    fn duplicate_theme_from_gui(&mut self, base: &str, name: &str) -> Result<bool, String> {
        Err(format!(
            "theme {base:?} -> {name:?}: expected an application theme owner"
        ))
    }
    /// Delete an active custom; example: `work.delete_theme_from_gui("Workshop")?`.
    fn delete_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        Err(format!(
            "theme {name:?}: expected an application theme owner"
        ))
    }
    /// Save the active custom snapshot; example: `work.save_theme_from_gui(name)?`.
    fn save_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        Err(format!(
            "theme {name:?}: expected an application theme storage owner"
        ))
    }

    /// Apply the canonical palette selection; example: `work.select_color_scheme_from_gui(&request)?`.
    fn select_color_scheme_from_gui(
        &mut self,
        request: &oblikovati_api::wire::SetColorSchemeArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "colour scheme {:?}: expected an application palette owner",
            request.name
        ))
    }

    /// Select the application theme; example: `work.select_theme_from_gui("Light")?`.
    fn select_theme_from_gui(&mut self, name: &str) -> Result<bool, String> {
        Err(format!(
            "theme {name:?}: expected an application theme owner"
        ))
    }

    /// Activate a visible document through its owner; example: `work.activate_document_from_gui(&request)?`.
    fn activate_document_from_gui(
        &mut self,
        request: &oblikovati_api::wire::ActivateDocumentArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "activate document {}: expected a document owner",
            request.id
        ))
    }

    /// Close a document through its owner's save guard; example: `work.close_document_from_gui(&request)?`.
    fn close_document_from_gui(
        &mut self,
        request: &oblikovati_api::wire::CloseDocumentArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "close document {}: expected an owner with document file support",
            request.id
        ))
    }
    /// Open a chosen document through the document owner; example: `work.open_document_from_gui(&request)?`.
    fn open_document_from_gui(
        &mut self,
        request: &oblikovati_api::wire::OpenDocumentArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "open document {:?}: expected an owner with document file support",
            request.full_document_name
        ))
    }
    /// Activate a stable sketch selected in the model browser.
    /// Example: `work.edit_existing_sketch_from_gui(document, sketch)?` after a double-click.
    fn edit_existing_sketch_from_gui(
        &mut self,
        document: u64,
        sketch: oblikovati_model::document_sketch::DocumentSketchId,
    ) -> Result<bool, String> {
        Err(format!(
            "browser sketch {sketch:?}, document {document}: expected an owner with sketch edit support"
        ))
    }
    /// Observe the current drawable before admitting API capture requests.
    /// Example: `work.observe_window_capture_extent(Some([1440, 900]))` before polling.
    fn observe_window_capture_extent(&mut self, _: Option<[u32; 2]>) {}
    /// Transfer bounded capture work to the native window; example: `work.take_window_capture_requests()`.
    fn take_window_capture_requests(&mut self) -> Vec<crate::WindowCaptureJob> {
        Vec::new()
    }
    /// Activate a supported placement engine; example: `work.place_sketch_tool_from_gui(GuiSketchTool::Circle)?`.
    fn place_sketch_tool_from_gui(
        &mut self,
        tool: oblikovati_gui::GuiSketchTool,
    ) -> Result<bool, String> {
        Err(format!("{tool:?}: expected an owner with an active sketch"))
    }
    /// Enter or finish the owner sketch edit; example: `work.edit_sketch_from_gui(true)?`.
    fn edit_sketch_from_gui(&mut self, start: bool) -> Result<bool, String> {
        Err(format!(
            "GUI sketch start={start}: expected a sketch document owner"
        ))
    }
    /// Create a part through the document owner; example: `work.create_part_document()?`.
    fn create_part_document(&mut self) -> Result<bool, String> {
        Err("New Part: expected a document owner with part creation support".into())
    }
    /// Consume a committed authoring update; example: `work.take_sketch_scene()?` after polling history.
    fn take_sketch_scene(&mut self) -> Result<Option<crate::HeadSketchSceneUpdate>, String> {
        Ok(None)
    }

    /// End an edit whose view could not be prepared; example: `work.decline_sketch_edit(cause)?` after a camera refusal.
    fn decline_sketch_edit(&mut self, cause: String) -> Result<(), String> {
        Err(format!(
            "sketch edit framing {cause}: expected an owner that can cancel the edit"
        ))
    }

    /// Copy the current placed sketch edit; example: `work.sketch_edit()` before camera synchronization.
    fn sketch_edit(&self) -> Option<crate::HeadSketchEdit> {
        None
    }

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
    /// Apply a parameter editor request on the document owner; example: `work.edit_parameter_from_gui(GuiParameterMutation::Add, &request)?`.
    fn edit_parameter_from_gui(
        &mut self,
        mutation: oblikovati_gui::GuiParameterMutation,
        request: &oblikovati_api::wire::ParameterSetArgs,
    ) -> Result<bool, String> {
        Err(format!(
            "GUI parameter {mutation:?} {request:?}: expected a document parameter owner"
        ))
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

// Keep authoring state on the same document owner when scripts or add-ins wrap frame work.
macro_rules! forward_owner_view_work {
    ($owner:ident) => {
        fn activate_document_from_gui(
            &mut self,
            request: &oblikovati_api::wire::ActivateDocumentArgs,
        ) -> Result<bool, String> {
            self.$owner.activate_document_from_gui(request)
        }

        fn close_document_from_gui(
            &mut self,
            request: &oblikovati_api::wire::CloseDocumentArgs,
        ) -> Result<bool, String> {
            self.$owner.close_document_from_gui(request)
        }
        fn open_document_from_gui(
            &mut self,
            request: &oblikovati_api::wire::OpenDocumentArgs,
        ) -> Result<bool, String> {
            self.$owner.open_document_from_gui(request)
        }
        fn edit_existing_sketch_from_gui(
            &mut self,
            document: u64,
            sketch: oblikovati_model::document_sketch::DocumentSketchId,
        ) -> Result<bool, String> {
            self.$owner.edit_existing_sketch_from_gui(document, sketch)
        }
        fn observe_window_capture_extent(&mut self, extent: Option<[u32; 2]>) {
            self.$owner.observe_window_capture_extent(extent);
        }
        fn take_window_capture_requests(&mut self) -> Vec<$crate::WindowCaptureJob> {
            self.$owner.take_window_capture_requests()
        }
        // #3796: an add-in layer must retain the same UI authoring operations as the script layer.
        fn create_part_document(&mut self) -> Result<bool, String> {
            self.$owner.create_part_document()
        }
        fn edit_sketch_from_gui(&mut self, start: bool) -> Result<bool, String> {
            self.$owner.edit_sketch_from_gui(start)
        }
        fn place_sketch_tool_from_gui(
            &mut self,
            tool: oblikovati_gui::GuiSketchTool,
        ) -> Result<bool, String> {
            self.$owner.place_sketch_tool_from_gui(tool)
        }
        fn decline_sketch_edit(&mut self, cause: String) -> Result<(), String> {
            self.$owner.decline_sketch_edit(cause)
        }
        fn sketch_edit(&self) -> Option<$crate::HeadSketchEdit> {
            self.$owner.sketch_edit()
        }
        fn sketch_preview(&self) -> Option<oblikovati_viewport::ViewportSketchPreview> {
            self.$owner.sketch_preview()
        }
        fn take_sketch_scene(&mut self) -> Result<Option<$crate::HeadSketchSceneUpdate>, String> {
            self.$owner.take_sketch_scene()
        }
        fn take_scene(&mut self) -> Result<Option<$crate::HeadSceneUpdate>, String> {
            self.$owner.take_scene()
        }
    };
}
pub(crate) use forward_owner_view_work;

#[cfg(test)]
mod document_activation_tests {
    use super::{HeadWork, IdleHeadWork};
    #[test]
    fn an_owner_without_documents_names_the_refused_activation() {
        let failure = IdleHeadWork
            .activate_document_from_gui(&oblikovati_api::wire::ActivateDocumentArgs { id: 41 })
            .unwrap_err();
        assert!(failure.contains("41") && failure.contains("document owner"));
    }
}
