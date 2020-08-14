package main

import (
	"fmt"
	"github.com/nats-io/nats.go"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

const (
	NatsHost      = "NATS_ADDR"
	NatsUser      = "NATS_USER"
	natsTokenPath = "/var/run/secrets/nats.io/token"
)

var (
	natsHost = os.Getenv(NatsHost)
	natsUser = os.Getenv(NatsUser)
)

func Token() (string, error) {
	token, err := ioutil.ReadFile(natsTokenPath)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

func main() {

	natsToken, err := Token()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("natsToken: %s\n", natsToken)

	log.Println("Connecting to nats server")

	nc, err := nats.Connect(fmt.Sprintf("nats://%s:%s@%s:4222", natsUser, natsToken, natsHost))
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	log.Println("Publishing to nats server")

	if err := nc.Publish("foo.bar", []byte("All is Well")); err != nil {
		log.Fatal(err)
	}

	log.Println("Published successfully")

	wg := sync.WaitGroup{}
	wg.Add(10)


	log.Println("Consuming nats queue")

	// Create a queue subscription on "updates" with queue name "workers"
	if _, err := nc.QueueSubscribe("foo.bar", "worker", func(m *nats.Msg) {

		log.Println(string(m.Data))
		wg.Done()

	}); err != nil {
		log.Fatal(err)
	}

	// Wait for messages to come in
	wg.Wait()

}
