# YANAgo

YANAgo is, as the name implies, **Y**et **A**nother **N**otes **A**pp - written in **go** and [labstack/echo](https://github.com/labstack/echo).

## Requirements

- Go
- A [PostgreSQL server](https://www.postgresql.org/download/)
- A [MinIO server](https://min.io/docs/minio/linux/operations/installation.html)

## Installation

First run:

```bash
git clone https://github.com/FranzMartyn/YANAgo
```

Then edit the yaml files in `db/` with your data

Run `make install` to install the dependencies, then `make run` to test if everything works

### PostgreSQL

You need to have a table called `note`:

```sql
CREATE TABLE note (
    id UUID NOT NULL,
    bucketname UUID NOT NULL,
    filename VARCHAR(255) NOT NULL
    created_at_utc TIMESTAMP NOT NULL
);
```

> [!IMPORTANT]
> If your database system does not have a `UUID` type, replace `UUID` with `VARCHAR(36)` or an equivalent.
> 
> If your database system doesn't have `TIMESTAMP` too, replace `TIMESTAMP` with `TEXT` or an equivalent.

And a table `user_`:

```sql
CREATE TABLE user_ (
    id UUID NOT NULL,
    fullname TEXT NOT NULL,
    encryptedpassword TEXT NOT NULL,
    email CITEXT NOT NULL
);
```

> [!IMPORTANT]
> If your database systems doesn't have `CITEXT`, replace `CITEXT` with something like `TEXT COLLATE NOCASE` for SQLite, `VARCHAR(255) COLLATE utf8mb4_unicode_ci` for MySQL and MariaDB, or an equivalent that allows text to be stored case-insensitively.

PostgreSQL documentation about [citext](https://www.postgresql.org/docs/current/citext.html)

To enable citext in PostgreSQL, run:

```sql
\c your_database_name;
CREATE EXTENSION IF NOT EXISTS citext;
```
