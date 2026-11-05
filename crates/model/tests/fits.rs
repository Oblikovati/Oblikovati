// SPDX-License-Identifier: GPL-2.0-only

use oblikovati_model::fits::fit_band;

#[test]
fn go_capture_matches_except_documented_standard_rounding_corrections() {
    let fixtures = [
        include_str!("fixtures/go-fits-small.tsv"),
        include_str!("fixtures/go-fits-large.tsv"),
    ];
    let mut cases = 0;
    for line in fixtures
        .into_iter()
        .flat_map(str::lines)
        .filter(|line| !line.starts_with('#'))
    {
        check_go_row(line);
        cases += 1;
    }
    assert_eq!(cases, 672);
}

fn check_go_row(line: &str) {
    let columns: Vec<&str> = line.split('\t').collect();
    let nominal: f64 = columns[0].parse().unwrap();
    let class = columns[1];
    let correction = corrected_deviation(nominal as u16, class.as_bytes()[0]);
    let band = fit_band(nominal, class).unwrap();
    for (actual, captured) in [(band.upper_mm, columns[2]), (band.lower_mm, columns[3])] {
        let expected_um = (captured.parse::<f64>().unwrap() * 1000.0).round() as i32 + correction;
        assert_eq!(actual, f64::from(expected_um) / 1000.0, "{line}");
    }
}

fn corrected_deviation(nominal: u16, letter: u8) -> i32 {
    // #3573: differences against Go aa8297d1, verified against ISO 286-2 tables.
    let shaft_delta = match (nominal, letter.to_ascii_lowercase()) {
        (10 | 30 | 80 | 180 | 250, b'd') => -1,
        (315, b'd') => 1,
        (400, b'd') => 2,
        (500, b'd') => 5,
        (250 | 315 | 400 | 500, b'f') => -1,
        _ => 0,
    };
    if letter.is_ascii_uppercase() {
        -shaft_delta
    } else {
        shaft_delta
    }
}

#[test]
fn symmetric_bands_preserve_half_micrometres_for_every_step_and_grade() {
    for nominal in [
        6.0, 10.0, 18.0, 30.0, 50.0, 80.0, 120.0, 180.0, 250.0, 315.0, 400.0, 500.0,
    ] {
        for grade in 5..=11 {
            let width = fit_band(nominal, &format!("H{grade}")).unwrap().upper_mm;
            let band = fit_band(nominal, &format!("js{grade}")).unwrap();
            assert_eq!(band.upper_mm, width / 2.0);
            assert_eq!(band.lower_mm, -width / 2.0);
            assert_eq!(band, fit_band(nominal, &format!("JS{grade}")).unwrap());
        }
    }
    assert_eq!(fit_band(50.0, "js7").unwrap().upper_mm, 0.0125);
    assert_eq!(fit_band(25.0, "JS7").unwrap().lower_mm, -0.0105);
}

#[test]
fn step_boundaries_are_upper_inclusive() {
    for boundary in [
        6.0_f64, 10.0, 18.0, 30.0, 50.0, 80.0, 120.0, 180.0, 250.0, 315.0, 400.0,
    ] {
        let below = fit_band(boundary.next_down(), "H7").unwrap();
        let exact = fit_band(boundary, "H7").unwrap();
        let above = fit_band(boundary.next_up(), "H7").unwrap();
        assert_eq!(below, exact);
        assert!(above.upper_mm > exact.upper_mm);
    }
    assert!(fit_band(3.0_f64.next_up(), "H7").is_ok());
    assert!(fit_band(500.0, "H7").is_ok());
}

#[test]
fn malformed_and_unsupported_inputs_are_named_errors() {
    for class in [
        "",
        "H",
        "7",
        "H4",
        "H12",
        "H07",
        "H+7",
        "H7x",
        " H7",
        "H7 ",
        "jS7",
        "x7",
        "é7",
        "H99999999999",
    ] {
        let error = fit_band(50.0, class).unwrap_err();
        assert_eq!(error.class, class);
        assert!(error.to_string().contains("expected"));
        assert!(error.to_string().contains(&format!("{class:?}")));
    }
    for nominal in [
        f64::NAN,
        f64::INFINITY,
        f64::NEG_INFINITY,
        -1.0,
        0.0,
        3.0,
        500.0_f64.next_up(),
    ] {
        let error = fit_band(nominal, "H7").unwrap_err();
        assert!(error.to_string().contains("(3, 500]"));
    }
}
