struct Data {
  unsigned long dataSize;
  unsigned char* buffer;
} EncryptionData;

int encryptBytes(unsigned char const *data, const unsigned long in_len, unsigned char **encrypted_data, unsigned long *encrypted_len);