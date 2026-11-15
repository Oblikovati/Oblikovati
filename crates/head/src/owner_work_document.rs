// SPDX-License-Identifier: GPL-2.0-only
//! Document lifecycle methods on the desktop work boundary.
macro_rules! desktop_document_work {
    () => {
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
