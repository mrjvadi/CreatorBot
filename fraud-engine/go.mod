module github.com/mrjvadi/creatorbot/fraud-engine

go 1.22

require (
	github.com/prometheus/client_golang v1.19.0
	github.com/gin-gonic/gin v1.10.0
	github.com/mrjvadi/creatorbot/shared v0.0.0
	go.mongodb.org/mongo-driver v1.15.0
)

replace github.com/mrjvadi/creatorbot/shared => ../shared
