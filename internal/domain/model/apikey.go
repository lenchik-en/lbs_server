package model

import "time"

type KeyScope string

const (
	ScopeLocate KeyScope = "locate"
	ScopeUpdate KeyScope = "update"
	ScopeBoth   KeyScope = "both"
)

type APIKey struct {
	Key       string
	Scope     KeyScope
	IsActive  bool
	CreatedAt time.Time
}
