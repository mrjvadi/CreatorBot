module github.com/mrjvadi/creatorbot/botmanager

go 1.21

require (
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/mrjvadi/creatorbot/shared v0.0.0
	github.com/mrjvadi/creatorbot/shared-core v0.0.0
	github.com/nats-io/nats.go v1.37.0
	github.com/redis/go-redis/v9 v9.5.4
	gopkg.in/telebot.v4 v4.0.0-beta.4
)

replace (
	github.com/mrjvadi/creatorbot/shared      => ../shared
	github.com/mrjvadi/creatorbot/shared-core => ../shared-core
)
