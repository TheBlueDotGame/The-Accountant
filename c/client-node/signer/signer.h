///
/// Copyright (C) 2023 by Computantis
///
/// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without l> imitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
/// 
/// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
///
/// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
///
#ifndef SIGNER_H
#define SIGNER_H
#define KEY_LEN 32

#include <stdbool.h>
#include <stddef.h>
#include <openssl/evp.h>
#include "../signature/signature.h"

///
/// Signer type allows to perform cryptographic operations on the given bytes buffer.
/// It uses ed25519 symmetric elliptic curve algorithm. 
/// Holds value of private key. Do not expose that value.
///
typedef struct {
    EVP_PKEY *evpkey;
} Signer;

///
/// RawCryprographicKey holds the raw key as bytes buffer;
///
typedef struct {
    unsigned char *buffer;
    size_t len;
} RawCryptoKey;

///
/// Signer_new creates a new Signer and returns a copy of that entity.
///
Signer Signer_new();

///
/// Signer_free frees inner values of a signer and sets the inner reference to NULL pointer;
///
void Signer_free(Signer *s);

///
/// Signer_save_pem saves private key to pem file.
/// Returns true on success or false otherwise;
///
bool Signer_save_pem(Signer *s, const char *f);

/// 
/// Signer_read_pem reads pem file to the Signer.
/// Returns true on success or false otherwise.
///
bool Signer_read_pem(Signer *s, const char *f);

///
/// Signer_get_private_key returns raw private key. 
///
RawCryptoKey Signer_get_private_key(Signer *s);

///
///Signer_get_public_key returns raw public key.
///
RawCryptoKey Signer_get_public_key(Signer *s);


/// 
/// RawCryptoKey_get_evp_public_key returns pointer to EVP_PKEY
/// from openssl.evp.h library that is a public key.
///
EVP_PKEY *RawCryptoKey_get_evp_public_key(RawCryptoKey *r);

/// 
/// RawCryptoKey_get_evp_private_key returns pointer to EVP_PKEY
/// from openssl.evp.h library that is a private key.
///
EVP_PKEY *RawCryptoKey_get_evp_private_key(RawCryptoKey *r);

///
/// RawCryptoKey_free frees the RawCryptoKey;
///
void RawCryptoKey_free(RawCryptoKey *r);

///
/// Signature Signer_sign signs the provided buffer of bytes.
/// Returns Signature with inner signature and digest buffers.
/// If message is a string then len can be -1 to 
/// allow the function to pre calculate the message size itself.
/// Function performs below steps:
/// - Creates sha256 digest from the message.
/// - Generate signature context for the Signer ed25519 private key.
/// - Signs the digest.
/// - Creates Signature structure.
/// For the ed25519 algorithm digest is 32 bytes long and signature is 64 bytes long.
///
Signature Signer_sign(Signer *s, unsigned char *msg, size_t len);

#endif
