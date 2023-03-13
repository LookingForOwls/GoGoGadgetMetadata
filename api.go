package main

import (
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"

	"github.com/julienschmidt/httprouter"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fmt.Fprint(w, "Metadata API!\n")
}

func Metadata(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	// Setup geth client
	client, err := ethclient.Dial(EnvConfigs.InfuraRPC)
	if err != nil {
		log.Fatalf("Failed to connect to the Ethereum client: %v", err)
	}
	// Convert tokenId in URL to Int64
	token, _ := strconv.ParseInt(ps.ByName("tokenId"), 0, 64)
	// Check if token minted
	if !Minted(client, token) {
		fmt.Fprintf(w, "Token %s Not Minted\n", ps.ByName("tokenId"))
		// If minted, serve json from MetadataDir
	} else {
		fileBytes, err := os.ReadFile(fmt.Sprintf("%s%s.json", EnvConfigs.MetadataDir, ps.ByName("tokenId")))
		if err != nil {
			fmt.Fprintf(w, "Metadata For Token: %s Not Found\n", ps.ByName("tokenId"))
		}

		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		w.Write(fileBytes)
	}

}

// Use infura to check if token has been minted using contracts onlyOwner() function.
func Minted(c *ethclient.Client, token int64) bool {
	address := common.HexToAddress(EnvConfigs.ContractAddress)
	instance, err := NewStorage(address, c)
	if err != nil {
		log.Fatalf("Failed to instantiate contract: %v", err)
	}

	owner, _ := instance.OwnerOf(&bind.CallOpts{}, big.NewInt(token))

	return owner != common.HexToAddress("0x0000000000000000000000000000000000000000")
}

func main() {
	InitEnvConfigs()

	router := httprouter.New()
	router.GET("/", Index)
	router.GET("/metadata/:tokenId", Metadata)

	log.Fatal(http.ListenAndServe(":8080", router))
}
