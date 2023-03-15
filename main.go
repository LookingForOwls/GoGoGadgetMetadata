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
var mintedStatus sync.Map

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Metadata API!\n")
}

func newWeb3(rpc string) (*web3, error) {
	c, err := ethclient.Dial(rpc)
	if err != nil {
		return nil, err
	}
	return &web3{
		client: c,
	}, nil
}

func (c *web3) metadataHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Convert tokenId in URL to Int64
	tokenID, err := strconv.ParseInt(ps.ByName("tokenId"), 0, 64)
	if err != nil {
		http.Error(w, "Invalid tokenId", http.StatusBadRequest)
		return
	}

	// Set default cache control for failures
	w.Header().Set("Cache-Control", "public, no-cache")

	// Check if token is minted if enabled
	checkowner, _ := strconv.ParseBool(os.Getenv("CHECK_OWNER"))

	if checkowner && !isMinted(c.client, tokenID) {
		http.Error(w, fmt.Sprintf("Token %d Not Minted\n", tokenID), http.StatusNotFound)
		return
	}

	// Check sync.Map to see if token mint status has been recorded
	if _, ok := mintedStatus.Load(tokenID); !ok {
		// If not in map, set token as minted in the map
		mintedStatus.Store(tokenID, true)
	}

	// Set cache control for successful responses
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	// Serve the metadata file
	fileBytes, err := os.ReadFile(fmt.Sprintf("metadata/%d", tokenID))
	if err != nil {
		http.Error(w, fmt.Sprintf("Metadata for token %d not found\n", tokenID), http.StatusNotFound)
		return
	}
	w.Write(fileBytes)
}

// Use infura to check if token has been minted using contracts onlyOwner() function.
func isMinted(c *ethclient.Client, tokenID int64) bool {
	contractAddress := getContractAddress()
	instance, err := NewStorage(contractAddress, c)
	if err != nil {
		log.Printf("Failed to instantiate contract: %v", err)
		return false
	}

	owner, _ := instance.OwnerOf(&bind.CallOpts{}, big.NewInt(tokenID))

	return owner != common.HexToAddress("0x0000000000000000000000000000000000000000")
}

func getContractAddress() common.Address {
	address := os.Getenv("CONTRACT_ADDRESS")
	if address == "" {
		log.Fatal("$CONTRACT_ADDRESS must be set")
	}
	return common.HexToAddress(address)
}

func main() {
	port := os.Getenv("PORT")
	infura := os.Getenv("INFURA")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	if infura == "" {
		log.Fatal("$INFURA must be set")
	}

	w, _ := newWeb3(infura)

	router := httprouter.New()
	router.GET("/", indexHandler)
	router.GET("/metadata/:tokenId", w.metadataHandler)

	log.Fatal(http.ListenAndServe(":"+port, router))
}
