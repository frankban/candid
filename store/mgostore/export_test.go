// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package mgostore

import (
	"time"

	"golang.org/x/net/context"

	"github.com/CanonicalLtd/candid/meeting"
)

var PutAtTime = func(ctx context.Context, s meeting.Store, id, address string, now time.Time) error {
	return s.(*meetingStore).put(ctx, id, address, now)
}
