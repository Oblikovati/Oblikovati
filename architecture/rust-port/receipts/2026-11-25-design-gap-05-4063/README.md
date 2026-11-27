# Native feature properties layout

The test build started from `1ef1a734f` and included the properties refactor,
in-progress File and Home styling, and existing local document-dialog edits.
The Linux runs used Xvfb, Openbox, and Vulkan Lavapipe.

- The native frame placed the Chamfer label and editable distance on the
  same baseline, with the input starting in the right column. The main
  action uses the theme's primary button. [Chamfer](Chamfer.png)
- Extrude displayed the selected sketch breadcrumb, inline Distance A,
  and the four direction buttons in a single row. [Extrude](Extrude.png)
- The published Distance A menu scenario passed all nine steps. It opened
  a copy of the supplied Part, checked the five menu entries and authored
  parameters, and verified that Cancel retained the 8 mm extrusion.
  [Cucumber](cucumber.json) · [JUnit](junit.xml) ·
  [Scenario arguments](scenario-arguments.json)
