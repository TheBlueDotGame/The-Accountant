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
#include <../signer/signer.h>
#include <../signature/signature.h>

static void convertToCharArrayLittleEndian(unsigned char *arr, long long a)
{
    for (int i = 0; i < 8; ++i)
    {
        arr[i] = (unsigned char)((((unsigned long long) a) >> (56 - (8*i))) & 0xFFu);
    }
}

Transaction Transaction_new(char *subject, unsigned char *data, char *receiver_address, Signer *s)
{
    // prepare buffer phase
    struct timeval now;
    gettimeofday(&now, NULL);
    RawCryptoKey raw_key = Signer_get_public_key(&s);
    char *issuer_address = encode_address_from_raw(raw_key.buffer, raw_key.len);
    
    size_t subject_len = strlen(subject);
    size_t data_len = strlen(data);
    size_t issuer_len = strlen(issuer_address);
    size_t receiver_len = strlen(receiver_address);
    size_t ts_len = 8;
    size_t buf_len = subject_len + data_len + issuer_len + receiver_len + ts_len; 

    long long ms_time = now.tv_sec + now.tv_usec;
    unsigned char ts[ts_len];
    convertToCharArrayLittleEndian(&ts, ms_time);
   
    unsigned char *buffer = malloc(sizeof(unsigned char) * buf_len);
    if (buffer == NULL)
    {
        printf("Failed to allocate [ %li ] bytes\n", buf_len);
        exit(1);
    }
    size_t p = 0;
    strncpy((char *)buffer, subject, subject_len);
    p = p + subject_len;
    strncpy((char *)buffer[p], (char *)data, data_len);
    p = p + data_len;
    strncpy((char *)buffer[p], issuer_address, issuer_len);
    p = p + issuer_len;
    strncpy((char *)buffer[p], receiver_address, receiver_len);
    p = p + receiver_len;
    strncpy((char *)buffer[p], (char *)ts, ts_len);

    // sign phase
    Signature signature = Signer_sign(s, buffer, buf_len);

    // create transasaction phase

    Transaction trx;
    trx.created_at = now,
    strncpy(trx.issuer_address, issuer_address, issuer_len);
    strncpy(trx.receiver_address, receiver_address, receiver_len);
    trx.subject = malloc(sizeof(char)*subject_len);
    if (trx.subject == NULL)
    {
        printf("Failed to allocate [ %li ] bytes\n", subject_len);
        exit(1);
    }
    strncpy(trx.subject, subject, subject_len);
    trx.data = malloc(sizeof(unsigned char) * data_len);
    if (trx.data == NULL)
    {
        printf("Failed to allocate [ %li ] bytes\n", data_len);
        exit(1);
    }
    strncpy((char*)trx.data, (char*)data, data_len);
    if (signature.signature_len != SIGNATURE_LEN)
    {
        printf("Wrong signature_len, expected: [ %li ], received: [ %li ]\n", SIGNATURE_LEN, signature.signature_len);
        exit(1);
    }
    strncpy((char *)trx.issuer_signature, signature->signature_buffer, SIGNATURE_LEN);
    trx.receiver_signature = "";
    if (signature.digest_len != SHA256_DIGEST_LENGTH)
    {
        printf("Wrong digest_len, expected: [ %li ], received: [ %li ]\n", SHA256_DIGEST_LENGTH, signature->digest_len);
        exit(1);
    }
    strncpy((char *)trx.hash, signature.digest_buffer, SHA256_DIGEST_LENGTH);

    // cleanup phase
    free(issuer_address);
    issuer_address = NULL;
    
    RawCryptoKey_free(raw_key);

    free(buffer);
    buffer = NULL;

    Signature_free(&signature);

    return trx;
}

