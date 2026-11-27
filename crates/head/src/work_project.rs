// SPDX-License-Identifier: GPL-2.0-only
//! Projects and project templates on the desktop work boundary.
macro_rules! project_work_methods {
    () => {
        /// Handle a Projects request, for example `work.project_file_from_gui(&action)?`.
        fn project_file_from_gui(
            &mut self,
            action: &oblikovati_gui::GuiProjectAction,
        ) -> Result<bool, String> {
            Err(format!(
                "project request {action:?}: expected a project owner"
            ))
        }
        /// Create a document from the catalog file, as in `work.create_project_template_document(&entry)?`.
        fn create_project_template_document(
            &mut self,
            template: &oblikovati_gui::GuiProjectTemplate,
        ) -> Result<bool, String> {
            Err(format!(
                "Project template {:?}: expected a document owner",
                template.path
            ))
        }
    };
}
pub(crate) use project_work_methods;
