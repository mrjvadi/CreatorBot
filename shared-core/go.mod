module github.com/mrjvadi/creatorbot/shared-core

go 1.21

require (
	github.com/google/uuid v1.6.0
	github.com/mrjvadi/creatorbot/shared v0.0.0
	go.mongodb.org/mongo-driver v1.15.0
	gorm.io/gorm v1.25.10
)

replace github.com/mrjvadi/creatorbot/shared => ../shared
