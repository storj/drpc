// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcopts

import "storj.io/drpc/drpcstats"

// Manager contains internal options for the drpcmanager package.
type Manager struct {
	statsCB func(string) *drpcstats.Stats
}

// GetManagerStatsCB returns the stats callback stored in the options.
func GetManagerStatsCB(opts *Manager) func(string) *drpcstats.Stats { return opts.statsCB }

// SetManagerStatsCB sets the stats callback stored in the options.
func SetManagerStatsCB(opts *Manager, statsCB func(string) *drpcstats.Stats) { opts.statsCB = statsCB }
