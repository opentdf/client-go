package main

import (
	"encoding/json"
	"path/filepath"
	"io/ioutil"
	"fmt"
	"log"
	"os"

	opentdf "github.com/virtru/opentdf-client-go-wrapper"

	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewDevelopment() // or NewProduction, or NewDevelopment
	if err != nil {
		log.Fatal("Logger initialization failed!")
	}
	defer logger.Sync()

	argLength := len(os.Args[1:])
	if argLength != 1 {
		log.Fatal("This tool expects exactly one argument - a path to a TDF file")
	}

	filePath := os.Args[1]

	filePathAbs, err := filepath.Abs(filePath)
	if err != nil {
		log.Fatalf("Could not load TDF file from path: %s", filePath)
	}

	fileBytes, err := ioutil.ReadFile(filePathAbs)
	if err != nil {
		log.Fatalf("Could not read TDF file: %s", filePath)
	}

	prettyPrintPolicyJSON(fileBytes, logger.Sugar())
}

func prettyPrintPolicyJSON(data []byte, logger *zap.SugaredLogger) {

	tdfSDK := getTDFClient(logger.Desugar())
	defer tdfSDK.Close()

	logger.Debug("Calling TDF SDK to read attrs of encrypted payload")

	dataPolicy, err := tdfSDK.GetPolicyFromTDF(data)
	if err != nil {
		logger.Fatal("TDF SDK decrypt failed", zap.Error(err))
		return;
	}

	logger.Debugf("Got policy from TDF: %+v", dataPolicy)

    b, err := json.MarshalIndent(dataPolicy, "", "    ")
	if err != nil {
        logger.Fatalf("Error serializing TDF policy to JSON: %s", err)
        return;
    }


    fmt.Printf("Policy JSON: \n%s\n", string(b))
}

func getTDFClient(logger *zap.Logger) opentdf.TDFClient {
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

	return tdfSDK
}
