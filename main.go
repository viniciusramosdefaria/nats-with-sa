package main

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	consulapi "github.com/hashicorp/consul/api"
)

const (
	ConsulHost              = "CONSUL_HTTP_ADDR"
	serviceAccountTokenPath = "/run/secrets/kubernetes.io/serviceaccount/token"
	consulAuthMethod        = "auth-method-consul-auth"
	natsTokenPath           = "/var/run/secrets/nats.io/token"
)

var (
	subj    = "foo.bar"
	payload = "All is Well"

	consulHost = os.Getenv(ConsulHost)
)

type Configs struct {
	Nats Nats `json: "nats"`
}

type Nats struct {
	Host string `json: "host"`
	User string `json: "user"`
}

func Token(path string) (string, error) {
	token, err := ioutil.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

func main() {
	consulToken, err := Token(serviceAccountTokenPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connecting to consul")
	consulctl, err := consulapi.NewClient(consulapi.DefaultConfig())
	if err != nil {
		panic(err)
	}

	acltoken, _, err := consulctl.ACL().Login(&consulapi.ACLLoginParams{
		AuthMethod:  consulAuthMethod,
		BearerToken: consulToken,
	}, nil)

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Reading configs...")

	kv := consulctl.KV()

	cfgBytes, _, err := kv.Get("/configs", &consulapi.QueryOptions{
		Namespace:  acltoken.Namespace,
		Datacenter: "dc1",
		Token:      acltoken.SecretID,
	})
	if err != nil {
		log.Fatal(err)
	}

	var cfg Configs
	if err := json.Unmarshal(cfgBytes.Value, &cfg); err != nil {
		log.Fatal(err)
	}

	log.Println(cfg)

	natsToken, err := Token(natsTokenPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("natsToken: %s\n", natsToken)

	log.Println("Connecting to nats server")

	user := cfg.Nats.User
	host := cfg.Nats.Host

	nc, err := nats.Connect(fmt.Sprintf("nats://%s:%s@%s:4222", user, natsToken, host))
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
