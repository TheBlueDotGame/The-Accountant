///
/// Copyright (C) 2023 by Computantis
///
/// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without l> imitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
/// 
/// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
///
/// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
///
#include <sys/time.h>
#include <unistd.h>
#include <string.h>
#include <openssl/sha.h>
#include "transaction.h"
#include "../signer/signer.h"
#include "../signature/signature.h"
#include "../wallet/wallet.h"

static void convertToCharArrayLittleEndian(unsigned char *arr, long long a)
{
    for (int i = 0; i < 8; ++i)
    {
        arr[i] = (unsigned char)((((unsigned long long) a) >> (56 - (8*i))) & 0xFFu);
    }
}

Transaction *Transaction_new(const char *subject, const unsigned char *data, const char *receiver_address, Signer *s)
{
    if (receiver_address == NULL || strlen(receiver_address) == 0)
    {
        printf("Given receiver address is empty\n");
        exit(1);
    }
    if (subject == NULL || strlen(subject) == 0)
    {
        printf("Given subject is empty\n");
        exit(1);
    }
    if (data == NULL || strlen((char *)data) == 0)
    {
        printf("Given data is empty\n");
        exit(1);
    }
    // prepare buffer phase
    struct timeval now;
    gettimeofday(&now, NULL);
    RawCryptoKey raw_key = Signer_get_public_key(s);
    char *issuer_address = encode_address_from_raw(WalletVersion, raw_key.buffer, raw_key.len);
    if (issuer_address == NULL || strlen(issuer_address) == 0)
    {
        printf("Failed to read issuer address\n");
        exit(1);
    }
    
    size_t subject_len = strlen(subject);
    size_t data_len = strlen((char *)data);
    size_t issuer_len = strlen(issuer_address);
    size_t receiver_len = strlen(receiver_address);
    size_t ts_len = 8;
    size_t buf_len = subject_len + data_len + issuer_len + receiver_len + ts_len; 

    long long ms_time = now.tv_sec + now.tv_usec;
    unsigned char ts[ts_len];
    convertToCharArrayLittleEndian(ts, ms_time);
   
    unsigned char *buffer = malloc(sizeof(unsigned char) * buf_len);
    if (buffer == NULL)
    {
        printf("Failed to allocate [ %li ] bytes\n", buf_len);
        exit(1);
    }
    size_t p = 0;
    strncpy((char *)buffer, subject, subject_len);
    p = p + subject_len;
    strncpy((char *)(buffer + p), (char *)data, data_len);
    p = p + data_len;
    strncpy((char *)(buffer + p), issuer_address, issuer_len);
    p = p + issuer_len;
    strncpy((char *)(buffer + p), receiver_address, receiver_len);
    p = p + receiver_len;
    strncpy((char *)(buffer + p), (char *)ts, ts_len);

    // sign phase
    Signature signature = Signer_sign(s, buffer, buf_len);

    // create transaction phase

    Transaction *trx = malloc(sizeof(Transaction));
    trx->created_at = now;

    trx->issuer_address = malloc(sizeof(char *)*issuer_len);
    if (trx->issuer_address == NULL)
    {
        printf("Failed to allocate [ %li ] bytes\n", issuer_len);
        exit(1);
    }
    strncpy(trx->issuer_address, issuer_address, issuer_len);

    trx->receiver_address = malloc(sizeof(char *)*issuer_len);
    if (trx->receiver_address == NULL)
    {
        printf("Failed to allocate [ %li ] bytes\n", receiver_len);
        exit(1);
    }
    strncpy(trx->receiver_address, receiver_address, receiver_len);

    trx->subject = malloc(sizeof(char)*subject_len);
    if (trx->subject == NULL)
    {
        printf("Failed to allocate [ %li ] bytes\n", subject_len);
        exit(1);
    }
    strncpy(trx->subject, subject, subject_len);

    trx->data = malloc(sizeof(unsigned char)*data_len);
    if (trx->data == NULL)
    {
        printf("Failed to allocate [ %li ] bytes\n", data_len);
        exit(1);
    }
    strncpy((char*)trx->data, (char*)data, data_len);

    if (signature.signature_len != SIGNATURE_LEN)
    {
        printf("Wrong signature_len, expected: [ %i ], received: [ %li ]\n", SIGNATURE_LEN, signature.signature_len);
        exit(1);
    }
    trx->issuer_signature = malloc(sizeof(unsigned char)*SIGNATURE_LEN);
    if (trx->issuer_signature == NULL)
    {
        printf("Failed to allocate [ %i ] bytes\n", SIGNATURE_LEN);
        exit(1);
    }
    strncpy((char *)trx->issuer_signature, (char *)signature.signature_buffer, SIGNATURE_LEN);
   
    trx->receiver_signature = NULL;

    if (signature.digest_len != SHA256_DIGEST_LENGTH)
    {
        printf("Wrong digest_len, expected: [ %i ], received: [ %li ]\n", SHA256_DIGEST_LENGTH, signature.digest_len);
        exit(1);
    }
    trx->hash = malloc(sizeof(unsigned char)*SHA256_DIGEST_LENGTH);
    if (trx->hash == NULL)
    {
        printf("Failed to allocate [ %i ] bytes\n", SHA256_DIGEST_LENGTH);
        exit(1);
    }
    strncpy((char *)trx->hash, (char *)signature.digest_buffer, SHA256_DIGEST_LENGTH);

    // cleanup phase
    free(issuer_address);
    issuer_address = NULL;
    
    RawCryptoKey_free(&raw_key);

    free(buffer);
    buffer = NULL;

    Signature_free(&signature);

    return trx;
}

void Transaction_free(Transaction **trx)
{
    if (*trx == NULL)
    {
        return;
    }
    if ((*trx)->data != NULL)
    {
        free((*trx)->data);
        (*trx)->data = NULL;
    }
    if ((*trx)->subject != NULL)
    {
        free((*trx)->subject);
        (*trx)->subject = NULL;
    }
    if ((*trx)->issuer_address != NULL)
    {
        free((*trx)->issuer_address);
        (*trx)->issuer_address = NULL;
    }
    if ((*trx)->receiver_address != NULL)
    {
        free((*trx)->receiver_address);
        (*trx)->receiver_address = NULL;
    }
    if ((*trx)->issuer_signature != NULL)
    {
        free((*trx)->issuer_signature);
        (*trx)->issuer_signature = NULL;
    }
    if ((*trx)->receiver_signature != NULL)
    {
        free((*trx)->receiver_signature);
        (*trx)->receiver_signature = NULL;
    }
    if ((*trx)->hash != NULL)
    {
        free((*trx)->hash);
        (*trx)->hash = NULL;
    }

    free(*trx);
    *trx = NULL;
    return;
}
