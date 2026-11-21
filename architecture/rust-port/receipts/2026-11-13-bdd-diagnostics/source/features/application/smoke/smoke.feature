Feature: diagnostic native window
  Scenario: continue after a failed native assertion
    Given an empty part document is active in the MCAD Software
    Then a recent-documents region is visible
    When I select File
    Then a recent-documents region is visible
  Scenario: next window starts clean
    Given an empty part document is active in the MCAD Software
    Then the File menu closes
