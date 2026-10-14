// SPDX-License-Identifier: GPL-2.0-only
use super::*;
fn get_view(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    preferences: &mut dyn SketchPreferenceAccess,
) -> OptionGroupView {
    Client::new(FileHost::new(files, source, "test").with_sketch_preferences(preferences))
        .options()
        .get_group(&GetOptionGroupArgs {
            group: OPTION_GROUP_SKETCH.into(),
        })
        .unwrap()
}
fn set_view(
    files: &mut DiscoveryWorkspace,
    source: &mut QueryGuids,
    preferences: &mut dyn SketchPreferenceAccess,
    view: &OptionGroupView,
) -> Result<OKResult, oblikovati_api::Error> {
    Client::new(FileHost::new(files, source, "test").with_sketch_preferences(preferences))
        .options()
        .set_sketch(view)
}

#[test]
fn hud_options_round_trip_through_the_injected_preference_owner() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let mut preferences = persisted_preferences(false);
    let client = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_sketch_preferences(&mut preferences),
    );
    let mut view = client.application().hud_options().unwrap();
    assert!(!view.enabled);
    view.dimension_input_in_cartesian_coordinates = true;
    view.create_dimensions_on_value_input = false;
    assert_eq!(client.application().set_hud_options(&view).unwrap(), view);
    assert_eq!(client.application().hud_options().unwrap(), view);
    assert!(preferences.sketch_options().dimension_input_cartesian);
    assert!(
        !preferences
            .sketch_options()
            .create_dimensions_on_value_input
    );
}

#[test]
fn failed_hud_option_save_keeps_the_previous_preference_state() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let mut preferences = persisted_preferences(true);
    let before = preferences.sketch_options().clone();
    let client = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_sketch_preferences(&mut preferences),
    );
    let mut view = client.application().hud_options().unwrap();
    view.enabled = !view.enabled;
    expect_api_decline(client.application().set_hud_options(&view), "write refused");
    drop(client);
    assert_eq!(preferences.sketch_options(), &before);
}

#[test]
fn injected_preference_owner_is_shared_between_api_calls_and_later_creations() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let mut preferences = persisted_preferences(false);
    let mut view = get_view(&mut files, &mut source, &mut preferences);
    assert!(view.sketch.as_ref().unwrap().auto_project_origin);
    let before = owner_revision(&files, document);
    view.sketch.as_mut().unwrap().auto_project_origin = false;
    assert!(
        set_view(&mut files, &mut source, &mut preferences, &view)
            .unwrap()
            .ok
    );
    assert_eq!(before, owner_revision(&files, document));
    assert_eq!(get_view(&mut files, &mut source, &mut preferences), view);
    let reply = create_with_options(&mut files, &mut source, &mut preferences);
    assert_created(&files, document, reply.sketch_index, false);
}
#[test]
fn wire_writes_preserve_display_and_input_fields_omitted_from_the_contract() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let mut preferences = persisted_preferences(false);
    let before = preferences.sketch_options().clone();
    let mut view = get_view(&mut files, &mut source, &mut preferences);
    view.sketch.as_mut().unwrap().grid_spacing_cm = 2.5;
    set_view(&mut files, &mut source, &mut preferences, &view).unwrap();
    assert_eq!(preferences.sketch_options().grid_spacing_cm, 2.5);
    let retained = SketchApplicationOptions {
        grid_spacing_cm: before.grid_spacing_cm,
        ..preferences.sketch_options().clone()
    };
    assert_eq!(retained, before);
    assert!(!preferences.sketch_options().heads_up_display);
    assert!(preferences.sketch_options().suppress_format_overrides);
}
#[test]
fn a_failed_preference_save_keeps_future_origin_projection_enabled() {
    let (mut files, document, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let mut preferences = persisted_preferences(true);
    let before = preferences.sketch_options().clone();
    let mut view = get_view(&mut files, &mut source, &mut preferences);
    view.sketch.as_mut().unwrap().auto_project_origin = false;
    expect_api_decline(
        set_view(&mut files, &mut source, &mut preferences, &view),
        "write refused",
    );
    assert_eq!(preferences.sketch_options(), &before);
    let reply = create_with_options(&mut files, &mut source, &mut preferences);
    assert_created(&files, document, reply.sketch_index, true);
}
#[test]
fn zero_wire_defaults_are_rejected_until_the_valid_grid_is_supplied() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let mut preferences = persisted_preferences(false);
    let before = preferences.sketch_options().clone();
    let mut view = OptionGroupView {
        group: OPTION_GROUP_SKETCH.into(),
        sketch: Some(StdBox::default()),
        ..Default::default()
    };
    expect_api_decline(
        set_view(&mut files, &mut source, &mut preferences, &view),
        "positive finite",
    );
    assert_eq!(preferences.sketch_options(), &before);
    view.sketch.as_mut().unwrap().grid_spacing_cm = 1.0;
    set_view(&mut files, &mut source, &mut preferences, &view).unwrap();
    assert!(!preferences.sketch_options().auto_project_origin);
    let hidden = preferences.sketch_options().heads_up_display;
    assert_eq!(hidden, before.heads_up_display);
}
#[test]
fn option_group_inventory_reports_only_the_ported_sketch_group() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let mut preferences = persisted_preferences(false);
    let client = Client::new(
        FileHost::new(&mut files, &mut source, "test").with_sketch_preferences(&mut preferences),
    );
    assert_eq!(
        client.options().groups().unwrap().groups.unwrap(),
        [OPTION_GROUP_SKETCH]
    );
    assert_unported_option_groups(&client);
}

#[test]
fn absent_and_mismatched_union_payloads_leave_preferences_unchanged() {
    let view = OptionGroupView {
        group: OPTION_GROUP_SKETCH.into(),
        ..Default::default()
    };
    assert_union_decline(view, "no sketch payload");
    let view = OptionGroupView {
        group: OPTION_GROUP_SKETCH.into(),
        sketch: Some(StdBox::default()),
        general: Some(StdBox::default()),
        ..Default::default()
    };
    assert_union_decline(view, "other groups");
    assert_union_decline(unsupported_option_payload(), "unported");
}

#[test]
fn preference_calls_require_an_injected_owner_but_do_not_need_an_active_document() {
    let mut files = DiscoveryWorkspace::new(QueryObserver::default(), DiscoveryFiles::default());
    let mut source = QueryGuids(0);
    let mut preferences = persisted_preferences(false);
    {
        let host = FileHost::new(&mut files, &mut source, "test");
        for method in oblikovati_automation::SKETCH_OPTION_METHODS {
            expect_api_decline(
                host.call(method, b"{}"),
                "injected application preference owner",
            );
        }
    }
    let view = get_view(&mut files, &mut source, &mut preferences);
    assert!(view.sketch.unwrap().auto_project_origin);
}
#[test]
fn malformed_option_json_never_changes_the_preference_owner() {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let mut preferences = persisted_preferences(false);
    let before = preferences.sketch_options().clone();
    {
        let host = FileHost::new(&mut files, &mut source, "test")
            .with_sketch_preferences(&mut preferences);
        expect_api_decline(
            host.call(METHOD_OPTIONS_SET_GROUP, b"{"),
            METHOD_OPTIONS_SET_GROUP,
        );
        expect_api_decline(
            host.call(METHOD_OPTIONS_GET_GROUP, b"{"),
            METHOD_OPTIONS_GET_GROUP,
        );
    }
    assert_eq!(preferences.sketch_options(), &before);
}

fn assert_union_decline(view: OptionGroupView, offending: &str) {
    let (mut files, _, mut source) = configured_files(DOCUMENT_PART, Default::default());
    let mut preferences = persisted_preferences(false);
    let before = preferences.sketch_options().clone();
    expect_api_decline(
        set_view(&mut files, &mut source, &mut preferences, &view),
        offending,
    );
    assert_eq!(preferences.sketch_options(), &before);
}
fn assert_unported_option_groups<C: Caller>(client: &Client<C>) {
    for group in [
        OPTION_GROUP_GENERAL,
        OPTION_GROUP_DISPLAY,
        OPTION_GROUP_PART,
        OPTION_GROUP_SAVE,
        "missing",
    ] {
        expect_api_decline(
            client.options().get_group(&GetOptionGroupArgs {
                group: group.into(),
            }),
            group,
        );
    }
}

fn unsupported_option_payload() -> OptionGroupView {
    OptionGroupView {
        group: "unported".into(),
        ..Default::default()
    }
}
