///
/// Copyright (C) 2023 by Computantis
///
/// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without l> imitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
/// 
/// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
///
/// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
///
#include <stdlib.h>
#include <string.h>
#include <openssl/evp.h>
#include <openssl/sha.h>
#include "signature.h"

void Signature_free(Signature *sig)
{
    if (sig == NULL)
    {
        return;
    }
    if (sig->signature_buffer != NULL)
    {
        OPENSSL_free(sig->signature_buffer);
        sig->signature_buffer = NULL;
    }
    if (sig->digest_buffer != NULL)
    {
        free(sig->digest_buffer);
        sig->digest_buffer = NULL;
    }
    sig->signature_len = 0;
    sig->digest_len = 0;
    return;
}

static bool digest_cmp(unsigned char *a, unsigned char* b)
{
    for (size_t i = 0; i < SHA256_DIGEST_LENGTH; i++)
    {
        if (a[i] != b[i])
        {
            return false;
        }
    }

    return true;
}

bool Signature_verify(Signature *sig, EVP_PKEY *pkey, unsigned char *msg, size_t msg_len)
{

    if (sig == NULL)
    {
        printf("Signerature is NULL\n");
        exit(1);
    }

    if (sig->signature_buffer == NULL || sig->signature_len == 0)
    {
        printf("Signature inner buffer is empty \n");
        exit(1);
    }
    if (sig->digest_buffer == NULL || sig->digest_len == 0)
    {
        printf("Digest inner buffer is empty \n");
        exit(1);
    }

    if (msg_len <= 0)
    {
        msg_len = strlen((char*)msg);
    }
    if (msg_len == 0)
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

    unsigned char *flag = SHA256(msg, msg_len, digest);
    if (flag == NULL)
    {
        printf("Hashing message failed\n");
        exit(1);
    }

    bool ok = digest_cmp(digest, sig->digest_buffer);
    if (!ok)
    {
        free(digest);
        digest = NULL;
        return false;
    }
    free(digest);
    digest = NULL;

    EVP_MD_CTX *mdctx = EVP_MD_CTX_create();
    if (mdctx == NULL)
    {
        printf("Context allocation failed\n");
        exit(1);
    }
    if (pkey == NULL)
    {
        printf("Public key is NULL\n");
        exit(1);
    }

    int success = EVP_DigestVerifyInit(mdctx, NULL, NULL, NULL, pkey);
    if (success != 1)
    {
        printf("Public key context allocation failed.\n");
        exit(1);
    }

    success = EVP_DigestVerify(mdctx, sig->signature_buffer, sig->signature_len, sig->digest_buffer, sig->digest_len);

    EVP_MD_CTX_free(mdctx);

    return success == 1;
}

