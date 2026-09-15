package presentation

import (
	"context"
	"testing"
)

// The bridge and the exit tunnel both need to hear about a rebind, so OnRebind
// is a list and not a single slot: a second registration must not silently
// replace the first, and every subscriber fires on every notification, in
// registration order.
func TestOnRebindFiresEverySubscriber(t *testing.T) {
	manager, _ := newTestManager(t)
	ctx := context.Background()

	var order []string
	manager.OnRebind(func(context.Context) { order = append(order, "bridge") })
	manager.OnRebind(func(context.Context) { order = append(order, "exit") })
	manager.OnRebind(nil)

	if err := manager.Rebind(ctx); err != nil {
		t.Fatalf("rebind: %v", err)
	}
	if len(order) != 2 || order[0] != "bridge" || order[1] != "exit" {
		t.Fatalf("subscribers fired as %v, want [bridge exit]", order)
	}

	if err := manager.Apply(ctx, ProfileStandard); err != nil {
		t.Fatalf("apply standard: %v", err)
	}
	if len(order) != 4 {
		t.Fatalf("an apply fired %d subscriber calls in total, want 4", len(order))
	}
}
