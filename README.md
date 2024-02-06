Little golang-based application that exposes an endpoint to retreive Services, or a single Service.

Uses sqlite for persistence, but could use any RDMS that `database/sql` supports.

The service uses OpenTelemetry to expose both a prometheus, and writes Otel traces to stdout.

Simple authentication using HTTP Basic Auth

Using this as something to compare with https://github.com/cskinfill/rusthacking
