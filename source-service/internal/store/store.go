// Package store contains source-service repositories.
// Depends only on ports.DB — no direct postgres imports.
package store

import "github.com/mrjvadi/creatorbot/shared/pkg/ports"

type Store struct{ db ports.DB }
func New(db ports.DB) *Store { return &Store{db: db} }
