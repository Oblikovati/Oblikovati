// SPDX-License-Identifier: GPL-2.0-only

//! Storage boundaries for complete document snapshots.

pub mod atomic;
pub mod load;
pub mod metadata;
pub mod package;

pub mod file_access;
pub mod project;

pub mod stream_file;

pub mod attachment_files;

pub mod project_file;
