module github.com/mrjvadi/creatorbot/agentmanager

go 1.22

require (
	github.com/mrjvadi/creatorbot/shared v0.0.0
	github.com/mrjvadi/creatorbot/shared-core v0.0.0
	github.com/nats-io/nats.go v1.37.0
)

replace (
	github.com/mrjvadi/creatorbot/shared      => ../shared
	github.com/mrjvadi/creatorbot/shared-core => ../shared-core
)
