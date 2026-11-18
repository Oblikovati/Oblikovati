# Sketch Constrain menu checks

Four focused runs passed on Linux with Xvfb, Openbox and Vulkan Lavapipe. They
used the `ribbon-panel-expansion` feature hash
`645576c3b62e89cdb6c668d5579123e6880a33f9e4a8020f922f1b08e0502ca2`.
The menu, switch and dock runs used commit `9bb19664e`; the Show Panels run
used that commit with the checks later committed as `cbe8bbb11`.

These focused runs do not replace the full-feature diagnostic or increase its
scenario count. The feature remains incomplete until a full current run passes.
