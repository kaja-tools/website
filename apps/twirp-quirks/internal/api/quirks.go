package api

import (
	"context"
	"strings"
)

type QuirksHandler struct{}

func NewQuirksHandler() *QuirksHandler {
	return &QuirksHandler{}
}

func (h *QuirksHandler) MethodWithAReallyLongNameGmthggupcbmnphflnnvu(ctx context.Context, req *Void) (*Message, error) {
	return &Message{
		Name: strings.Repeat("Ha ", 1000),
	}, nil
}

type Quirks_2Handler struct{}

func NewQuirks_2Handler() *Quirks_2Handler {
	return &Quirks_2Handler{}
}

func (h *Quirks_2Handler) CamelCaseMethod(ctx context.Context, req *Void) (*Void, error) {
	return &Void{}, nil
}
