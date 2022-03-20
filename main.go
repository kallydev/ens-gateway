package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/kallydev/ens-gateway/contract/ens"
	"github.com/sirupsen/logrus"
)

const (
	RootDomain = ".localhost"
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:   true,
		FullTimestamp: true,
	})
}

// https://github.com/jgimeno/go-namehash, thank you jgimeno
func NameHash(name string) common.Hash {
	node := common.Hash{}
	if len(name) > 0 {
		labels := strings.Split(name, ".")
		for i := len(labels) - 1; i >= 0; i-- {
			labelSha := crypto.Keccak256Hash([]byte(labels[i]))
			node = crypto.Keccak256Hash(node.Bytes(), labelSha.Bytes())
		}
	}
	return node
}

func main() {
	// You can also use other public endpoints
	ethereumClient, err := ethclient.Dial("wss://mainnet.infura.io/ws/v3/your_token")
	if err != nil {
		logrus.Fatalln(err)
	}
	resolver, err := ens.NewResolver(ens.AddressResolver, ethereumClient)
	if err != nil {
		logrus.Fatalln(err)
	}
	http.HandleFunc("/", func(responseWriter http.ResponseWriter, request *http.Request) {
		defer func() {
			_ = request.Body.Close()
		}()
		node := NameHash(strings.ReplaceAll(request.Host, RootDomain, ""))
		value, err := resolver.Text(&bind.CallOpts{}, node, "url")
		if err != nil {
			logrus.Errorln(err)
			responseWriter.WriteHeader(http.StatusInternalServerError)
			return
		}
		if value == "" {
			responseWriter.WriteHeader(http.StatusNotFound)
			return
		}
		targetURL, err := url.Parse(value)
		if err != nil {
			logrus.Errorln(err)
			responseWriter.WriteHeader(http.StatusInternalServerError)
			return
		}
		request.Host = targetURL.Host
		reverseProxy := httputil.NewSingleHostReverseProxy(targetURL)
		reverseProxy.ServeHTTP(responseWriter, request)
	})
	if err := http.ListenAndServe(":80", nil); err != nil {
		logrus.Fatalln(err)
	}
}
