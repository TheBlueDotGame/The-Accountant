#ifndef SIGNER_H
#define SIGNER_H
#define KEY_LEN 32

#include <openssl/evp.h>
#include <stdbool.h>
#include <stddef.h>

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
/// Signature is an entity holding the signature and digest of the message that was a subject of signing.
///
typedef struct {
    unsigned char *digest_buffer;
    unsigned char *signature_buffer;
    size_t digest_len;
    size_t signature_len;
} Signature;

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
/// RawCryptoKey_free frees the RawCryptoKey;
///
void RawCryptoKey_free(RawCryptoKey *r);

///
/// Signature Signer_sign signs the provided buffer of bytes.
/// Is returning Signature with inner signature and digest.
/// If message is a string then len can be passed as -1, 
/// function will then calculate the message size itself.
/// This function performs below steps:
/// - Creates sha256 digest from the message.
/// - Generate signature context for the Signer ed25519 private key.
/// - Signs the digest generating the signature.
/// - constrect Signature struct.
/// Digest is 32 bytes long and signature is 64 bytes long for the ed25519 algorithm.
///
Signature Signer_sign(Signer *s, unsigned char *msg, size_t len);

///
/// Signature_free frees the signature.
///
void Signature_free(Signature *sig);

#endif
