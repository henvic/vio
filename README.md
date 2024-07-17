# IP lookup

[![GoDoc](https://godoc.org/github.com/henvic/vio?status.svg)](https://godoc.org/github.com/henvic/vio) [![Build Status](https://github.com/henvic/vio/workflows/Integration/badge.svg)](https://github.com/henvic/vio/actions?query=workflow%3AIntegration) [![Coverage Status](https://coveralls.io/repos/henvic/vio/badge.svg)](https://coveralls.io/r/henvic/vio)

## Requirements
```shell
$ go install github.com/jackc/tern/v2@latest # v2.2.1
$ go install go.uber.org/mock/mockgen@latest # v0.4.0
```


## Environment variables

| Environment Variable             | Description                                                                     |
| -------------------------------- | ------------------------------------------------------------------------------- |
| PostgreSQL environment variables | Please check https://www.postgresql.org/docs/current/libpq-envars.html          |
| INTEGRATION_TESTDB               | When running go test, database tests will only run if `INTEGRATION_TESTDB=true` |


## Testing
On a machine with access to PostgreSQL, copy data_dump.csv to the root of the project and then continue with the following instructions.

The PostgreSQL configuration works via environment variables:
https://www.postgresql.org/docs/current/libpq-envars.html

As long as you can use [psql](https://www.postgresql.org/docs/current/app-psql.html) on the current working directory of the project, you should be able to invoke the following commands.
The migration and the application will use the database currently set via the `PGDATABASE` environment variable.
The integration tests will create temporary databases on the server configured via environment variables.

To run application:

```sh
# Create a database
$ psql -c "CREATE DATABASE vio;"
# Set the environment variable PGDATABASE
$ export PGDATABASE=vio
# Run migrations
$ make migrate
# Execute application
$ make server
2021/11/22 07:21:21 HTTP server listening at localhost:8080
2021/11/22 07:21:21 gRPC server listening at 127.0.0.1:8082
```


```shell
# To populate the data, run
$ make import
# To check a value, run
$ curl -v "localhost:8080/v1/lookup?ip=127.0.0.1"
```

To run tests:

```sh
# Run all tests passing INTEGRATION_TESTDB explicitly
$ INTEGRATION_TESTDB=true make true
```
