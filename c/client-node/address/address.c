#include <stdlib.h>
#include <stdbool.h>
#include <string.h>
#include <stdio.h>
#include "address.h"
#include "libbase58.h"

char *encode_address_from_raw(unsigned char *raw, size_t len)
{
    if (len != PUBLIC_KEY_LEN)
    {
        printf("public private key length is not valid, expected: [ %i ], got: [ %zi ]\n", PUBLIC_KEY_LEN, len);
        exit(1);
    }

    size_t b58_len = ADDRESS_LEN;
    char *b58 = malloc(sizeof(char)*b58_len);
    if (b58 == NULL)
    {
        printf("failed to allocate [ %li ] bytes\n", b58_len);
        exit(1);
    }
    
    bool ok = b58enc(b58, &b58_len, (void *)raw, len);
    if (!ok)
    {
        printf("base58 encoding faild\n");
        exit(1);
    }

    return b58;
}

int decode_address_to_raw(char *str, unsigned char **raw)
{
    size_t len = PUBLIC_KEY_LEN;
    *raw = malloc(sizeof(unsigned char)*len);
    if (*raw == NULL)
    {
        printf("failed to allocate [ %li ] bytes\n", len);
        exit(1);
    }
    
    bool ok = b58tobin(*raw, &len, str, strlen(str));
    if (!ok)
    {
        printf("base58 decoding faild\n");
        exit(1);
    }

    return len;
}
