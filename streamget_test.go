package patchy_test

import (
	"context"
	"testing"
	"time"

	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

func TestStreamGetHeartbeat(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	stream, err := patchyc.StreamGet[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Equal(t, "foo", s1.Text)

	time.Sleep(6 * time.Second)

	select {
	case _, ok := <-stream.Chan():
		if ok {
			require.Fail(t, "unexpected stream")
		} else {
			require.Fail(t, "unexpected closure")
		}

	default:
	}

	require.Less(t, time.Since(stream.LastEventReceived()), 6*time.Second)
}

func TestStreamGet(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	stream, err := patchyc.StreamGet[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Equal(t, "foo", s1.Text)
}

func TestStreamGetUpdate(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	stream, err := patchyc.StreamGet[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Equal(t, "foo", s1.Text)

	_, err = patchyc.Update[testType](ctx, ta.pyc, created.ID, &testType{Text: "bar"}, nil)
	require.NoError(t, err)

	s2 := stream.Read()
	require.NotNil(t, s2, stream.Error())
	require.Equal(t, "bar", s2.Text)
}

func TestStreamGetPrev(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	stream1, err := patchyc.StreamGet[testType](ctx, ta.pyc, created.ID, nil)
	require.NoError(t, err)

	defer stream1.Close()

	s1 := stream1.Read()
	require.NotNil(t, s1, stream1.Error())
	require.Equal(t, "foo", s1.Text)
	require.EqualValues(t, 0, s1.Num)

	// Validate that previous version passing only compares the ETag
	s1.Num = 1

	stream2, err := patchyc.StreamGet[testType](ctx, ta.pyc, created.ID, &patchyc.GetOpts{Prev: s1})
	require.NoError(t, err)

	defer stream2.Close()

	s2 := stream2.Read()
	require.NotNil(t, s2, stream2.Error())
	require.Equal(t, "foo", s2.Text)
	require.EqualValues(t, 1, s2.Num)
}
