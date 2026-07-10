package store

import (
	"context"
	"errors"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/mrjvadi/creatorbot/uploader-bot/internal/models"
)

func (s *Store) GetOrCreateUser(ctx context.Context, tgID int64, username, firstName string) (*models.User, error) {
	u, err := s.GetUser(ctx, tgID)
	if err != nil {
		return nil, err
	}
	if u != nil {
		return u, nil
	}
	nu := &models.User{
		TelegramID: tgID,
		Username:   username,
		FirstName:  firstName,
	}
	nu.ID = newID()
	nu.InstanceID = s.instanceID
	nu.CreatedAt = time.Now()
	nu.UpdatedAt = time.Now()
	if _, err := s.col(colUsers).InsertOne(ctx, nu); err != nil {
		return nil, err
	}
	return nu, nil
}

func (s *Store) GetUser(ctx context.Context, tgID int64) (*models.User, error) {
	var u models.User
	err := s.col(colUsers).FindOne(ctx, s.f(bson.E{Key: "telegram_id", Value: tgID}), &u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) UpdateUser(ctx context.Context, u *models.User) error {
	return s.col(colUsers).UpdateOne(ctx,
		s.f(bson.E{Key: "_id", Value: u.ID}),
		set(bson.D{
			{Key: "username", Value: u.Username},
			{Key: "first_name", Value: u.FirstName},
			{Key: "is_blocked", Value: u.IsBlocked},
			{Key: "free_downloads", Value: u.FreeDownloads},
			{Key: "sub_expires_at", Value: u.SubExpiresAt},
			{Key: "sub_plan_id", Value: u.SubPlanID},
		}))
}

func (s *Store) BlockUser(ctx context.Context, tgID int64, block bool) error {
	return s.col(colUsers).UpdateOne(ctx,
		s.f(bson.E{Key: "telegram_id", Value: tgID}),
		set(bson.D{{Key: "is_blocked", Value: block}}))
}

func (s *Store) SetUserSub(ctx context.Context, tgID int64, planID string, days int) error {
	exp := time.Now().AddDate(0, 0, days)
	return s.col(colUsers).UpdateOne(ctx,
		s.f(bson.E{Key: "telegram_id", Value: tgID}),
		set(bson.D{
			{Key: "sub_plan_id", Value: planID},
			{Key: "sub_expires_at", Value: exp},
		}))
}

// ResetDownloadCounts شمارش دانلود رایگان همه‌ی کاربران را صفر می‌کند و
// لاگ‌های دانلود را پاک می‌کند. (UpdateMany در اینترفیس نیست؛ تک‌تک انجام می‌شود.)
func (s *Store) ResetDownloadCounts(ctx context.Context) error {
	users, _, err := s.ListUsers(ctx, 1, 1_000_000)
	if err != nil {
		return err
	}
	for _, u := range users {
		s.logErr("ResetDownloadCounts: reset user", s.col(colUsers).UpdateOne(ctx,
			s.f(bson.E{Key: "_id", Value: u.ID}),
			set(bson.D{{Key: "free_downloads", Value: 0}})))
	}
	// پاک‌کردن لاگ‌های دانلود
	var logs []models.DownloadLog
	if err := s.col(colDownloads).Find(ctx, s.f(), &logs); err != nil {
		s.logErr("ResetDownloadCounts: list logs", err)
	} else {
		for _, l := range logs {
			s.logErr("ResetDownloadCounts: delete log", s.col(colDownloads).DeleteOne(ctx, s.f(bson.E{Key: "_id", Value: l.ID})))
		}
	}
	return nil
}

func (s *Store) SearchUser(ctx context.Context, query string) (*models.User, error) {
	var filter bson.D
	if id, err := strconv.ParseInt(query, 10, 64); err == nil {
		filter = s.f(bson.E{Key: "telegram_id", Value: id})
	} else {
		uname := query
		if len(uname) > 0 && uname[0] == '@' {
			uname = uname[1:]
		}
		// escapeMongoRegex: ورودی ادمین را لغوی (literal) می‌کند تا از ReDoS
		// جلوگیری شود — همان تابعی که در code.go برای SearchCodes استفاده می‌شود.
		filter = s.f(bson.E{Key: "username", Value: primitive.Regex{Pattern: escapeMongoRegex(uname), Options: "i"}})
	}
	var u models.User
	err := s.col(colUsers).FindOne(ctx, filter, &u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) ListUsers(ctx context.Context, page, limit int) ([]models.User, int64, error) {
	total, err := s.col(colUsers).CountDocuments(ctx, s.f())
	s.logErr("ListUsers: count", err)
	if page < 1 {
		page = 1
	}
	var users []models.User
	err = s.col(colUsers).Find(ctx, s.f(), &users,
		ports_sortDesc("created_at"),
		ports_skip(int64((page-1)*limit)),
		ports_limit(int64(limit)),
	)
	return users, total, err
}
