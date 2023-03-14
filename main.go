package main

import (
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/julienschmidt/httprouter"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

type web3 struct {
	client *ethclient.Client
}

// Map for token minted status
var sm sync.Map

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Metadata API!\n")
}

func NewWeb3(rpc string) (*web3, error) {
	c, err := ethclient.Dial(rpc)
	if err != nil {
		return nil, err
	}
	return &web3{
		client: c,
	}, nil
}

func (c *web3) Metadata(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Convert tokenId in URL to Int64
	token, _ := strconv.ParseInt(ps.ByName("tokenId"), 0, 64)
	if !Minted(c.client, token) {
		fmt.Fprintf(w, "Token %s Not Minted\n", ps.ByName("tokenId"))
		return
	}
	// // Check sync.Map to see if token mint status has been recorded
	// _, ok := sm.Load(token)
	// // If token set as minted in map skip web3 call.
	// if !ok {
	// 	// If not in map check if token minted
	// 	if !Minted(c.client, token) {
	// 		fmt.Fprintf(w, "Token %s Not Minted\n", ps.ByName("tokenId"))
	// 		return
	// 	}
	// 	sm.Store(token, true)
	// }

	fileBytes, err := os.ReadFile(fmt.Sprintf("%s%s.json", "metadata/", ps.ByName("tokenId")))
	if err != nil {
		fmt.Fprintf(w, "Metadata For Token: %s Not Found\n", ps.ByName("tokenId"))
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.Write(fileBytes)
}

// Use infura to check if token has been minted using contracts onlyOwner() function.
func Minted(c *ethclient.Client, token int64) bool {
	address := common.HexToAddress(os.Getenv("CONTRACT_ADDRESS"))
	instance, err := NewStorage(address, c)
	if err != nil {
		log.Fatalf("Failed to instantiate contract: %v", err)
	}

	owner, _ := instance.OwnerOf(&bind.CallOpts{}, big.NewInt(token))

	return owner != common.HexToAddress("0x0000000000000000000000000000000000000000")
}

func main() {
	port := os.Getenv("PORT")
	infura := os.Getenv("INFURA")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	w, _ := NewWeb3(infura)

	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/metadata/:tokenId", w.Metadata)

	log.Fatal(http.ListenAndServe(":"+port, router))
}
