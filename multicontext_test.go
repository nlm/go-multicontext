package multicontext

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithContexts(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, []context.Context{ctx}, WithContexts(ctx).(MultiContext).Ctxs)
}

type keyType string

var testKey = keyType("hello")
var testKeyUnknown = keyType("nonexistent")
var testKeyEmpty = keyType("")
var testValue = "world"
var testDate = time.Now()
var testCases = map[string]context.Context{
	"todo":       context.TODO(),
	"background": context.Background(),
	"value":      context.WithValue(context.Background(), testKey, testValue),
	"deadline": func() context.Context {
		ctx, cancel := context.WithDeadline(context.Background(), testDate)
		defer cancel()
		return ctx
	}(),
	"timeout": func() context.Context {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		return ctx
	}(),
	"cancel": func() context.Context {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		return ctx
	}(),
}

func TestMultiContext_Deadline(t *testing.T) {
	for name, sctx := range testCases {
		t.Run(name, func(t *testing.T) {
			mctx := WithContexts(sctx)
			sdeadline, sok := sctx.Deadline()
			mdeadline, mok := mctx.Deadline()
			assert.Equal(t, sdeadline, mdeadline)
			assert.Equal(t, sok, mok)
		})
	}
}

func TestMultiContext_Err(t *testing.T) {
	for name, sctx := range testCases {
		t.Run(name, func(t *testing.T) {
			mctx := WithContexts(sctx)
			serr := sctx.Err()
			merr := mctx.Err()
			assert.Equal(t, serr, merr)
		})
	}
}

func TestMultiContext_Value(t *testing.T) {
	for name, sctx := range testCases {
		for _, key := range []keyType{
			testKey,
			testKeyEmpty,
			testKeyUnknown,
		} {
			t.Run(fmt.Sprint(name, "+key(", key, ")"), func(t *testing.T) {
				mctx := WithContexts(sctx)
				serr := sctx.Value(key)
				merr := mctx.Value(key)
				assert.Equal(t, serr, merr)
			})
		}
	}
}

func TestMultiContext_Done(t *testing.T) {
	for name, sctx := range testCases {
		t.Run(name, func(t *testing.T) {
			mctx := WithContexts(sctx)
			var sx, mx struct{}
			var sok, mok bool
			select {
			case sx, sok = <-sctx.Done():
				t.Log("single: received from channel")
			default:
				t.Log("single: defaulted")
			}
			select {
			case mx, mok = <-mctx.Done():
				t.Log("multi: received from channel")
			default:
				t.Log("multi: defaulted")
			}
			assert.Equal(t, sok, mok)
			assert.Equal(t, sx, mx)
		})
	}
}

func TestMultiContext_Deadline_multi(t *testing.T) {
	const iter = 10
	ctxs := make([]context.Context, 0)
	now := time.Now()
	for i := 0; i < iter; i++ {
		ctx, cancel := context.WithDeadline(context.Background(), now.Add(time.Duration(i)*time.Minute))
		defer cancel()
		ctxs = append(ctxs, ctx)
		t.Log(ctx)
	}
	for i := 0; i < iter; i++ {
		ctx, cancel := context.WithDeadline(context.Background(), now.Add(time.Duration(-i)*time.Minute))
		defer cancel()
		ctxs = append(ctxs, ctx)
		t.Log(ctx)
	}
	mctx := WithContexts(ctxs...)
	mdeadline, mok := mctx.Deadline()
	sdeadline, sok := ctxs[0].Deadline()
	assert.NotEqual(t, sdeadline, mdeadline)
	assert.Equal(t, sok, mok)
	ldeadline, lok := ctxs[len(ctxs)-1].Deadline()
	assert.Equal(t, ldeadline, mdeadline)
	assert.Equal(t, lok, mok)
}

func TestMultiContext_Value_order(t *testing.T) {
	ctx1 := context.WithValue(context.Background(), testKey, 1)
	ctx2 := context.WithValue(context.Background(), testKey, 2)
	mctx1 := WithContexts(ctx1, ctx2)
	assert.Equal(t, mctx1.Value(testKey), 1)
	mctx2 := WithContexts(ctx2, ctx1)
	assert.Equal(t, mctx2.Value(testKey), 2)
}

func TestMultiContext_Err_order(t *testing.T) {
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()
	ctx2, cancel2 := context.WithCancel(context.Background())
	cancel2()
	t.Run("single+nocancel", func(t *testing.T) {
		sctx1 := WithContexts(ctx1)
		assert.ErrorIs(t, sctx1.Err(), nil)
	})
	t.Run("single+cancel", func(t *testing.T) {
		sctx2 := WithContexts(ctx2)
		assert.ErrorIs(t, sctx2.Err(), context.Canceled)
	})
	t.Run("multi", func(t *testing.T) {
		mctx1 := WithContexts(ctx1, ctx2)
		assert.ErrorIs(t, mctx1.Err(), context.Canceled)
		mctx2 := WithContexts(ctx2, ctx1)
		assert.ErrorIs(t, mctx2.Err(), context.Canceled)
	})
}

func TestMultiContext_Empty_Deadline(t *testing.T) {
	mctx := WithContexts()
	deadline, ok := mctx.Deadline()
	assert.Zero(t, deadline)
	assert.False(t, ok)
}

func TestMultiContext_Empty_Value(t *testing.T) {
	mctx := WithContexts()
	v := mctx.Value(testKey)
	assert.Zero(t, v)
}

func TestMultiContext_Empty_Err(t *testing.T) {
	mctx := WithContexts()
	v := mctx.Err()
	assert.Zero(t, v)
}

func TestMultiContext_Empty_Done(t *testing.T) {
	mctx := WithContexts()
	v := mctx.Done()
	assert.Zero(t, v)
}
