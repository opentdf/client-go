package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/opentdf/client-go"

	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment() // or NewProduction, or NewDevelopment
	if err != nil {
		log.Fatalf("Logger initialization failed!")
	}

	defer logger.Sync()

	//slammer(logger)
	sequentialOIDC(logger)

}

func sequentialOIDC(logger *zap.Logger) {
	var wg sync.WaitGroup
	user := os.Getenv("TDF_USER")
	clientId := os.Getenv("TDF_CLIENTID")
	clientSecret := os.Getenv("TDF_CLIENTSECRET")
	orgName := os.Getenv("TDF_ORGNAME")
	kasURL := os.Getenv("TDF_KAS_URL")
	idpURL := os.Getenv("TDF_OIDC_URL")
	externalToken := os.Getenv("TDF_EXTERNALTOKEN")

	var tdfSDK client.TDFClient

	if externalToken != "" {
		tdfSDK = client.NewTDFClientOIDCTokenExchange(user, orgName, clientId, clientSecret, externalToken, idpURL, kasURL, logger)
	} else {
		tdfSDK = client.NewTDFClientOIDC(user, orgName, clientId, clientSecret, idpURL, kasURL, logger)
	}

	for i := 1; i <= 1000; i++ {
		wg.Add(1)
		doRoundtrip(logger, i, &wg, tdfSDK)
	}
	tdfSDK.Close()
}

func doRoundtrip(logger *zap.Logger, iter int, wg *sync.WaitGroup, tdfSDK client.TDFClient) {
	defer wg.Done()

	msg, timeElapsed := track(fmt.Sprintf("encrypt #%d", iter))

	var dataAttr []string
	dataAttr = append(dataAttr,
		"https://example.com/attr/Classification/value/C",
		"https://example.com/attr/COI/value/PRF",
	)

	stringStore, _ := client.NewTDFStorageString("holla at ya boi")
	res, _ := tdfSDK.EncryptToString(stringStore, "<some-metadata>", dataAttr)
	logger.Sugar().Debugf("Got TDF encrypted payload %s", string(res))
	duration(msg, timeElapsed)

	time.Sleep(5 * time.Second)

	msg, timeElapsed = track(fmt.Sprintf("decrypt #%d", iter))
	resStore, _ := client.NewTDFStorageString(string(res))
	decRes, _ := tdfSDK.DecryptTDF(resStore)
	duration(msg, timeElapsed)
	fmt.Printf("Round trip decrypted: %s", decRes)
}

func track(msg string) (string, time.Time) {
	return msg, time.Now()
}

func duration(msg string, start time.Time) {
	log.Printf("\nOperation %v: %v\n", msg, time.Since(start))
}
