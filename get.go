package patchy

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/vfaronov/httpheader"
)

type GetOpts struct {
	IfNoneMatch []httpheader.EntityTag

	// This is "any" because making GetOpts generic complicates too many things
	Prev any
}

var (
	ErrInvalidIfNoneMatch           = errors.New("invalid If-None-Match")
	ErrIfNoneMatchUnknownType       = fmt.Errorf("unknown type (%w)", ErrInvalidIfNoneMatch)
	ErrIfNoneMatchInvalidGeneration = fmt.Errorf("invalid generation (%w)", ErrInvalidIfNoneMatch)
)

func parseGetOpts(r *http.Request) *GetOpts {
	return &GetOpts{
		IfNoneMatch: httpheader.IfNoneMatch(r.Header),
	}
}
