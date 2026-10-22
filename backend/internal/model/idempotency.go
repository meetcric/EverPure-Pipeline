package model

// IdempotencyRecord is the response saved under an Idempotency-Key.
// Body is the exact JSON we sent, so a replay is byte-for-byte the original.
type IdempotencyRecord struct {
	RequestHash string
	StatusCode  int
	Body        []byte
}
