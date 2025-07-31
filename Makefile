
run:
	go run server.go

install:
	go install

# For myself
r: run

# For myself too
n:
	nvim server.go yana/minio.go yana/postgres.go yana/yanaErrors.go

