package helper

import (
	"context"

	"go.opentelemetry.io/otel/baggage"
)

// SetBaggage sets a key-value pair in the context baggage.
func SetBaggage(ctx context.Context, key, value string) (context.Context, error) {
	member, err := baggage.NewMember(key, value)
	if err != nil {
		return ctx, err
	}

	bag, err := baggage.New(member)
	if err != nil {
		return ctx, err
	}

	existing := baggage.FromContext(ctx)
	for _, m := range existing.Members() {
		if m.Key() != key {
			newMember, err := baggage.NewMember(m.Key(), m.Value())
			if err != nil {
				continue
			}
			bag, _ = bag.SetMember(newMember)
		}
	}

	return baggage.ContextWithBaggage(ctx, bag), nil
}

// GetBaggage retrieves a value from the context baggage.
func GetBaggage(ctx context.Context, key string) string {
	bag := baggage.FromContext(ctx)
	return bag.Member(key).Value()
}
