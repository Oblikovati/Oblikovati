// SPDX-License-Identifier: GPL-2.0-only
//! The file dialogs, and the prompts that protect unsaved work when a document closes.
#[path = "close_prompt.rs"]
pub(crate) mod close_prompt;
#[path = "file_list.rs"]
pub(crate) mod file_list;
#[path = "first_saves.rs"]
pub(crate) mod first_saves;
#[path = "folder_browser.rs"]
pub(crate) mod folder_browser;
#[path = "save_as.rs"]
pub(crate) mod save_as;
