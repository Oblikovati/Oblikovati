// SPDX-License-Identifier: GPL-2.0-only
use crate::*;
use oblikovati_api::wire::METHOD_DOCUMENTS_LIST_ATTACHMENTS;
const FILE_METHOD_GROUPS: [&[&str]; 19] = [
    &QUERY_METHODS,
    &ASSET_QUERY_METHODS,
    &ASSET_COMMAND_METHODS,
    &ASSET_ASSIGNMENT_METHODS,
    &BODY_QUERY_METHODS,
    &BODY_COMMAND_METHODS,
    &FEATURE_LIFECYCLE_METHODS,
    &FILE_QUERY_METHODS,
    &FILE_COMMAND_METHODS,
    &APPLICATION_PREFERENCE_METHODS,
    &CLIENT_COMMAND_METHODS,
    &EDIT_METHODS,
    &TRANSACTION_METHODS,
    &SKETCH_COMMAND_METHODS,
    &SKETCH_QUERY_METHODS,
    &DATUM_COMMAND_METHODS,
    &DATUM_QUERY_METHODS,
    &DERIVED_COMMAND_METHODS,
    &[METHOD_DOCUMENTS_LIST_ATTACHMENTS],
];
/// Discover the ported file-owner routes in stable order; argument/owner preconditions still apply.
/// Example: `ScriptProgram { methods: file_host_methods(), ..program }` builds its Lua groups from these names.
pub fn file_host_methods() -> Vec<String> {
    let mut methods: Vec<String> = FILE_METHOD_GROUPS
        .into_iter()
        .flatten()
        .map(|method| (*method).into())
        .collect();
    methods.sort();
    methods.dedup();
    methods
}
#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn inventories_include_every_group_once_without_unconfigured_preferences() {
        let methods = file_host_methods();
        assert!(methods.windows(2).all(|pair| pair[0] < pair[1]));
        assert_groups(&methods);
        assert!(
            !methods
                .iter()
                .any(|name| SKETCH_OPTION_METHODS.contains(&name.as_str()))
        );
    }
    fn assert_groups(methods: &[String]) {
        for group in [
            &FILE_COMMAND_METHODS[..],
            &APPLICATION_PREFERENCE_METHODS,
            &ASSET_QUERY_METHODS,
            &ASSET_COMMAND_METHODS,
            &ASSET_ASSIGNMENT_METHODS,
            &BODY_QUERY_METHODS,
            &BODY_COMMAND_METHODS,
            &FEATURE_LIFECYCLE_METHODS,
            &SKETCH_COMMAND_METHODS,
            &EDIT_METHODS,
            &QUERY_METHODS,
        ] {
            assert!(
                group
                    .iter()
                    .all(|name| methods.iter().any(|method| method == name))
            );
        }
    }
}
