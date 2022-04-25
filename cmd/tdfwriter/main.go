package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	opentdf "github.com/opentdf/client-go"

	"go.uber.org/zap"
)

func main() {

	var cliDataAttrs string
	var stringPayload string
	var outFile string

	logger, err := zap.NewDevelopment() // or NewProduction, or NewDevelopment
	if err != nil {
		log.Fatalf("Logger initialization failed!")
	}

	defer logger.Sync()

	flag.StringVar(&cliDataAttrs, "a", "https://example.com/attr/Classification/value/C,https://example.com/attr/COI/value/PRF", "Specify list of data attrs to be applied, separated by a comma")
	flag.StringVar(&stringPayload, "p", "holla at ya boi", "Specify string data to encrypt")
	flag.StringVar(&outFile, "o", "out.tdf", "Specify output filename")
	flag.Parse()

	dataAttrs := strings.Split(cliDataAttrs, ",")
	encryptTDF(logger, stringPayload, outFile, dataAttrs)

}

func encryptTDF(logger *zap.Logger, dataString, outPath string, dataAttr []string) {
	user := os.Getenv("TDF_USER")
	clientId := os.Getenv("TDF_CLIENTID")
	clientSecret := os.Getenv("TDF_CLIENTSECRET")
	orgName := os.Getenv("TDF_ORGNAME")
	kasURL := os.Getenv("TDF_KAS_URL")
	idpURL := os.Getenv("TDF_OIDC_URL")
	externalToken := os.Getenv("TDF_EXTERNALTOKEN")

	var tdfSDK opentdf.TDFClient

	if externalToken != "" {
		tdfSDK = opentdf.NewTDFClientOIDCTokenExchange(user, orgName, clientId, clientSecret, externalToken, idpURL, kasURL, logger)
	} else {
		tdfSDK = opentdf.NewTDFClientOIDC(user, orgName, clientId, clientSecret, idpURL, kasURL, logger)
	}

	res, _ := tdfSDK.EncryptString(dataString, dataAttr)
	logger.Sugar().Debugf("Got TDF encrypted payload %s", string(res))
	writeFile(outPath, string(res))

	//Decrypt as well, just to validate the flow/demo
	decRes, _ := tdfSDK.DecryptTDF(res)
	fmt.Printf("Round trip decrypted: %s", decRes)
	tdfSDK.Close()
}

func writeFile(outfilePath, tdfData string) {
	file, err := os.Create(outfilePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	_, err = file.WriteString(tdfData)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Wrote TDF to: %s", outfilePath)
}
