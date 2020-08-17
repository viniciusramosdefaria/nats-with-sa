build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags='-s -w' -o ./nats-test -v main.go
	docker build -t viniciusramosdefaria/nats-test:latest .
	docker push viniciusramosdefaria/nats-test:latest
	kubectl -n prod delete pod -l app=nats-test
	sleep 3
	kubectl -n prod logs -l app=nats-test