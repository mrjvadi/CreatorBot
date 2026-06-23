module github.com/mrjvadi/creatorbot/webhook-gateway

go 1.21

require (
	github.com/gin-gonic/gin v1.10.0
	github.com/mrjvadi/creatorbot/shared v0.0.0
	github.com/nats-io/nats.go v1.37.0
)

replace github.com/mrjvadi/creatorbot/shared => ../shared
