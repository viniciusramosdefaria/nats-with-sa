package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

const (
	NatsHost      = "NATS_ADDR"
	NatsUser      = "NATS_USER"
	natsTokenPath = "/var/run/secrets/nats.io/token"
)

var (
	natsHost = os.Getenv(NatsHost)
	natsUser = os.Getenv(NatsUser)
	subj     = "foo.bar"
	payload  = "All is Well"
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

	wg := sync.WaitGroup{}
	wg.Add(3)

	go request(nc, &wg)
	go subscribe(nc, &wg)

	wg.Wait()

}

func subscribe(nc *nats.Conn, wg *sync.WaitGroup) {
	log.Println("Consuming nats queue")

	if _, err := nc.QueueSubscribe(subj, "worker", func(m *nats.Msg) {
		log.Println(string(m.Data))
		m.Respond([]byte("OK"))
		wg.Done()

	}); err != nil {
		log.Fatal(err)
	}

}

func request(nc *nats.Conn, wg *sync.WaitGroup) {
	log.Println("Requesting to nats server")

	msg, err := nc.Request(subj, []byte(payload), 5*time.Second)
	if err != nil {
		if nc.LastError() != nil {
			log.Fatalf("%v for request", nc.LastError())
		}
		log.Fatalf("%v for request", err)
	} else {
		log.Printf("Published [%s] : '%s'", subj, payload)
		log.Printf("Received  [%v] : '%s'", msg.Subject, string(msg.Data))
	}
	wg.Done()
}
