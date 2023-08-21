#ifndef SIGNER_H
#define SIGNER_H

#include <openssl/evp.h>
#include <stdbool.h>
#include <stddef.h>

///
/// Signer type allows to perform cryptographic opperations on the given bytes buffer.
/// It uses ed25519 symetric eliptic curve algorithm. 
///
typedef struct {
    EVP_PKEY *evpkey;
} Signer;

///
/// RawCryprographicKey keeps the raw key as bytes buffer;
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
///Singner_is_ready checks if Signer is ready to perform signing.
///
bool Signer_is_ready(Signer *s);

///
/// Signer_get_private_key returns raw private key. 
///
RawCryptoKey Signer_get_private_key(Signer *s);


///
/// RawCryptoKey_free frees the RawCryptoKey;
///
void RawCryptoKey_free(RawCryptoKey *r);

#endif
