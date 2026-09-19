//go:build h1producer && !h1consumer

package engine

const (
	h1NewHistoryProducer    = true
	h1NormalizedLMRConsumer = false
)

type h1CompletedQuietBuffer [256]Move

func h1AppendCompletedQuiet(buffer *h1CompletedQuietBuffer, count *int, move Move) {
	if *count < len(buffer) {
		buffer[*count] = move
		*count = *count + 1
	}
}

func h1CompletedQuiets(buffer *h1CompletedQuietBuffer, count int) []Move {
	return buffer[:count]
}
