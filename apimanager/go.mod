module github.com/mrjvadi/creatorbot/apimanager

go 1.22

require (
	github.com/prometheus/client_golang v1.19.0
	github.com/gin-gonic/gin v1.10.0
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/mrjvadi/creatorbot/shared v0.0.0
	github.com/mrjvadi/creatorbot/shared-core v0.0.0
)

replace (
	github.com/mrjvadi/creatorbot/shared      => ../shared
	github.com/mrjvadi/creatorbot/shared-core => ../shared-core
)
