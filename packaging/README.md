# Native package inputs

See [RELEASING.md](../RELEASING.md#application-integration-contract) for required
resources, binary metadata, released dependency pins, and native smoke tests.

The resources and smoke tests must come from the application port. They are
deliberately absent now. Packaging fails by name when a required input is absent;
it never publishes a build tool as the application or an empty package as a pass.
