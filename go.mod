module app

go 1.11

require (
	app/_generated/cmsfr/inventory v0.0.0
	app/_generated/cmsfr/vmaas v0.0.0
	github.com/DataDog/zstd v1.4.4 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/antihax/optional v1.0.0
	github.com/bitly/go-simplejson v0.5.0
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869
	github.com/deepmap/oapi-codegen v1.3.2 // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20191001013358-cfbb681360f0 // indirect
	github.com/frankban/quicktest v1.6.0 // indirect
	github.com/gin-contrib/gzip v0.0.1
	github.com/gin-gonic/gin v1.4.0
	github.com/go-sql-driver/mysql v1.4.1
	github.com/jinzhu/gorm v1.9.11
	github.com/jinzhu/now v1.1.1 // indirect
	github.com/json-iterator/go v1.1.8 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.2 // indirect
	github.com/lib/pq v1.2.0 // indirect
	github.com/pierrec/lz4 v2.3.0+incompatible // indirect
	github.com/prometheus/client_golang v1.2.1 // indirect
	github.com/satori/go.uuid v1.2.0
	github.com/segmentio/kafka-go v0.3.4
	github.com/sirupsen/logrus v1.4.2
	github.com/stretchr/testify v1.4.0
	github.com/swaggo/files v0.0.0-20190704085106-630677cd5c14
	github.com/swaggo/gin-swagger v1.2.0
	github.com/swaggo/swag v1.5.1
	github.com/ugorji/go v1.1.7 // indirect
	github.com/zsais/go-gin-prometheus v0.1.0
	golang.org/x/oauth2 v0.0.0-20191122200657-5d9234df094c
	google.golang.org/appengine v1.6.5 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
)

replace app/_generated/cmsfr/inventory v0.0.0 => ./_generated/cmsfr/inventory

replace app/_generated/cmsfr/vmaas v0.0.0 => ./_generated/cmsfr/vmaas

replace github.com/ugorji/go v1.1.4 => github.com/ugorji/go/codec v0.0.0-20190204201341-e444a5086c43
