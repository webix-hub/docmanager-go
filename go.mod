module wfs-ls

go 1.13

require (
	github.com/dhowden/tag v0.0.0-20191122115059-7e5c04feccd8
	github.com/disintegration/imaging v1.6.2
	github.com/go-chi/chi v4.0.3+incompatible
	github.com/go-chi/cors v1.0.0
	github.com/go-sql-driver/mysql v1.5.0
	github.com/golang-migrate/migrate/v4 v4.10.0
	github.com/jinzhu/configor v1.1.1
	github.com/jmoiron/sqlx v1.2.0
	github.com/rwcarlsen/goexif v0.0.0-20190401172101-9e8deecbddbd
	github.com/sergi/go-diff v0.0.0-00010101000000-000000000000
	github.com/unrolled/render v1.0.2
	github.com/xbsoftware/wfs v0.0.0-20200304115134-e605d8615890
	github.com/xbsoftware/wfs-db v0.0.0-20200304161452-662f70426b5e
	golang.org/x/net v0.0.0-20200319234117-63522dbf7eec // indirect
)

replace github.com/sergi/go-diff => github.com/kullias/go-diff v1.1.1-0.20200701111408-ad5742b32d97
