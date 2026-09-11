module github.com/klinova/kinara-os/pkg/events

go 1.21

require (
	github.com/google/uuid v1.4.0
	github.com/jackc/pgx/v5 v5.9.0
	github.com/klinova/kinara-os/pkg/auth v0.0.0
)

// Local replace for GOWORK=off CI mode. go.work resolves this in workspace mode.
replace github.com/klinova/kinara-os/pkg/auth => ../auth
