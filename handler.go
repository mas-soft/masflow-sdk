package masflowsdk

import "context"

// Handler is the sync activity handler signature.
// TReq is the input type, TRes is the output type.
type Handler[TReq, TRes any] func(ctx context.Context, req TReq) (TRes, error)

// VoidHandler is the handler signature for activities that return only an error.
type VoidHandler[TReq any] func(ctx context.Context, req TReq) error

// AsyncCallbackInfo provides workflow context for async-capable activities.
// The activity can use this information to signal back to the workflow
// when its async work completes.
type AsyncCallbackInfo struct {
	WorkflowID      string
	RunID           string
	CallbackSignal  string
	CallbackTimeout string
}

// AsyncHandler is the handler signature for async-capable activities.
// The AsyncCallbackInfo provides workflow_id, run_id, and callback_signal
// so the activity can signal back when its async work completes.
type AsyncHandler[TReq, TRes any] func(ctx context.Context, req TReq, async *AsyncCallbackInfo) (TRes, error)
