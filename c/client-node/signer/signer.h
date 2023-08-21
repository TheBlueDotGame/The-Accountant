#ifndef SIGNER_H
#define SIGNER_H

#include <openssl/evp.h>

///
/// Signer contains the functionality to sign data and create digest as well as validate data digest and signature;
///
typedef struct {
    EVP_PKEY *evpkey;
} Signer;

///
/// Signer_new creates a new Signer and returns a copy of that entity.
///
Signer Signer_new();

///
/// Signer_free frees inner values of a signer and sets the inner reference to NULL pointer;
///
void Signer_free(Signer *s);

#endif
