package etcd

import (
	"github.com/coreos/etcd/client"
	"log"
	"golang.org/x/net/context"
	"time"
	"fmt"
	"strings"
)

const (
	root = "/workflow/"
	timeout = 3 * time.Second
)

func formatKey(id, key string) string {
	if strings.HasPrefix(key, "/") {
		return fmt.Sprint("%s%s%s", root, id, key)
	} else {
		return fmt.Sprint("%s%s/%s", root, id, key)
	}
}

func keysAPI(endpoints []string) client.KeysAPI {
	cfg := client.Config{
		Endpoints:               endpoints,
		Transport:               client.DefaultTransport,
		HeaderTimeoutPerRequest: timeout,
	}
	c, err := client.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	return client.NewKeysAPI(c)
}

func Create(endpoints []string, id, key, value string) {
	kapi := keysAPI(endpoints)
	key = formatKey(id, key)

	resp, err := kapi.Create(context.Background(), key, value)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Set is done. Metadata is %q\n", resp)
	}

}

func Get(endpoints []string, id, key string) string {
	kapi := keysAPI(endpoints)
	key = formatKey(id, key)

	resp, err := kapi.Get(context.Background(), key, nil)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("%q key has %q value\n", resp.Node.Key, resp.Node.Value)
	}
	return resp.Node.Value
}

func Delete(endpoints []string, id, key string) {
	kapi := keysAPI(endpoints)
	key = formatKey(id, key)

	resp, err := kapi.Delete(context.Background(), key, nil)
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("Delete is done. Metadata is %q\n", resp)
	}
}



