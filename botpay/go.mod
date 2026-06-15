module github.com/mrjvadi/creatorbot/botpay

go 1.22

require (
	github.com/prometheus/client_golang v1.19.0
	github.com/gin-gonic/gin v1.10.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/google/uuid v1.6.0
	github.com/mrjvadi/creatorbot/shared v0.0.0
	github.com/nats-io/nats.go v1.37.0
	gopkg.in/telebot.v4 v4.0.0-beta.4
	gorm.io/driver/postgres v1.5.9
	modernc.org/sqlite v1.29.1
	gorm.io/gorm v1.25.10
)

replace github.com/mrjvadi/creatorbot/shared => ../shared
