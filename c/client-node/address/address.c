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
#include "address.h"
#include "libbase58.h"

char *encode_address_from_raw(unsigned char *raw, size_t len)
{
    if (len != PUBLIC_KEY_LEN)
    {
        printf("Public private key length is not valid, expected: [ %i ], got: [ %zi ]\n", PUBLIC_KEY_LEN, len);
        exit(1);
    }

    size_t b58_len = ADDRESS_LEN;
    char *b58 = malloc(sizeof(char)*b58_len);
    if (b58 == NULL)
    {
        printf("Failed to allocate [ %li ] bytes\n", b58_len);
        exit(1);
    }
    
    bool ok = b58enc(b58, &b58_len, (void *)raw, len);
    if (!ok)
    {
        printf("Base58 encoding faild\n");
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
        printf("Failed to allocate [ %li ] bytes\n", len);
        exit(1);
    }
    
    bool ok = b58tobin(*raw, &len, str, strlen(str));
    if (!ok)
    {
        printf("Base58 decoding faild\n");
        exit(1);
    }

    return len;
}
