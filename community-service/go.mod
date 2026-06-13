module github.com/mrjvadi/creatorbot/community-service

go 1.22

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/google/uuid v1.6.0
	github.com/mrjvadi/creatorbot/shared v0.0.0
	go.mongodb.org/mongo-driver v1.15.0
	gorm.io/driver/postgres v1.5.9
	gorm.io/gorm v1.25.10
)

replace github.com/mrjvadi/creatorbot/shared => ../shared
