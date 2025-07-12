n:
	# For myself
	nvim server.go yana/minio.go yana/postgres.go yana/yanaErrors.go

run:
	go run server.go

# For myself too
r: run

install:
	go install
