package store

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// Stats آمار کلی ربات.
type Stats struct {
	TotalUsers int64
	TotalCodes int64
	TotalFiles int64
	TodayUsers int64
	ActiveSubs int64
}

func (s *Store) GetStats(ctx context.Context) Stats {
	var st Stats
	var err error
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	st.TotalUsers, err = s.col(colUsers).CountDocuments(ctx, s.f())
	s.logErr("GetStats: total users", err)
	st.TotalCodes, err = s.col(colCodes).CountDocuments(ctx, s.f())
	s.logErr("GetStats: total codes", err)
	st.TotalFiles, err = s.col(colFiles).CountDocuments(ctx, s.f())
	s.logErr("GetStats: total files", err)
	st.TodayUsers, err = s.col(colUsers).CountDocuments(ctx,
		s.f(bson.E{Key: "created_at", Value: bson.D{{Key: "$gte", Value: today}}}))
	s.logErr("GetStats: today users", err)
	st.ActiveSubs, err = s.col(colUsers).CountDocuments(ctx,
		s.f(bson.E{Key: "sub_expires_at", Value: bson.D{{Key: "$gt", Value: now}}}))
	s.logErr("GetStats: active subs", err)
	return st
}
