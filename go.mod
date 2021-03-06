module github.com/rvolosatovs/lorawan-codec

// Dependency of lorawan-stack.
replace gopkg.in/DATA-DOG/go-sqlmock.v1 => gopkg.in/DATA-DOG/go-sqlmock.v1 v1.3.0

// Dependency of lorawan-stack.
replace gocloud.dev => gocloud.dev v0.19.0

go 1.14

require (
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826
	go.thethings.network/lorawan-stack/v3 v3.8.5-0.20200701092401-336d059baf02
)
