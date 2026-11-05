// SPDX-License-Identifier: GPL-2.0-only
use crate::encode;
use oblikovati_api::{Error, client::Caller, wire::*};
use oblikovati_model::theme::ThemeLibrary;

/// Read-only theme calls share the application library, independent of document focus.
pub const THEME_QUERY_METHODS: [&str; 2] = [METHOD_THEME_ACTIVE, METHOD_THEME_LIST];

/// An immutable view of the owner's semantic theme library.
pub struct ThemeQueryHost<'a> {
    library: &'a ThemeLibrary,
}
impl<'a> ThemeQueryHost<'a> {
    /// Borrow the active application library; example: `Client::new(ThemeQueryHost::new(&library))`.
    pub fn new(library: &'a ThemeLibrary) -> Self {
        Self { library }
    }
}
impl Caller for ThemeQueryHost<'_> {
    fn call(&self, method: &str, _request: &[u8]) -> Result<Vec<u8>, Error> {
        match method {
            METHOD_THEME_ACTIVE => encode(&self.active_view()),
            METHOD_THEME_LIST => encode(&self.list_view()),
            _ => Err(Error(format!(
                "theme method {method:?}: expected one of {THEME_QUERY_METHODS:?}"
            ))),
        }
    }
}
impl ThemeQueryHost<'_> {
    fn active_view(&self) -> ThemeView {
        let active = self.library.active();
        ThemeView {
            name: active.name().into(),
            kind: active.kind().0.to_string(),
            colors: Some(active.hex_colors()),
        }
    }
    fn list_view(&self) -> ListThemesResult {
        ListThemesResult {
            themes: Some(
                self.library
                    .themes()
                    .iter()
                    .map(|theme| ThemeSummary {
                        name: theme.name().into(),
                        kind: theme.kind().0.to_string(),
                        active: theme.name() == self.library.active().name(),
                    })
                    .collect(),
            ),
        }
    }
}
