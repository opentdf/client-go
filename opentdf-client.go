package opentdf

import (
	"encoding/json"
	"fmt"
	"unsafe"

	"go.uber.org/zap"
)

//The following directives tell `cgo` where to find the C static library and header files,
//and should be modified if those files are located elsewhere

// #cgo darwin LDFLAGS: -lopentdf_static_combined -lstdc++
// #cgo linux LDFLAGS: -lopentdf_static_combined -lpthread -ldl -lm -lstdc++
//
// #include <stdlib.h>
// #include <stdbool.h>
// #include <tdf_constants_c.h>
// #include <tdf_client_c.h>
//
// //These C helper functions are called from the below Go code -
// //passing string arrays to C requires a big of fiddling
// //Yes, these are real C functions in comments - do not delete them
//
// static char**makeCharArray(int size) {
//   return calloc(sizeof(char*), size);
// }
//
// static void setArrayString(char **a, char *s, int n) {
//   a[n] = s;
// }
//
// static void freeCharArray(char **a, int size) {
//   int i;
//   for (i = 0; i < size; i++)
//     free(a[i]);
//   free(a);
// }
import "C"

//NOTE: Everyone looking at or evaluating this code should first read
//https://dave.cheney.net/2016/01/18/cgo-is-not-go, which is full of extremely
//excellent criticism/gotchas around the use of `cgo`, from one of Go's lead language architects.
//Dave Cheney is 100% right about how the use of CGO makes C your lowest common denomiator and neuters most of
//Go's natural, modern benenfits. And yet, this exists as a stopgap.
type tdfCInterop struct {
	sdkPtr                C.TDFClientPtr
	credsPtr              C.TDFCredsPtr
	cStringPointersToFree []*C.char
	kasURL                string
	logger                *zap.SugaredLogger
}

//See https://github.com/virtru/tdf-spec/blob/master/schema/AttributeObject.md
// {
// "attribute": "https://example.com/attr/classification/value/topsecret"
// }
type TDFAttribute struct {
	Attribute string `json:"attribute"`
}

// See https://github.com/virtru/tdf-spec/blob/master/schema/PolicyObject.md
// {
//"uuid": "1111-2222-33333-44444-abddef-timestamp",
//"body": {
//    "dataAttributes": [<Attribute Object>],
//    "dissem": ["user-id@domain.com"]
//  },
//"tdf_spec_version:": "x.y.z"
//}
type TDFPolicy struct {
	UUID        string        `json:"uuid"`
	Body        TDFPolicyBody `json:"body"`
	SpecVersion string        `json:"tdf_spec_version"`
}

type TDFPolicyBody struct {
	DataAttributes    []TDFAttribute `json:"dataAttributes"`
	DisseminationList []string       `json:"dissem"`
}

type TDFClient interface {
	Close()
	EncryptFile(inFile, outFile string, dataAttribs []string) error
	EncryptString(data string, dataAttribs []string) ([]byte, error)
	DecryptTDF(data []byte) (string, error)
	GetPolicyFromTDF(data []byte) (*TDFPolicy, error)
}

func NewTDFClientOIDC(email string, orgName string, clientId string, clientSecret string, oidcURL string, kasURL string, logger *zap.Logger) TDFClient {
	cSDK := tdfCInterop{logger: logger.Sugar(), kasURL: kasURL}
	cSDK.initializeOIDCClient(C.CString(email), C.CString(orgName), C.CString(clientId), C.CString(clientSecret), C.CString(oidcURL), C.CString(kasURL))

	//If Zap logging level == debug, then make TDF SDK internal request logging very verbose
	if zapDebug := logger.Check(zap.DebugLevel, "debugging"); zapDebug != nil {
		cSDK.checkTDFStatus(C.TDFEnableConsoleLogging(cSDK.sdkPtr, C.TDFLogLevelDebug), "TDFEnableConsoleLogging")
	}
	return &cSDK
}

func NewTDFClientOIDCTokenExchange(email, orgName, clientId, clientSecret, externalAccessToken, oidcURL, kasURL string, logger *zap.Logger) TDFClient {
	cSDK := tdfCInterop{logger: logger.Sugar(), kasURL: kasURL}
	cSDK.initializeOIDCClientTokenExchange(C.CString(email), C.CString(orgName), C.CString(clientId), C.CString(clientSecret), C.CString(externalAccessToken), C.CString(oidcURL), C.CString(kasURL))

	//If Zap logging level == debug, then make TDF SDK internal request logging very verbose
	if zapDebug := logger.Check(zap.DebugLevel, "debugging"); zapDebug != nil {
		cSDK.checkTDFStatus(C.TDFEnableConsoleLogging(cSDK.sdkPtr, C.TDFLogLevelDebug), "TDFEnableConsoleLogging")
	}
	return &cSDK
}

// Note that callers MUST invoke Close() on the TDFClient
// when they're done with it, ideally via a `defer TDFClient.Close()`
// If this does not happen, memory manually allocated in the C memspace will not be freed
func (tdfsdk *tdfCInterop) Close() {
	C.TDFDestroyClient(tdfsdk.sdkPtr)
	C.TDFDestroyCredential(tdfsdk.credsPtr)

	for _, cstrPnter := range tdfsdk.cStringPointersToFree {
		C.free(unsafe.Pointer(cstrPnter))
	}
}

//EncryptString takes a Go string, a format (ZIP or HTML), and a policy object, and encrypts the string
func (tdfsdk *tdfCInterop) EncryptString(data string, dataAttribs []string) ([]byte, error) {
	return tdfsdk.encryptString(data, tdfsdk.kasURL, dataAttribs)
}

func (tdfsdk *tdfCInterop) DecryptTDF(data []byte) (string, error) {
	return tdfsdk.decryptBytes(data)
}

func (tdfsdk *tdfCInterop) GetPolicyFromTDF(data []byte) (*TDFPolicy, error) {
	var tdfPolicy TDFPolicy
	policyJSON, err := tdfsdk.getPolicyStringFromTDFBytes(data)
	if err != nil {
		tdfsdk.logger.Errorf("Error getting policy from TDF file! Error was %s", err)
		return nil, err
	}
	err = json.Unmarshal([]byte(policyJSON), &tdfPolicy)
	if err != nil {
		tdfsdk.logger.Errorf("Error parsing policy JSON string obtained from TDF file! Policy string was %s and error was %s", policyJSON, err)
		return nil, err
	}

	return &tdfPolicy, nil
}

func (tdfsdk *tdfCInterop) EncryptFile(inFile, outFile string, dataAttribs []string) error {
	return tdfsdk.encryptFile(inFile, outFile, tdfsdk.kasURL, dataAttribs)
}

func (tdfsdk *tdfCInterop) initializeOIDCClient(
	email *C.char,
	orgName *C.char,
	clientId *C.char,
	clientSecret *C.char,
	oidcURL *C.char,
	kasURL *C.char) {

	tdfsdk.credsPtr = C.TDFCreateCredentialClientCreds(oidcURL, clientId, clientSecret, orgName)
	if tdfsdk.credsPtr == nil {
		tdfsdk.logger.Fatal("Could not initialize TDF C SDK credential object!")
	}

	tdfsdk.logger.Info("Initializing TDF C SDK")
	tdfsdk.sdkPtr = C.TDFCreateClient(tdfsdk.credsPtr, kasURL)
	if tdfsdk.sdkPtr == nil {
		tdfsdk.logger.Fatal("Could not initialize TDF C SDK!")
	}

	tdfsdk.cStringPointersToFree = append(tdfsdk.cStringPointersToFree,
		email,
		orgName,
		clientId,
		clientSecret,
		oidcURL,
		kasURL)

	tdfsdk.logger.Debug("TDF C SDK initialized")
}

func (tdfsdk *tdfCInterop) initializeOIDCClientTokenExchange(
	email *C.char,
	orgName *C.char,
	clientId *C.char,
	clientSecret *C.char,
	externalAccessToken *C.char,
	oidcURL *C.char,
	kasURL *C.char) {

	tdfsdk.credsPtr = C.TDFCreateCredentialTokenExchange(oidcURL, clientId, clientSecret, externalAccessToken, orgName)
	if tdfsdk.credsPtr == nil {
		tdfsdk.logger.Fatal("Could not initialize TDF C SDK credential object!")
	}

	tdfsdk.logger.Info("Initializing TDF C SDK")
	tdfsdk.sdkPtr = C.TDFCreateClient(tdfsdk.credsPtr, kasURL)
	if tdfsdk.sdkPtr == nil {
		tdfsdk.logger.Fatal("Could not initialize TDF C SDK!")
	}

	tdfsdk.cStringPointersToFree = append(tdfsdk.cStringPointersToFree,
		email,
		orgName,
		clientId,
		clientSecret,
		externalAccessToken,
		oidcURL,
		kasURL)

	tdfsdk.logger.Debug("TDF C SDK initialized")
}

func (tdfsdk *tdfCInterop) encryptFile(inFilename, outFilename, kasURL string, dataAttribs []string) error {
	inFile := C.CString(inFilename)
	outFile := C.CString(outFilename)
	defer C.free(unsafe.Pointer(inFile))
	defer C.free(unsafe.Pointer(outFile))

	storagePtr := C.TDFCreateTDFStorageFileType(inFile)
	if storagePtr == nil {
		tdfsdk.logger.Fatal("Could not initialize TDF C SDK TDF storage object!")
	}
	defer C.TDFDestroyStorage(storagePtr)

	thingsToFree := tdfsdk.addDataAttributes(dataAttribs, kasURL)
	//Free up resources created in above func after encrypt happens
	defer func() {
		for _, f := range thingsToFree {
			f()
		}
	}()

	err := tdfsdk.checkTDFStatus(C.TDFEncryptFile(tdfsdk.sdkPtr, storagePtr, outFile), "TDFEncryptFile")
	if err != nil {
		tdfsdk.logger.Errorf("Error encrypting file!")
		return err
	}

	return nil
}

func (tdfsdk *tdfCInterop) encryptString(data, kasURL string, dataAttribs []string) ([]byte, error) {
	inData := []byte(data)

	inSize, inPtr := convertGoBufToCBuf(inData)

	storagePtr := C.TDFCreateTDFStorageStringType(inPtr, (C.uint)(inSize))
	if storagePtr == nil {
		tdfsdk.logger.Fatal("Could not initialize TDF C SDK TDF storage object!")
	}
	defer C.TDFDestroyStorage(storagePtr)

	thingsToFree := tdfsdk.addDataAttributes(dataAttribs, kasURL)
	//Free up resources created in above func after encrypt happens
	defer func() {
		for _, f := range thingsToFree {
			f()
		}
	}()

	var outPtr C.TDFBytesPtr
	var outSize C.TDFBytesLength
	err := tdfsdk.checkTDFStatus(C.TDFEncryptString(tdfsdk.sdkPtr, storagePtr, &outPtr, &outSize), "TDFEncryptString")
	if err != nil {
		tdfsdk.logger.Errorf("Error encrypting string! Error was %s", err)
		return nil, err
	}

	outLen := C.int(C.uint(outSize))
	strBuf := C.GoBytes(unsafe.Pointer(outPtr), outLen)
	//GoBytes copies data from C memspace to Go memspace, so we're free to free the
	//C memspace here
	C.free(unsafe.Pointer(outPtr))
	tdfsdk.logger.Debugf("Got buffer %s with length %d", string(strBuf), C.int(outLen))
	return strBuf, nil
}

func (tdfsdk *tdfCInterop) addDataAttributes(dataAttrs []string, kasURL string) []func() {
	kasEndpoint := C.CString(kasURL)
	defer C.free(unsafe.Pointer(kasEndpoint))

	var thingsToFree []func()
	for _, dataAttr := range dataAttrs {
		attr := C.CString(dataAttr)
		thingsToFree = append(thingsToFree, func() { C.free(unsafe.Pointer(attr)) })
		tdfsdk.checkTDFStatus(C.TDFAddDataAttribute(tdfsdk.sdkPtr, attr, kasEndpoint), "TDFAddDataAttribute")
	}

	return thingsToFree
}

func (tdfsdk *tdfCInterop) decryptBytes(data []byte) (string, error) {

	size, ptr := convertGoBufToCBuf(data)

	var outPtr C.TDFBytesPtr
	var outSize C.TDFBytesLength
	err := tdfsdk.checkTDFStatus(C.TDFDecryptString(tdfsdk.sdkPtr, ptr, (C.uint)(size), &outPtr, &outSize),
		"TDFDecryptString")
	if err != nil {
		tdfsdk.logger.Errorf("Error decrypting bytes!, error was %s", err)
		return "", err
	}

	outLen := C.int(C.uint(outSize))
	strBuf := C.GoBytes(unsafe.Pointer(outPtr), outLen)
	//GoBytes copies data from C memspace to Go memspace, so we're free to free the
	//C memspace here
	C.free(unsafe.Pointer(outPtr))
	decStr := string(strBuf)
	tdfsdk.logger.Debugf("Got buffer %s with length %d", decStr, C.int(outLen))
	return decStr, nil
}

func (tdfsdk *tdfCInterop) getPolicyStringFromTDFBytes(data []byte) (string, error) {

	size, ptr := convertGoBufToCBuf(data)

	var outPtr C.TDFBytesPtr
	var outSize C.TDFBytesLength
	err := tdfsdk.checkTDFStatus(C.TDFGetPolicy(tdfsdk.sdkPtr, ptr, (C.uint)(size), &outPtr, &outSize),
		"TDFGetPolicy")
	if err != nil {
		tdfsdk.logger.Errorf("Error getting policy string from TDF bytes!, error was %s", err)
		return "", err
	}

	outLen := C.int(C.uint(outSize))
	strBuf := C.GoBytes(unsafe.Pointer(outPtr), outLen)
	//GoBytes copies data from C memspace to Go memspace, so we're free to free the
	//C memspace here
	C.free(unsafe.Pointer(outPtr))
	decStr := string(strBuf)
	tdfsdk.logger.Debugf("Got policy string buffer %s with length %d", decStr, C.int(outLen))
	return decStr, nil
}

func convertGoBufToCBuf(buf []byte) (size C.uint, ptr *C.uchar) {
	var bufptr *byte
	if cap(buf) > 0 {
		bufptr = &(buf[:1][0])
	}
	return (C.uint)(len(buf)), (*C.uchar)(bufptr)
}

func (tdfsdk *tdfCInterop) checkTDFStatus(status C.TDF_STATUS, cFuncName string) error {
	if status == C.TDF_STATUS_SUCCESS {
		return nil
	} else if status == C.TDF_STATUS_INVALID_PARAMS {
		return fmt.Errorf("Bad param calling %s", cFuncName)
	} else {
		return fmt.Errorf("Something went horribly wrong calling: %s, got code %d", cFuncName, status)
	}
}
