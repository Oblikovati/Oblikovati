// SPDX-License-Identifier: GPL-2.0-only
//! Make rooted in-memory document fixture paths fully qualified on every host.

macro_rules! document_fixture_path {
    ($rooted:literal) => {{
        #[cfg(windows)]
        let path = concat!("C:", $rooted);
        #[cfg(not(windows))]
        let path = $rooted;
        path
    }};
}
