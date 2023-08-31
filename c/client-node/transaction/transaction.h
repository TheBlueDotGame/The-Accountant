///
/// Copyright (C) 2023 by Computantis
///
/// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without l> imitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
/// 
/// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
///
/// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
///
#ifndef TRANSACTION_H
#define TRANSACTION_H

#include <sys/time.h>
#include <unistd.h>
#include <openssl/sha.h>
#include <../signer/signer.h>
#include <../address/address.h>
#include <../signature/signature.h>

/// 
/// Transaction seals the embedded data cryptographically.
///
typedef struct {
    struct timeval  created_at;
    char            issuer_address[ADDRESS_LEN];
    char            receiver_address[ADDRESS_LEN];
    char            *subject;
    unsigned char   *data;
    unsigned char   [SIGNATURE_LEN]issuer_signature;
    unsigned char   [SIGNATURE_LEN]receiver_signature;
    unsigned char   [SHA256_DIGEST_LENGTH]hash;
} Transaction;

///
/// Transaction_new creates new transaction signing the timestamp, subject, message and the receiver.
/// The receiver_address is in base58 encoded format.
///
Transaction Transaction_new(char *subject, const unsigned char *data, const char *receiver_address, Signer *s);

/// 
/// Transaction_receiver_sign signs transaction by the receiver only if message digest is correct and issuer signature is valid,
/// otherwise returns false.
/// the data string and receiver_address are compied.
/// Function caller is responsible for cleaning the data and receiver_address string by itself.
///
bool Transaction_receiver_sign(Transaction *trx, signer_f signer_sign, Signer *s);

///
/// Transaction_get_data returns underlining data as a copy.
/// Allows to free Transaction still keeping valid data.
/// Function caller is required to free data string itself.
///
unsigned char *Transaction_get_data(Transaction *trx);

///
/// Transaction_free frees the transaction and all underlining data.
///
void Transaction_free(Transaction *trx);

#endif
