///
/// Copyright (C) 2023 by Computantis
///
/// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without l> imitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
/// 
/// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
///
/// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
///
#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <string.h>
#include <openssl/evp.h>
#include <openssl/pem.h>
#include <openssl/sha.h>
#include "signer.h"
#include "../signature/signature.h"

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

static bool signer_is_ready(Signer *s)
{
    if (s == NULL)
    {
        return false;
    }
    return s->evpkey != NULL;
}

bool Signer_save_pem(Signer *s, const char *f)
{
    FILE* outfile;

    if (!signer_is_ready(s))
    {
        return false;
    }
 
    outfile = fopen(f, "wb");
    if (outfile == NULL)
    {
        return false;
    }

    int flag = PEM_write_PrivateKey(outfile, s->evpkey, NULL, NULL, 0, NULL, NULL);
    fclose(outfile);
    return flag == 1;
}

bool Signer_read_pem(Signer *s, const char *f)
{
    FILE* infile;

    infile = fopen(f, "r");
    if (infile == NULL)
    {
        return false;
    }

    s->evpkey = PEM_read_PrivateKey(infile, NULL, NULL, NULL);
    fclose(infile);
    return s->evpkey != NULL;
}

RawCryptoKey Signer_get_private_key(Signer *s)
{
    RawCryptoKey raw_key;
    if (!signer_is_ready(s))
    {
        return raw_key;
    }

    raw_key.buffer = malloc(KEY_LEN);
    raw_key.len = KEY_LEN;

    int success = EVP_PKEY_get_raw_private_key(s->evpkey, raw_key.buffer, &(raw_key.len));
    if (success != 1)
    {
        printf("Writing raw private key failed failed\n");
        exit(1);
    }

    return raw_key;
}

RawCryptoKey Signer_get_public_key(Signer *s)
{
    RawCryptoKey raw_key;
    if (!signer_is_ready(s))
    {
        return raw_key;
    }

    raw_key.buffer = malloc(KEY_LEN);
    raw_key.len = KEY_LEN;
    int success = EVP_PKEY_get_raw_public_key(s->evpkey, raw_key.buffer, &(raw_key.len));
    if (success != 1)
    {
        printf("Writing raw public key failed failed\n");
        exit(1);
    }

    return raw_key;
}

EVP_PKEY *RawCryptoKey_get_evp_public_key(RawCryptoKey *r)
{
    return EVP_PKEY_new_raw_public_key(EVP_PKEY_ED25519, NULL, r->buffer, r->len);
}

EVP_PKEY *RawCryptoKey_get_evp_private_key(RawCryptoKey *r)
{
    return EVP_PKEY_new_raw_private_key(EVP_PKEY_ED25519, NULL, r->buffer, r->len);
}

void RawCryptoKey_free(RawCryptoKey *r)
{
    if (r == NULL)
    {
        return;
    }
    r->len = 0;
    
    if (r->buffer == NULL)
    {
        return;
    }
    free(r->buffer);
    r->buffer = NULL;
    return;
}

Signature Signer_sign(Signer *s, unsigned char *msg, size_t len)
{
    if (s == NULL)
    {
        printf("Signer is NULL\n");
        exit(1);
    }
    if (len <= 0)
    {
        len = strlen((char*)msg);
    }
    if (len == 0)
    {
        printf("Message of zero length cannot be signed\n");
        exit(1);
    }

    size_t digest_len = SHA256_DIGEST_LENGTH; 
    unsigned char *digest = malloc(sizeof(unsigned char) * digest_len);
    if (digest == NULL)
    {
        printf("Allocating memory of size [ %li ] bytes for digest buffer failed\n", digest_len);
        exit(1);
    }

    unsigned char *flag = SHA256(msg, len, digest);
    if (flag == NULL)
    {
        printf("Hashing message failed\n");
        exit(1);
    } 

    EVP_MD_CTX *mdctx = EVP_MD_CTX_create();
    if (mdctx == NULL)
    {
        printf("Context allocation failed\n");
        exit(1);
    }
    
    int success = EVP_DigestSignInit(mdctx, NULL, NULL, NULL, s->evpkey);
    if (success != 1)
    {
        printf("Digest sign allocation failed\n");
        exit(1);
    }
    
    size_t sig_len = 0;
    success =  EVP_DigestSign(mdctx, NULL, &sig_len, digest, digest_len);
    if (success != 1)
    {
        printf("Calculating signature length failed\n");
        exit(1);
    }

    unsigned char *signature = OPENSSL_zalloc(sizeof(unsigned char) * sig_len);
    if (signature == NULL)
    {
        printf("Signature allocation failed\n");
        exit(1);
    }
    success = EVP_DigestSign(mdctx, signature, &sig_len, digest, digest_len);
    if (success != 1)
    {
        printf("Signing digest failed\n");
        exit(1);
    }

    EVP_MD_CTX_destroy(mdctx);

    Signature sig = (Signature){ .digest_buffer = digest, .digest_len = digest_len, .signature_buffer = signature, .signature_len = sig_len };

    return sig;
}

