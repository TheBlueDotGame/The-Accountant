#ifndef SIGNER_H
#define SIGNER_H
#define KEY_LEN 32

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
/// Signer_save_pem saves private key to pem file.
/// Returns true on success or flase otherwise;
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
/// RawCryptoKey_free frees the RawCryptoKey;
///
void RawCryptoKey_free(RawCryptoKey *r);

#endif
