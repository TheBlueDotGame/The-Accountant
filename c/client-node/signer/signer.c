#include "signer.h"
#include <openssl/evp.h>

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
    
    Signer s = (Signer){
        .evpkey = pkey,
    };

    return s;
}

void Signer_free(Signer *s)
{
    if (s == NULL)
    {
        return;
    }

    EVP_PKEY_free(s->evpkey);
    s->evpkey = NULL;
    return;
}
