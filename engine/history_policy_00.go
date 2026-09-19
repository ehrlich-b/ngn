//go:build !h1producer && !h1consumer

package engine

const (
	h1NewHistoryProducer    = false
	h1NormalizedLMRConsumer = false
)

type h1CompletedQuietBuffer struct{}

func h1AppendCompletedQuiet(*h1CompletedQuietBuffer, *int, Move) {}
func h1CompletedQuiets(*h1CompletedQuietBuffer, int) []Move      { return nil }
