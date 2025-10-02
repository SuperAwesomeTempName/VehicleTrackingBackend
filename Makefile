.DEFAULT_GOAL := help

# Add the following 'help' target to your Makefile
# And add help text after each target name starting with '\#\#'
help:			## Show this help
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #
tidy:			## format code and tidy modfile
	go fmt ./...
	go mod tidy -v

audit:			## run quality control checks
	go mod verify
	go vet ./...
	go test -race -buildvcs -vet=off ./...

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #
run:			## run the application
	go run main.go

test:			## run tests
	go test -v ./...

# ==================================================================================== #
# BUILD
# ==================================================================================== #
build:			## build the application
	go build -ldflags='-s' -o=./bin/api main.go

clean:			## clean build artifacts
	rm -rf ./bin

# ==================================================================================== #
# DOCKER CONTAINER
# ==================================================================================== #
docker-build:		## build docker image
	docker build -t vehicletrackingbackend .

docker-run:		## run docker container
	docker run -p 8080:8080 vehicletrackingbackend

up:			## docker-compose up
	docker-compose up -d

up-build:		## docker-compose up --build
	docker-compose up -d --build

down:			## docker-compose down
	docker-compose down