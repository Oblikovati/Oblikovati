// SPDX-License-Identifier: GPL-2.0-only
//! Projects and project templates on the desktop work boundary.
macro_rules! project_work_methods {
    () => {
        /// Create a project file, for example `work.create_project_from_gui(1, workspace, "Gearbox")?`.
        fn create_project_from_gui(
            &mut self,
            request: u64,
            workspace: &std::path::Path,
            name: &str,
        ) -> Result<bool, String> {
            Err(format!(
                "project request {request} for {name:?} in {workspace:?}: expected a project owner"
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
