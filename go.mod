module github.com/rvolosatovs/lorawan-codec

// Copy-paste from lorawan-stack
replace github.com/grpc-ecosystem/grpc-gateway => github.com/TheThingsIndustries/grpc-gateway v1.14.4-gogo
replace github.com/robertkrimen/otto => github.com/TheThingsIndustries/otto v0.0.0-20181129100957-6ddbbb60554a
replace github.com/blang/semver => github.com/blang/semver v0.0.0-20190414182527-1a9109f8c4a1
replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.1+incompatible
replace github.com/labstack/echo/v4 => github.com/labstack/echo/v4 v4.1.2
replace gopkg.in/DATA-DOG/go-sqlmock.v1 => gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0
replace github.com/russross/blackfriday => github.com/russross/blackfriday v1.5.2
replace github.com/nicksnyder/go-i18n => github.com/nicksnyder/go-i18n v1.10.0

go 1.14

require github.com/rvolosatovs/lorawan-stack/v3 v3.7.3
