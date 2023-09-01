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
#include <stdbool.h>
#include <string.h>
#include <stdio.h>
#include <openssl/sha.h>
#include "address.h"
#include "libbase58.h"

static void checksum(unsigned char *payload, unsigned char *dest, size_t payload_len, size_t dest_len)
{
    if (dest_len > SHA256_DIGEST_LENGTH)
    {
        printf("checksum payload cannot be performad on destination length bigger than [ %i ] \n", SHA256_DIGEST_LENGTH);
        exit(1);
    }
    unsigned char digest_0[SHA256_DIGEST_LENGTH];
    unsigned char *flag = SHA256(payload, payload_len, digest_0);
    if (flag == NULL)
    {
        printf("Hashing payload failed\n");
        exit(1);
    }

    unsigned char digest_1[SHA256_DIGEST_LENGTH];
    flag = SHA256(digest_0, (size_t)SHA256_DIGEST_LENGTH, digest_1);
    if (flag == NULL)
    {
        printf("Hashing digest_0 failed\n");
        exit(1);
    }

   void *flag_m = memcpy(dest, digest_1, dest_len);
    if (flag_m == NULL)
    {
        printf("Copying checksum failed\n");
        exit(1);
    }

    return;
}

char *encode_address_from_raw(unsigned char version, unsigned char *raw, size_t len)
{
    if (len != PUBLIC_KEY_LEN)
    {
        printf("Public private key length is not valid, expected: [ %i ], got: [ %li ]\n", PUBLIC_KEY_LEN, len);
        exit(1);
    }

    unsigned char encode[(size_t)1+PUBLIC_KEY_LEN+CHECKSUM_LEN];
    encode[0] = version;
    void *flag = memcpy(encode+1, raw, (size_t)PUBLIC_KEY_LEN);
    if (flag == NULL)
    {
        printf("Copying public key failed\n");
        exit(1);
    }
    flag = memcpy(encode+1+(int)PUBLIC_KEY_LEN, raw, (size_t)CHECKSUM_LEN);
    if (flag == NULL)
    {
        printf("Copying public key failed\n");
        exit(1);
    }

    size_t b58_len = ADDRESS_LEN;
    char *b58 = malloc(sizeof(char)*b58_len);
    if (b58 == NULL)
    {
        printf("Failed to allocate [ %li ] bytes\n", b58_len);
        exit(1);
    }
    
    bool ok = b58enc(b58, &b58_len, (void *)encode, (size_t)1+PUBLIC_KEY_LEN+CHECKSUM_LEN);
    if (!ok)
    {
        printf("Base58 encoding faild\n");
        exit(1);
    }

    return b58;
}

int decode_address_to_raw(unsigned char version, char *str, unsigned char **raw)
{
    size_t len = PUBLIC_KEY_LEN+CHECKSUM_LEN+1;
    unsigned char decoded[PUBLIC_KEY_LEN+CHECKSUM_LEN+1]; 
    bool ok = b58tobin(decoded, &len, str, strlen(str));
    if (!ok)
    {
        printf("Base58 decoding faild\n");
        exit(1);
    }

    unsigned char actual_checksum[CHECKSUM_LEN];
    void *flag = memcpy(actual_checksum, decoded + 1 + PUBLIC_KEY_LEN, CHECKSUM_LEN);
    if (flag == NULL)
    {
        printf("Copying checksum failed\n");
        exit(1);
    }

    unsigned char vrs = decoded[0];
    if (vrs != version)
    {
        return 0;
    }

    *raw = malloc(sizeof(unsigned char) * (size_t)PUBLIC_KEY_LEN);
    if (*raw == NULL)
    {
        printf("Failed to allocate [ %i ] bytes\n", PUBLIC_KEY_LEN);
        exit(1);
    }

    flag = memcpy(*raw, decoded + 1, (size_t)PUBLIC_KEY_LEN);
    if (flag == NULL)
    {
        printf("Copying public key failed failed\n");
        exit(1);
    }

    unsigned char pub_key_vrs[1+PUBLIC_KEY_LEN];
    pub_key_vrs[0] = version;
    flag = memcpy(pub_key_vrs+1, *raw, (size_t)PUBLIC_KEY_LEN);
    if (flag == NULL)
    {
        printf("Copying public key failed failed\n");
        exit(1);
    }

    unsigned char target_checksum[CHECKSUM_LEN];
    checksum(pub_key_vrs, target_checksum, 1+PUBLIC_KEY_LEN, CHECKSUM_LEN);

    int equality = strncmp((char *)actual_checksum, (char *)target_checksum, (size_t)CHECKSUM_LEN);
    if (equality == 0)
    {
        free(*raw);
        *raw = NULL;
        return 0;
    }

    return len;
}
