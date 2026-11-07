// SPDX-License-Identifier: GPL-2.0-only
//! The owner of a window with no documents: everything asked of it declines by its default.
use crate::{HeadWork, HeadWorkStatus};

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
    #[test]
    fn idle_work_declines_curve_format_commands_without_an_owner() {
        let failure = IdleHeadWork
            .toggle_sketch_format_from_gui(
                oblikovati_model::sketch::SketchCurveFormat::Construction,
            )
            .unwrap_err();
        assert!(failure.contains("Construction") && failure.contains("document editor owner"));
    }
}
