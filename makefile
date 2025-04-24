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
	