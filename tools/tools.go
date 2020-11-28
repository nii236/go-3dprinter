// +build tools

package totalex

//go:generate go build -o ../bin/migrate -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate
//go:generate go build -o ../bin/sqlboiler github.com/volatiletech/sqlboiler/v4
//go:generate go build -o ../bin/sqlboiler-psql github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-psql
//go:generate go build -o ../bin/air github.com/cosmtrek/air

import (
	_ "github.com/cosmtrek/air"
	_ "github.com/golang-migrate/migrate"
	_ "github.com/mattn/goreman"
	_ "github.com/volatiletech/sqlboiler/v4"
	_ "github.com/volatiletech/sqlboiler/v4/drivers/sqlboiler-psql"
)
