#include "signer.h"
#include <openssl/evp.h>
#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>

Signer Signer_new()
{
    EVP_PKEY *pkey = NULL;
    EVP_PKEY_CTX *pctx = EVP_PKEY_CTX_new_id(EVP_PKEY_ED25519, NULL);
    EVP_PKEY_keygen_init(pctx);
    EVP_PKEY_keygen(pctx, &pkey);
    EVP_PKEY_CTX_free(pctx);

    if (pkey == NULL)
    {
        printf("EVP_PKEY_ED25519 private key generation failed.\n");
        exit(1);
    }
    
    Signer s = (Signer){ .evpkey = pkey };

    return s;
}

void Signer_free(Signer *s)
{
    if (s == NULL)
    {
        return;
    }

    if (s->evpkey == NULL)
    {
        return;
    }

    EVP_PKEY_free(s->evpkey);
    s->evpkey = NULL;
    return;
}

bool Signer_is_ready(Signer *s)
{
    if (s == NULL)
    {
        return false;
    }
    return s->evpkey != NULL;
}


RawCryptoKey Signer_get_private_key(Signer *s)
{
    RawCryptoKey raw_key;
    if (!Signer_is_ready(s))
    {
        return raw_key;
    }

    raw_key.buffer = malloc(32);
    raw_key.len = 32;

    EVP_PKEY_get_raw_private_key(s->evpkey, raw_key.buffer, &(raw_key.len));

    return raw_key;
}

void RawCryptoKey_free(RawCryptoKey *r)
{
    if (r == NULL)
    {
        return;
    }
    if (r->buffer == NULL)
    {
        r->len = 0;
        return;
    }
    free(r->buffer);
    r->buffer = NULL;
    r->len = 0;
    return;
}
