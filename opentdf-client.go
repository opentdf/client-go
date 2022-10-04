package client

import (
	"encoding/json"
	"errors"
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

// NOTE: Everyone looking at or evaluating this code should first read
// https://dave.cheney.net/2016/01/18/cgo-is-not-go, which is full of extremely
// excellent criticism/gotchas around the use of `cgo`, from one of Go's lead language architects.
// Dave Cheney is 100% right about how the use of CGO makes C your lowest common denomiator and neuters most of
// Go's natural, modern benenfits. And yet, this exists as a stopgap.
type tdfCInterop struct {
	sdkPtr                C.TDFClientPtr
	credsPtr              C.TDFCredsPtr
	cStringPointersToFree []*C.char
	kasURL                string
	logger                *zap.SugaredLogger
}

// See https://github.com/opentdf/spec/blob/master/schema/AttributeObject.md
// {
// "attribute": "https://example.com/attr/classification/value/topsecret"
// }
type TDFAttribute struct {
	Attribute string `json:"attribute"`
}

// See https://github.com/opentdf/spec/blob/master/schema/PolicyObject.md
// {
// "uuid": "1111-2222-33333-44444-abddef-timestamp",
//
//	"body": {
//	   "dataAttributes": [<Attribute Object>],
//	   "dissem": ["user-id@domain.com"]
//	 },
//
// "tdf_spec_version:": "x.y.z"
// }
type TDFPolicy struct {
	UUID        string        `json:"uuid"`
	Body        TDFPolicyBody `json:"body"`
	SpecVersion string        `json:"tdf_spec_version"`
}

type TDFPolicyBody struct {
	DataAttributes    []TDFAttribute `json:"dataAttributes"`
	DisseminationList []string       `json:"dissem"`
}

// Note that right now the client-cpp storage type only works for TDF INPUT data, not OUTPUT data.
// This will be added eventually
type TDFStorage struct {
	storagePtr   C.TDFStorageTypePtr
	thingsToFree []func()
}

type TDFClient interface {
	Close()
	EncryptToFile(data *TDFStorage, outFile, metadata string, dataAttribs []string) error
	EncryptToString(data *TDFStorage, metadata string, dataAttribs []string) ([]byte, error)
	GetEncryptedMetadata(data *TDFStorage) (string, error)
	DecryptTDF(data *TDFStorage) (string, error)
	DecryptTDFPartial(data *TDFStorage, offset, length uint32) (string, error)
	GetPolicyFromTDF(data *TDFStorage) (*TDFPolicy, error)
}

// Creates a new S3-based TDF storage object
func NewTDFStorageS3(s3url, awsAccessKeyID, awsSecretKey, awsRegion string) (*TDFStorage, error) {
	inS3Url := C.CString(s3url)
	inKeyId := C.CString(awsAccessKeyID)
	inSecretKey := C.CString(awsSecretKey)
	inRegion := C.CString(awsRegion)
	thingsToFree := []func(){
		func() { C.free(unsafe.Pointer(inS3Url)) },
		func() { C.free(unsafe.Pointer(inKeyId)) },
		func() { C.free(unsafe.Pointer(inSecretKey)) },
		func() { C.free(unsafe.Pointer(inRegion)) },
	}

	storagePtr := C.TDFCreateTDFStorageS3Type(inS3Url, inKeyId, inSecretKey, inRegion)
	if storagePtr == nil {
		return nil, errors.New("Could not initialize TDF C SDK TDF S3 storage object!")
	}
	storage := TDFStorage{storagePtr, thingsToFree}
	return &storage, nil
}

// Creates a new file-based TDF storage object
func NewTDFStorageFile(filepath string) (*TDFStorage, error) {

	inFile := C.CString(filepath)
	thingsToFree := []func(){func() { C.free(unsafe.Pointer(inFile)) }}

	storagePtr := C.TDFCreateTDFStorageFileType(inFile)
	if storagePtr == nil {
		return nil, errors.New("Could not initialize TDF C SDK TDF file storage object!")
	}
	storage := TDFStorage{storagePtr, thingsToFree}
	return &storage, nil
}

// Creates a new string-based TDF storage object
func NewTDFStorageString(data string) (*TDFStorage, error) {
	inData := []byte(data)
	inSize, inPtr := convertGoBufToCBuf(inData)

	storagePtr := C.TDFCreateTDFStorageStringType(inPtr, (C.uint)(inSize))
	if storagePtr == nil {
		return nil, errors.New("Could not initialize TDF C SDK TDF string storage object!")
	}
	storage := TDFStorage{storagePtr, nil}
	return &storage, nil
}

// Should be invoked by caller when it's done with the storage location.
// Note that callers MUST invoke Close() on the TDFStorage object
// when they're done with it, ideally via a `defer storage.Close()`
// If this does not happen, memory manually allocated in the C memspace will not be freed
func (storage *TDFStorage) Close() {
	C.TDFDestroyStorage(storage.storagePtr)
	//Free up resources (cstrings, etc) created as part of this storage object
	defer func() {
		for _, f := range storage.thingsToFree {
			f()
		}
	}()
}

// Creates a new TDF client that will use OIDC client secret credentials to authenticate.
func NewTDFClientOIDC(email string, orgName string, clientId string, clientSecret string, oidcURL string, kasURL string, logger *zap.Logger) TDFClient {
	cSDK := tdfCInterop{logger: logger.Sugar(), kasURL: kasURL}
	cSDK.initializeOIDCClient(C.CString(email), C.CString(orgName), C.CString(clientId), C.CString(clientSecret), C.CString(oidcURL), C.CString(kasURL))

	//If Zap logging level == debug, then make TDF SDK internal request logging very verbose
	if zapDebug := logger.Check(zap.DebugLevel, "debugging"); zapDebug != nil {
		cSDK.checkTDFStatus(C.TDFEnableConsoleLogging(cSDK.sdkPtr, C.TDFLogLevelDebug), "TDFEnableConsoleLogging")
	}
	return &cSDK
}

// Creates a new TDF client that will use OIDC token exchange credentials to authenticate.
func NewTDFClientOIDCTokenExchange(email, orgName, clientId, clientSecret, externalAccessToken, oidcURL, kasURL string, logger *zap.Logger) TDFClient {
	cSDK := tdfCInterop{logger: logger.Sugar(), kasURL: kasURL}
	cSDK.initializeOIDCClientTokenExchange(C.CString(email), C.CString(orgName), C.CString(clientId), C.CString(clientSecret), C.CString(externalAccessToken), C.CString(oidcURL), C.CString(kasURL))

	//If Zap logging level == debug, then make TDF SDK internal request logging very verbose
	if zapDebug := logger.Check(zap.DebugLevel, "debugging"); zapDebug != nil {
		cSDK.checkTDFStatus(C.TDFEnableConsoleLogging(cSDK.sdkPtr, C.TDFLogLevelDebug), "TDFEnableConsoleLogging")
	}
	return &cSDK
}

// Destroys the TDFClient instance.
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

// EncryptToString takes a TDFStorage object containing the plaintext data to encrypt, an (optional, can be empty) string of metadata,
// and a policy object, and encrypts the string + metadata with the policy, returning the encrypted string.
func (tdfsdk *tdfCInterop) EncryptToString(data *TDFStorage, metadata string, dataAttribs []string) ([]byte, error) {
	return tdfsdk.encryptToString(data, tdfsdk.kasURL, metadata, dataAttribs)
}

// DecryptTDF takes a a TDFStorage object containing encrypted TDF data, and decrypts the contents, returning the decrypted string.
func (tdfsdk *tdfCInterop) DecryptTDF(data *TDFStorage) (string, error) {
	return tdfsdk.decryptBytes(data)
}

// DecryptTDFPartial takes a a TDFStorage object containing encrypted TDF data, and decrypts the from the given (plaintext) byte range, returning the decrypted plaintext for that range.
func (tdfsdk *tdfCInterop) DecryptTDFPartial(data *TDFStorage, offset, length uint32) (string, error) {
	return tdfsdk.decryptPartialBytes(data, offset, length)
}

func (tdfsdk *tdfCInterop) GetEncryptedMetadata(data *TDFStorage) (string, error) {
	return tdfsdk.getEncryptedMetadataFromTDF(data)
}

func (tdfsdk *tdfCInterop) GetPolicyFromTDF(data *TDFStorage) (*TDFPolicy, error) {
	var tdfPolicy TDFPolicy
	policyJSON, err := tdfsdk.getPolicyStringFromTDF(data)
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

// EncryptToFile takes a TDFStorage object containing the plaintext data to encrypt, an (optional, can be empty) string of metadata, an output filename,
// and a policy object, and encrypts the string + metadata with the policy, writing the result to the provided
// output filename.
func (tdfsdk *tdfCInterop) EncryptToFile(data *TDFStorage, outFile, metadata string, dataAttribs []string) error {
	return tdfsdk.encryptToFile(data, outFile, tdfsdk.kasURL, metadata, dataAttribs)
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

func (tdfsdk *tdfCInterop) encryptToFile(data *TDFStorage, outFilename, kasURL, metadata string, dataAttribs []string) error {
	outFile := C.CString(outFilename)
	defer C.free(unsafe.Pointer(outFile))

	thingsToFree := tdfsdk.addDataAttributes(dataAttribs, kasURL)
	//Free up resources created in above func after encrypt happens
	defer func() {
		for _, f := range thingsToFree {
			f()
		}
	}()

	//Only bother to set metadata if we have any to set.
	if metadata != "" {
		inMetadata := []byte(metadata)
		inMetSize, inMetPtr := convertGoBufToCBuf(inMetadata)
		err := tdfsdk.checkTDFStatus(C.TDFSetEncryptedMetadata(tdfsdk.sdkPtr, inMetPtr, (C.uint)(inMetSize)), "TDFSetEncryptedMetadata")
		if err != nil {
			tdfsdk.logger.Errorf("Error setting encrypted metadata string before encrypt! Error was %s", err)
			return err
		}
	}

	err := tdfsdk.checkTDFStatus(C.TDFEncryptFile(tdfsdk.sdkPtr, data.storagePtr, outFile), "TDFEncryptFile")
	if err != nil {
		tdfsdk.logger.Errorf("Error encrypting file!")
		return err
	}

	return nil
}

func (tdfsdk *tdfCInterop) encryptToString(data *TDFStorage, kasURL, metadata string, dataAttribs []string) ([]byte, error) {
	thingsToFree := tdfsdk.addDataAttributes(dataAttribs, kasURL)
	//Free up resources created in above func after encrypt happens
	defer func() {
		for _, f := range thingsToFree {
			f()
		}
	}()

	//Only bother to set metadata if we have any to set.
	if metadata != "" {
		inMetadata := []byte(metadata)
		inMetSize, inMetPtr := convertGoBufToCBuf(inMetadata)
		err := tdfsdk.checkTDFStatus(C.TDFSetEncryptedMetadata(tdfsdk.sdkPtr, inMetPtr, (C.uint)(inMetSize)), "TDFSetEncryptedMetadata")
		if err != nil {
			tdfsdk.logger.Errorf("Error setting encrypted metadata string before encrypt! Error was %s", err)
			return nil, err
		}
	}

	var outPtr C.TDFBytesPtr
	var outSize C.TDFBytesLength
	err := tdfsdk.checkTDFStatus(C.TDFEncryptString(tdfsdk.sdkPtr, data.storagePtr, &outPtr, &outSize), "TDFEncryptString")
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

func (tdfsdk *tdfCInterop) decryptBytes(data *TDFStorage) (string, error) {
	var outPtr C.TDFBytesPtr
	var outSize C.TDFBytesLength
	err := tdfsdk.checkTDFStatus(C.TDFDecryptString(tdfsdk.sdkPtr, data.storagePtr, &outPtr, &outSize),
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

func (tdfsdk *tdfCInterop) decryptPartialBytes(data *TDFStorage, offset, length uint32) (string, error) {
	var outPtr C.TDFBytesPtr
	var outSize C.TDFBytesLength
	var offsetC C.TDFBytesLength
	var lengthC C.TDFBytesLength

	offsetC = C.uint(offset)
	lengthC = C.uint(length)

	err := tdfsdk.checkTDFStatus(C.TDFDecryptDataPartial(tdfsdk.sdkPtr, data.storagePtr, offsetC, lengthC, &outPtr, &outSize),
		"TDFDecryptDataPartial")
	if err != nil {
		tdfsdk.logger.Errorf("Error decrypting partial bytes!, error was %s", err)
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

func (tdfsdk *tdfCInterop) getEncryptedMetadataFromTDF(data *TDFStorage) (string, error) {
	var outPtr C.TDFBytesPtr
	var outSize C.TDFBytesLength

	err := tdfsdk.checkTDFStatus(C.TDFGetEncryptedMetadata(tdfsdk.sdkPtr, data.storagePtr, &outPtr, &outSize),
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
	tdfsdk.logger.Debugf("Got encrypted metadata string buffer %s with length %d", decStr, C.int(outLen))
	return decStr, nil
}

func (tdfsdk *tdfCInterop) getPolicyStringFromTDF(data *TDFStorage) (string, error) {

	var outPtr C.TDFBytesPtr
	var outSize C.TDFBytesLength

	err := tdfsdk.checkTDFStatus(C.TDFGetPolicy(tdfsdk.sdkPtr, data.storagePtr, &outPtr, &outSize),
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
