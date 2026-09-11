package events

import "errors"

// MaxHop is the maximum derivation depth for an event chain.
// PublishDerived returns ErrHopLimitExceeded when parent.Hop == MaxHop.
const MaxHop = 5

// ErrNoClaims is returned by Publish when the context carries no tenant claims.
// This happens when the handler is reached without JWT middleware, which is a
// configuration error in the calling service.
var ErrNoClaims = errors.New("events: no tenant claims in context")

// ErrHopLimitExceeded is returned by PublishDerived when the resulting event
// would exceed MaxHop. Callers should treat this as a terminal condition for
// the event chain — log and discard rather than retry.
var ErrHopLimitExceeded = errors.New("events: hop limit exceeded (max 5)")
