// SPDX-License-Identifier: GPL-2.0-only
//! Document lifecycle methods on the desktop work boundary.
macro_rules! desktop_document_work {
    () => {
        fn project_file_from_gui(
            &mut self,
            action: &oblikovati_gui::GuiProjectAction,
        ) -> Result<bool, String> {
            use oblikovati_gui::GuiProjectAction;
            let (request, result) = match action {
                GuiProjectAction::Create {
                    request,
                    workspace,
                    name,
                } => (
                    request,
                    self.files
                        .create_project(workspace, name)
                        .map(|project| project.path.display().to_string())
                        .map_err(|cause| cause.to_string()),
                ),
                GuiProjectAction::Open { request, path } => {
                    (request, self.load_desktop_project(path))
                }
            };
            self.project_file_result = Some((*request, result));
            Ok(true)
        }
        fn drawing_layout_from_gui(
            &mut self,
            request: &oblikovati_gui::GuiDrawingLayoutAction,
        ) -> Result<bool, String> {
            self.drawing_layout_action(request)
        }
        fn load_full_from_gui(&mut self, id: u64) -> Result<bool, String> {
            oblikovati_api::client::Client::new(self.document_host())
                .documents()
                .load_full(&oblikovati_api::wire::LoadFullDocumentArgs { id })
                .map_err(|cause| cause.to_string())?;
            Ok(true)
        }
        fn open_document_from_gui(
            &mut self,
            request: &oblikovati_api::wire::OpenDocumentArgs,
        ) -> Result<bool, String> {
            let opened = oblikovati_api::client::Client::new(self.document_host())
                .documents()
                .open(request)
                .map_err(|cause| cause.to_string())?;
            self.note_recent_file(&request.full_document_name);
            self.command_notice = Some(DesktopCommandNotice::DocumentOpened {
                document: opened.id,
                path: request.full_document_name.clone(),
            });
            Ok(true)
        }
        fn open_deferred_drawing_from_gui(&mut self, path: &str) -> Result<bool, String> {
            let document = self
                .document_host()
                .open_deferred_drawing_path(std::path::Path::new(path))
                .map_err(|cause| cause.to_string())?;
            self.note_recent_file(path);
            self.command_notice = Some(DesktopCommandNotice::DocumentOpened {
                document,
                path: path.to_owned(),
            });
            Ok(true)
        }
        fn close_document_from_gui(
            &mut self,
            request: &oblikovati_api::wire::CloseDocumentArgs,
        ) -> Result<bool, String> {
            let result = oblikovati_api::client::Client::new(self.document_host())
                .documents()
                .close(request)
                .map_err(|cause| cause.to_string())?;
            self.forget_document_windows(request.id);
            self.sketch_memory.curvature.forget_document(request.id);
            self.command_notice = Some(match super::commands::warning_details(result.warnings) {
                Some(detail) => DesktopCommandNotice::ClosedWithWarning {
                    document: request.id,
                    detail,
                },
                None => DesktopCommandNotice::DocumentClosed {
                    document: request.id,
                },
            });
            Ok(true)
        }
    };
}
