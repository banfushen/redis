module github.com/go-redis/redis/extra/rediscmd/v8

go 1.15

replace github.com/banfushen/redis/v8 => ../..

require (
	github.com/banfushen/redis/v8 v8.11.4
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
)
