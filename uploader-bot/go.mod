module github.com/mrjvadi/creatorbot/uploader-bot

go 1.22

require (
	github.com/google/uuid v1.6.0
	github.com/mrjvadi/creatorbot/shared v0.0.0
	github.com/mrjvadi/creatorbot/shared-core v0.0.0
	gopkg.in/telebot.v4 v4.0.0-beta.4
	gorm.io/gorm v1.25.10
)

// go.work resolves this locally — no need to publish shared.
replace (
	github.com/mrjvadi/creatorbot/shared      => ../shared
	github.com/mrjvadi/creatorbot/shared-core => ../shared-core
)
