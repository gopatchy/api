package patchy_test

import (
	"context"
	"testing"

	"github.com/gopatchy/patchy"
	"github.com/gopatchy/patchyc"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	created1, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	created2, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	created3, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "zig"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.ElementsMatch(t, []string{"foo", "bar", "zig"}, []string{list[0].Text, list[1].Text, list[2].Text})
	require.ElementsMatch(
		t,
		[]string{created1.ID, created2.ID, created3.ID},
		[]string{list[0].ID, list[1].ID, list[2].ID},
	)
}

func TestListEquals(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{
		Filters: []*patchyc.Filter{
			{
				Path:  "text",
				Op:    "eq",
				Value: "bar",
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.ElementsMatch(t, []string{"bar"}, []string{list[0].Text})
}

func TestListInvalidOp(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{
		Filters: []*patchyc.Filter{
			{
				Path:  "text",
				Op:    "invalid",
				Value: "bar",
			},
		},
	})
	require.Error(t, err)
	require.Nil(t, list)
}

func TestListGreaterThan(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{
		Filters: []*patchyc.Filter{
			{
				Path:  "text",
				Op:    "gt",
				Value: "bar",
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.ElementsMatch(t, []string{"foo"}, []string{list[0].Text})
}

func TestListGreaterThanOrEqual(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "zig"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{
		Filters: []*patchyc.Filter{
			{
				Path:  "text",
				Op:    "gte",
				Value: "foo",
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.ElementsMatch(t, []string{"foo", "zig"}, []string{list[0].Text, list[1].Text})
}

func TestListHasPrefix(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{
		Filters: []*patchyc.Filter{
			{
				Path:  "text",
				Op:    "hp",
				Value: "f",
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.ElementsMatch(t, []string{"foo"}, []string{list[0].Text})
}

func TestListIn(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{
		Filters: []*patchyc.Filter{
			{
				Path:  "text",
				Op:    "in",
				Value: "foo,zig",
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.ElementsMatch(t, []string{"foo"}, []string{list[0].Text})
}

func TestListLessThan(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{
		Filters: []*patchyc.Filter{
			{
				Path:  "text",
				Op:    "lt",
				Value: "foo",
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.ElementsMatch(t, []string{"bar"}, []string{list[0].Text})
}

func TestListLessThanOrEqual(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "zig"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{
		Filters: []*patchyc.Filter{
			{
				Path:  "text",
				Op:    "lte",
				Value: "foo",
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.ElementsMatch(t, []string{"foo", "bar"}, []string{list[0].Text, list[1].Text})
}

func TestListLimit(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{Limit: 1})
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Contains(t, []string{"foo", "bar"}, list[0].Text)
}

func TestListOffset(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "zig"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{Offset: 1})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Contains(t, []string{"foo", "bar", "zig"}, list[0].Text)
	require.Contains(t, []string{"foo", "bar", "zig"}, list[1].Text)
	require.NotEqual(t, list[0].Text, list[1].Text)
}

func TestListAfter(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "zig"})
	require.NoError(t, err)

	list1, err := patchyc.List[testType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.Len(t, list1, 3)

	list2, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{After: list1[0].ID})
	require.NoError(t, err)
	require.Len(t, list2, 2)
	require.Equal(t, list2[0].Text, list1[1].Text)
	require.Equal(t, list2[1].Text, list1[2].Text)
}

func TestListSort(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "zig"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{Sorts: []string{"text"}})
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.Equal(t, []string{"bar", "foo", "zig"}, []string{list[0].Text, list[1].Text, list[2].Text})
}

func TestListSortAsc(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "zig"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{Sorts: []string{"+text"}})
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.Equal(t, []string{"bar", "foo", "zig"}, []string{list[0].Text, list[1].Text, list[2].Text})
}

func TestListSortDesc(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "zig"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{Sorts: []string{"-text"}})
	require.NoError(t, err)
	require.Len(t, list, 3)
	require.Equal(t, []string{"zig", "foo", "bar"}, []string{list[0].Text, list[1].Text, list[2].Text})
}

func TestListSortBeforeOffset(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "zig"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{
		Offset: 1,
		Sorts:  []string{"text"},
	})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, []string{"foo", "zig"}, []string{list[0].Text, list[1].Text})
}

func TestListSortBeforeLimit(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "zig"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{
		Limit: 2,
		Sorts: []string{"text"},
	})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, []string{"bar", "foo"}, []string{list[0].Text, list[1].Text})
}

func TestListPrev(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, "foo", list[0].Text)

	// Doesn't actually verify that the list isn't sent over the wire, just that it doesn't fail.
	list2, err := patchyc.List[testType](ctx, ta.pyc, &patchyc.ListOpts{Prev: list})
	require.NoError(t, err)
	require.Len(t, list2, 1)
	require.Equal(t, "foo", list2[0].Text)
}

func TestListHook(t *testing.T) {
	t.Parallel()

	ta := newTestAPI(t)
	defer ta.shutdown(t)

	patchy.SetListHook[testType](ta.api, func(_ context.Context, opts *patchy.ListOpts, _ *patchy.API) error {
		opts.Filters = append(opts.Filters, &patchy.Filter{
			Path:  "text",
			Op:    "gt",
			Value: "eek",
		})

		return nil
	})

	ctx := context.Background()

	_, err := patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "foo"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "bar"})
	require.NoError(t, err)

	_, err = patchyc.Create[testType](ctx, ta.pyc, &testType{Text: "zig"})
	require.NoError(t, err)

	list, err := patchyc.List[testType](ctx, ta.pyc, nil)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.ElementsMatch(t, []string{"foo", "zig"}, []string{list[0].Text, list[1].Text})
}
