package gotest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"test/goclient"
)

func TestStreamGetHeartbeat(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestTypeRequest{Text: "foo"})
	require.NoError(t, err)

	stream, err := c.StreamGetTestType(ctx, created.ID, nil)
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

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestTypeRequest{Text: "foo"})
	require.NoError(t, err)

	stream, err := c.StreamGetTestType(ctx, created.ID, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Equal(t, "foo", s1.Text)
}

func TestStreamGetUpdate(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestTypeRequest{Text: "foo"})
	require.NoError(t, err)

	stream, err := c.StreamGetTestType(ctx, created.ID, nil)
	require.NoError(t, err)

	defer stream.Close()

	s1 := stream.Read()
	require.NotNil(t, s1, stream.Error())
	require.Equal(t, "foo", s1.Text)

	_, err = c.UpdateTestType(ctx, created.ID, &goclient.TestTypeRequest{Text: "bar"}, nil)
	require.NoError(t, err)

	s2 := stream.Read()
	require.NotNil(t, s2, stream.Error())
	require.Equal(t, "bar", s2.Text)
}

func TestStreamGetPrev(t *testing.T) {
	t.Parallel()

	defer registerTest(t)()
	c := getClient(t)
	ctx := context.Background()

	created, err := c.CreateTestType(ctx, &goclient.TestTypeRequest{Text: "foo"})
	require.NoError(t, err)

	stream1, err := c.StreamGetTestType(ctx, created.ID, nil)
	require.NoError(t, err)

	defer stream1.Close()

	s1 := stream1.Read()
	require.NotNil(t, s1, stream1.Error())
	require.Equal(t, "foo", s1.Text)
	require.EqualValues(t, 0, s1.Num)

	// Validate that previous version passing only compares the ETag
	s1.Num = 1

	stream2, err := c.StreamGetTestType(ctx, created.ID, &goclient.GetOpts{Prev: s1})
	require.NoError(t, err)

	defer stream2.Close()

	s2 := stream2.Read()
	require.NotNil(t, s2, stream2.Error())
	require.Equal(t, "foo", s2.Text)
	require.EqualValues(t, 1, s2.Num)
}
