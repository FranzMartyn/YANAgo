# YANAgo

**YANAgo** is, as the name implies, **Y**et **A**nother **N**otes **A**pp: A web app written in **go** and [labstack/echo](https://github.com/labstack/echo).

## Requirements

- Go
- A [PostgreSQL server](https://www.postgresql.org/download/)
- A [MinIO server](https://min.io/docs/minio/linux/operations/installation.html)

## Installation

First run:

```bash
git clone https://github.com/FranzMartyn/YANAgo
```

Then edit `config/postgresql.yml` and `config/minio.yml` with your data.

Run `make install` to install the dependencies, then `make run` to start the server.

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
> If your database system doesn't have a `UUID` type, replace `UUID` with `VARCHAR(36)` or an equivalent.
>
> If your database system doesn't have `TIMESTAMP`, replace `TIMESTAMP` with `TEXT` or an equivalent.

And a table `user_`:

```sql
CREATE TABLE user_ (
    id UUID NOT NULL,
    fullname TEXT NOT NULL,
    encryptedpassword TEXT NOT NULL,
    email CITEXT NOT NULL
);
```

PostgreSQL documentation about [citext](https://www.postgresql.org/docs/current/citext.html)

To enable citext in PostgreSQL, run:

```sql
\c your_database_name;
CREATE EXTENSION IF NOT EXISTS citext;
```

> [!IMPORTANT]
> If your database system doesn't have `CITEXT`, replace `CITEXT` with something that allows text to be stored case-insensitively. For example:
>
> - `TEXT COLLATE NOCASE` on SQLite
> - `VARCHAR(255) COLLATE utf8mb4_unicode_ci` on MySQL and MariaDB
> - ...
