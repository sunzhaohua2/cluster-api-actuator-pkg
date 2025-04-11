package capi

import (
	"context"
)

type MachineSetAPI interface {
	CreateMachineSet(ctx context.Context) error
	Cleanup(ctx context.Context)
}
