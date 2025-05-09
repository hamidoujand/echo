run:
	go run cmd/service/main.go 

tidy:
	go mod tidy 
	go mod vendor 

upgrade:
	go get -u -v ./... 
	go mod tidy
	go mod vendor


client:
	go run cmd/client/main.go 

###############################################################################
# Docker 
pull: 
	docker pull nats:2.10 

nats-up:
	docker run --name=nats -p 4222:4222 nats:2.10 -js 

nats-down:
	docker stop nats 
	docker rm nats -v 