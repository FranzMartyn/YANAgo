<div style="text-align: center;">
  <img src="your-image-url.png" alt="Alt text" width="300">
</div>

# YANAgo

YANAgo is, as the name implies, **Y**et **A**nother **N**otes **A**pp - written in **go** and [labstack/echo](https://github.com/labstack/echo).

## Requirements

- Go
- A [PostgreSQL server](https://www.postgresql.org/docs/current/tutorial-install.html)
- A [MinIO server](https://min.io/docs/minio/linux/operations/installation.html)

## Installation

```bash
git clone https://github.com/FranzMartyn/YANAgo
```

Then edit the yaml files in `db/` with your data

Run `make install` to install the dependencies, then `make run` to test if everything works

### PostgreSQL

You need to have a table called `note`:

```sql
CREATE TABLE note (
    id uuid NOT NULL,
    bucketname uuid NOT NULL,
    filename character varchar(255) NOT NULL
    created_at_utc TIMESTAMP
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
