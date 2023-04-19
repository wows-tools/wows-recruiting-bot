module github.com/kakwa/wows-recruiting-bot

go 1.19

require (
	github.com/IceflowRE/go-wargaming/v3 v3.0.0
	github.com/pemistahl/lingua-go v1.3.1
	go.uber.org/zap v1.24.0
	golang.org/x/exp v0.0.0-20221106115401-f9659909a136
	gorm.io/driver/sqlite v1.4.4
	gorm.io/gorm v1.24.3
	moul.io/zapgorm2 v1.3.0
)

require (
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/mattn/go-sqlite3 v1.14.15 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/multierr v1.9.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)

replace github.com/IceflowRE/go-wargaming/v3 v3.0.0 => github.com/kakwa/go-wargaming/v3 v3.0.0-20230419210655-e7fd4c5725ae
