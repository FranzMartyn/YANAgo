# YANAgo

[//]: <> (Maybe add an AI generated logo of YANAgo or something)

(WIP)

YANAgo is, as the name implies, **Y**et **A**nother **N**otes **A**pp - written in **go** and [labstack/echo](https://github.com/labstack/echo).

## Requirements

- Go
- [flosch/pongo2](https://github.com/flosch/pongo2)
- [google/uuid](https://github.com/google/uuid)
- [labstack/echo/v4](https://github.com/labstack/echo/v4)
- [lib/pq](https://github.com/lib/pq)
- [minio/minio-go/v7](https://github.com/minio/minio-go/v7)
- [yaml](https://gopkg.in/yaml.v3)

## Installation

```
git clone https://github.com/FranzMartyn/YANAgo
```

You need to have access to a [PostgreSQL](https://www.postgresql.org/docs/current/tutorial-install.html) and a [MinIO](https://min.io/docs/minio/linux/operations/installation.html) server.

Then edit the yaml files in `db/` with your data

Run `go run server.go` or `go run .` to test if everything is setup correctly

### PostgreSQL

You need to have a table called `note`:

```sql
CREATE TABLE note (
    id uuid NOT NULL,
    userid uuid NOT NULL,
    filename character varchar(255) NOT NULL
);
```

(If you're not using PostgreSQL, replace `uuid` with `varchar(36)`)

Also create a table `user_`:

```sql
CREATE TABLE user_ (
    id UUID NOT NULL,
    fullname TEXT NOT NULL,
    encryptedpassword TEXT NOT NULL,
    email citext
);
```

PostgreSQL documentation about [citext](https://www.postgresql.org/docs/current/citext.html)

To enable citext in PostgreSQL, run:

```sql
\c your_database_name;
CREATE EXTENSION IF NOT EXISTS citext;
```
