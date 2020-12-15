#include <iostream>
#include <sstream>
#include <string>
#include <array>
#include <string.h>

// Only interface required.
#include "virtru_client.h"
#include "tdf3_constants.h"

using namespace virtru;

extern "C" {

int encryptBytes(unsigned char const *data, const unsigned long in_len, unsigned char **output, unsigned long *out_len) {
    *output = NULL;
    *out_len = NULL;

    try {
        auto user = std::getenv("VIRTRU_SDK_USER");
        auto appId = std::getenv("VIRTRU_SDK_APP_ID");

        if (user == NULL) {
            std::cerr << "Error retrieving user from 'VIRTRU_SDK_USER'" << std::endl;
            return 1;
        }

        if (appId == NULL) {
            std::cerr << "Error retrieving appId from 'VIRTRU_SDK_APP_ID'" << std::endl;
            return 1;
        }

        unsigned int offset = 0;
        auto encryptSourceCB = [&offset, &data, &in_len](virtru::Status& status)->BufferSpan {
            if (offset < in_len) {
                status = Status::Success;
                auto bytes_left = in_len - offset;
                auto old_offset = offset;
                offset += std::min(16ul, bytes_left);

                return { data + old_offset,  offset - old_offset};
            } else if (offset >= in_len) {
                status = Status::Success;
                return { data + offset, 0 };
            } else {
                status = virtru::Status::Failure;
                return { nullptr, 0 };
            }
        };

        unsigned long outputOffset = 0;
        unsigned long buf_size = in_len;
        auto outputBuf = new uint8_t[buf_size];

        auto encryptSinkCB = [&outputBuf, &outputOffset, &buf_size](BufferSpan bufferSpan) {
            if (outputOffset + bufferSpan.dataLength > buf_size) {
                auto new_buf_size = std::max(2 * buf_size, buf_size + bufferSpan.dataLength);
                auto new_buf = new uint8_t[new_buf_size];
                memcpy(new_buf, outputBuf, outputOffset);
                free(outputBuf);
                outputBuf = new_buf;
                buf_size = new_buf_size;
            }

            memcpy(outputBuf + outputOffset, bufferSpan.data, bufferSpan.dataLength);
            outputOffset += bufferSpan.dataLength;

            return virtru::Status::Success;
        };

        // Create an instance of the Virtru client.
        Client client {user, appId};

        EncryptDataParams encryptParams{encryptSourceCB, encryptSinkCB};

        // Encrypt the plaintext
        auto policyId = client.encryptData(encryptParams);

        unsigned long int decryptOffset = 0;
        auto decryptSourceCB = [&outputBuf, &decryptOffset, &outputOffset](virtru::Status& status)->BufferSpan {
            if (decryptOffset < outputOffset) {
                auto bytes_left = outputOffset - decryptOffset;
                auto old_offset = decryptOffset;
                decryptOffset += std::min(16ul, bytes_left);

                status = virtru::Status::Success;

                return { outputBuf + old_offset, decryptOffset - old_offset };
            } else if (decryptOffset >= outputOffset) {
                status = Status::Success;
                return { outputBuf + decryptOffset, 0 };
            } else {
                status = virtru::Status::Failure;
                return { nullptr, 0 };
            }
        };

        auto decryptBuf = new uint8_t[2 * in_len];
        unsigned long dOffset = 0;

        auto decryptSinkCB = [&decryptBuf, &dOffset, &in_len](BufferSpan bufferSpan) {
            if (dOffset + bufferSpan.dataLength >= (in_len + 1)) {
                std::cerr << "this is why we are failing " << in_len << " " << dOffset << bufferSpan.dataLength << std::endl;
                return virtru::Status::Failure;
            }

            memcpy(decryptBuf + dOffset, bufferSpan.data, bufferSpan.dataLength);
            dOffset += bufferSpan.dataLength;

            return virtru::Status::Success;
        };

        client.decryptData(decryptSourceCB, decryptSinkCB);

        if (memcmp(data, decryptBuf, in_len) == 0) {
            std::cout << "Encrypt and decrypt data api operations completed successfully" << std::endl;
        } else {
            std::cout << "Encrypt and decrypt data api operations failed" << std::endl;
        }

        *output = outputBuf;
        *out_len = outputOffset;

        // NOTE: Policy updates operations can be run on policyId.
        std::cout << "WE did it " << outputOffset << std::endl;

        return 0;
    }
    catch (const std::exception& exception) {
        std::cerr << "Exception " << exception.what() << std::endl;

        return 1;
    }
}

}