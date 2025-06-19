# YANAgo

[//]: <> (Maybe add an AI generated logo of YANAgo or something)

(WIP)

YANAgo is, as the name implies, **Y**et **A**nother **N**otes **A**pp - written in Go and [labstack/echo](https://github.com/labstack/echo).

## Installation

```
git clone https://github.com/FranzMartyn/YANAgo
```

You need to have access to a [PostgreSQL](https://www.postgresql.org/docs/current/tutorial-install.html) and a [MinIO](https://min.io/docs/minio/linux/operations/installation.html) server.

Then edit the yaml files in `db/` with your data

Run `go run server.go` or `go run .` to test if everything is setup correctly

### PostgreSQL

You need to have a table called `user_` with the following rows:

*WIP: Insert row data here. A picture might be enough*
