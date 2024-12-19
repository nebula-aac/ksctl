package waiter

import (
	"context"
	"errors"
	"gotest.tools/v3/assert"
	"testing"
	"time"
)

func TestWaiterRun_SuccessOnFirstAttempt(t *testing.T) {
	ctx := context.Background()

	executeFunc := func() error {
		return nil
	}

	isSuccessful := func() bool {
		return true
	}

	errorFunc := func(err error) (error, bool) {
		return nil, false
	}

	successFunc := func() error {
		return nil
	}

	backOff := NewWaiter(1*time.Second, 1, 3)

	err := backOff.Run(ctx, log, executeFunc, isSuccessful, errorFunc, successFunc, "Waiting message")
	assert.Assert(t, err == nil)
}

func TestWaiterRun_RetryOnFailure(t *testing.T) {
	ctx := context.Background()

	callCount := 0
	executeFunc := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("execute error")
		}
		return nil
	}

	isSuccessful := func() bool {
		return callCount == 3
	}

	errorFunc := func(err error) (error, bool) {
		return nil, false
	}

	successFunc := func() error {
		return nil
	}

	backOff := NewWaiter(1*time.Second, 1, 3)

	err := backOff.Run(ctx, log, executeFunc, isSuccessful, errorFunc, successFunc, "Waiting message")
	assert.Assert(t, err == nil)

	assert.Equal(t, 3, callCount)
}

func TestWaiterRun_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	executeFunc := func() error {
		return errors.New("execute error")
	}

	isSuccessful := func() bool {
		return false
	}

	errorFunc := func(err error) (error, bool) {
		return nil, false
	}

	successFunc := func() error {
		return nil
	}

	backOff := NewWaiter(1*time.Second, 1, 3)

	go func() {
		time.Sleep(2 * time.Second)
		cancel()
	}()

	err := backOff.Run(ctx, log, executeFunc, isSuccessful, errorFunc, successFunc, "Waiting message")
	assert.Assert(t, err != nil && ksctlErrors.ErrContextCancelled.Is(err))

	assert.Equal(t, context.Canceled, ctx.Err())
}

func TestWaiterRun_MaxRetriesExceeded(t *testing.T) {
	ctx := context.Background()

	executeFunc := func() error {
		return errors.New("execute error")
	}

	isSuccessful := func() bool {
		return false
	}

	errorFunc := func(err error) (error, bool) {
		return nil, false
	}

	successFunc := func() error {
		return nil
	}

	backOff := NewWaiter(1*time.Second, 1, 3)

	err := backOff.Run(ctx, log, executeFunc, isSuccessful, errorFunc, successFunc, "Waiting message")
	assert.Assert(t, err != nil && ksctlErrors.ErrTimeOut.Is(err))
}
