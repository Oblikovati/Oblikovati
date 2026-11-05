// SPDX-License-Identifier: GPL-2.0-only
use crate::{decode, encode};
use oblikovati_api::{Error, wire::*};
use oblikovati_model::color_scheme::{ColorScheme, ColorSchemeRegistry};

/// Application palettes are independent of document transactions.
pub const COLOR_SCHEME_METHODS: [&str; 3] = [
    METHOD_COLOR_SCHEMES_LIST,
    METHOD_COLOR_SCHEMES_GET_ACTIVE,
    METHOD_COLOR_SCHEMES_SET_ACTIVE,
];
pub(crate) fn apply(
    registry: &mut ColorSchemeRegistry,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    match method {
        METHOD_COLOR_SCHEMES_LIST => encode(&gallery_view(registry)),
        METHOD_COLOR_SCHEMES_GET_ACTIVE => encode(&scheme_view(registry, registry.active())),
        METHOD_COLOR_SCHEMES_SET_ACTIVE => select(registry, method, request),
        _ => Err(Error(format!(
            "color scheme method {method:?}: expected one of {COLOR_SCHEME_METHODS:?}"
        ))),
    }
}
fn gallery_view(registry: &ColorSchemeRegistry) -> ColorSchemesResult {
    ColorSchemesResult {
        schemes: Some(
            registry
                .schemes()
                .iter()
                .map(|scheme| scheme_view(registry, scheme))
                .collect(),
        ),
    }
}
fn select(
    registry: &mut ColorSchemeRegistry,
    method: &str,
    request: &[u8],
) -> Result<Vec<u8>, Error> {
    let request: SetColorSchemeArgs = decode(method, request)?;
    registry.set_active(&request.name).map_err(Error)?;
    encode(&scheme_view(registry, registry.active()))
}
fn scheme_view(registry: &ColorSchemeRegistry, scheme: &ColorScheme) -> ColorSchemeView {
    ColorSchemeView {
        name: scheme.name.clone(),
        active: scheme.name == registry.active().name,
        // Go's list also reports the application override on every gallery entry.
        background_type: registry.background_type(),
        screen_color: scheme.screen.clone(),
        top_screen: scheme.top.clone(),
        bottom_screen: scheme.bottom.clone(),
        highlight: scheme.highlight.clone(),
        primary_select: scheme.primary_select.clone(),
        second_select: scheme.secondary_select.clone(),
    }
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn unknown_dispatch_method_refuses_without_changing_registry() {
        let mut registry = ColorSchemeRegistry::default();
        let before = registry.clone();
        assert!(
            apply(&mut registry, "unknown", b"{}")
                .unwrap_err()
                .to_string()
                .contains("unknown")
        );
        assert_eq!(registry, before);
    }
}
