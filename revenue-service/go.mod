module github.com/mrjvadi/creatorbot/revenue-service

go 1.22

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/mrjvadi/creatorbot/shared v0.0.0
	github.com/nats-io/nats.go v1.37.0
	gorm.io/driver/postgres v1.5.9
	gorm.io/gorm v1.25.10
)

replace github.com/mrjvadi/creatorbot/shared => ../shared
