package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"golang.org/x/time/rate"
)

type web3 struct {
	client *ethclient.Client
}

// Rate limit requests 10 per second
var limiter = rate.NewLimiter(20, 3)

// Total number of minted tokens
var totalSupply int64

func main() {
	port := os.Getenv("PORT")
	infura := os.Getenv("INFURA")
	address := os.Getenv("CONTRACT_ADDRESS")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	if infura == "" {
		log.Fatal("$INFURA must be set")
	}

	if address == "" {
		log.Fatal("$CONTRACT_ADDRESS must be set")
	}

	if os.Getenv("CHECK_MINTED") == "" {
		log.Fatal("$CHECK_MINTED must be set")
	}

	// Create web3 client
	w, _ := newWeb3(infura)

	// instantiate contract
	instance, err := NewStorage(common.HexToAddress(address), w.client)
	if err != nil {
		log.Fatalf("Failed to instantiate contract: %v", err)
	}

	// Update total token supply every second
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	go updateTotalSupplyTicker(ticker, instance)

	// Start httprouter
	router := httprouter.New()
	router.GET("/", indexHandler)
	router.GET("/metadata/:tokenId", metadataHandler)

	log.Fatal(http.ListenAndServe(":"+port, router))
}

func indexHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Metadata API!\n")
}

func metadataHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	checkMinted, _ := strconv.ParseBool(os.Getenv("CHECK_MINTED"))
	// Set default cache control for failures
	w.Header().Set("Cache-Control", "public, no-cache")
	// Convert tokenId in URL to Int64
	tokenID, err := strconv.ParseInt(ps.ByName("tokenId"), 0, 64)
	if err != nil {
		http.Error(w, "Invalid tokenId", http.StatusBadRequest)
		return
	}

	// If tokenId < totalSupply() return metadata
	if !checkMinted || tokenID <= totalSupply {
		serveMetadata(w, tokenID)
		return
	}

	// return error if not minted
	http.Error(w, fmt.Sprintf("Token %d Not Minted\n", tokenID), http.StatusNotFound)
	return
}

func serveMetadata(w http.ResponseWriter, tokenID int64) {
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
	return
}

// Create new Web3 RPC Client
func newWeb3(rpc string) (*web3, error) {
	c, err := ethclient.Dial(rpc)
	if err != nil {
		return nil, err
	}
	return &web3{
		client: c,
	}, nil
}

// Query contract for totalSupply() and update global variable
func updateTotalSupplyTicker(ticker *time.Ticker, i *Storage) {
	for {
		select {
		case <-ticker.C:
			supply, err := i.TotalSupply(&bind.CallOpts{})
			if err != nil {
				log.Printf("RPC Error: %v", err)
				continue // Continue with the next iteration of the loop if there is an error
			}
			totalSupply = supply.Int64()
		}
	}
}
