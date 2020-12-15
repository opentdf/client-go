package main

// #cgo CFLAGS: -I${SRCDIR}/wrapper -I${SRCDIR}/virtru-tdf3-lib-cpp/include
// #cgo LDFLAGS: -L${SRCDIR}/wrapper -lwrapper -L${SRCDIR}/virtru-tdf3-lib-cpp/lib -lvirtru_tdf3_static_combined
// #include "wrapper.h"
/*
#include <string.h>

int callEncrypt() {
	const char *data = "We want to encrypt thissssssss";
	const unsigned long data_len = strlen(data) + 1;
	char* encrypted;
	unsigned int encrypted_len;

	encryptBytes(data, data_len, &encrypted, &encrypted_len);
}
*/
import "C"

func main() {
	C.callEncrypt()
}
