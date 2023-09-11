///
/// Copyright (C) 2023 by Computantis
///
/// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without l> imitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
/// 
/// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
///
/// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
///
#ifndef SIGNATURE_H
#define SIGNATURE_H
#define KEY_LEN 32
#define SIGNATURE_LEN 64

#include <openssl/evp.h>
#include <stdbool.h>
#include <stddef.h>

/// 
/// Signature is an entity holding the signature and digest of the message.
///
typedef struct {
    unsigned char   *digest_buffer;
    unsigned char   *signature_buffer;
    size_t          digest_len;
    size_t          signature_len;
} Signature;

///
/// Signature_free frees the signature.
///
void Signature_free(Signature *sig);

///
/// Signature_verify verifies the signature for the given message.
/// It produces digest from given message
/// and then verifies signature for that digest.
///
bool Signature_verify(Signature *sig, EVP_PKEY *pkey, unsigned char *msg, size_t msg_len);

#endif
