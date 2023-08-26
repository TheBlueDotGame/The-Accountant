///
/// Copyright (C) 2023 by Computantis
///
/// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without l> imitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
/// 
/// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
///
/// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
///
#include <stdbool.h>
#include <stdio.h>
#include <string.h>
#include <openssl/evp.h>
#include "test-framework/unity.h"
#include "client.h"
#include "./signer/signer.h"
#include "./address/address.h"
#include "./signature/signature.h"


void setUp(void)
{
}

void tearDown(void)
{
}

static void test_dummy(void)
{
    TEST_ASSERT_TRUE(check_client(1));
}

static void test_signer_new()
{
    Signer s = Signer_new();
    TEST_ASSERT_NOT_NULL(s.evpkey);
    
    Signer_free(&s);
    TEST_ASSERT_NULL(s.evpkey);
}

static void test_signer_public_key()
{
    // Prepare
    Signer s = Signer_new();
    TEST_ASSERT_NOT_NULL(s.evpkey);
    
    // Test
    RawCryptoKey raw_key = Signer_get_public_key(&s);
    TEST_ASSERT_NOT_NULL(raw_key.buffer);
    TEST_ASSERT_EQUAL_UINT(32, raw_key.len);

    // Test cleanup
    RawCryptoKey_free(&raw_key);
    TEST_ASSERT_NULL(raw_key.buffer);
    TEST_ASSERT_EQUAL_UINT(0, raw_key.len);
    
    // Prepare clenup
    Signer_free(&s);
    TEST_ASSERT_NULL(s.evpkey);
}

static void test_signer_private_key(void)
{
       // Prepare
    Signer s = Signer_new();
    TEST_ASSERT_NOT_NULL(s.evpkey);
    
    // Test
    RawCryptoKey raw_key = Signer_get_private_key(&s);
    TEST_ASSERT_NOT_NULL(raw_key.buffer);
    TEST_ASSERT_EQUAL_UINT(32, raw_key.len);

    // Test cleanup
    RawCryptoKey_free(&raw_key);
    TEST_ASSERT_NULL(raw_key.buffer);
    TEST_ASSERT_EQUAL_UINT(0, raw_key.len);


    // Prepare clenup
    Signer_free(&s);
    TEST_ASSERT_NULL(s.evpkey);
}

static void test_signer_save_read_pem(void)
{
    // Prepare
    Signer s0 = Signer_new();
    TEST_ASSERT_NOT_NULL(s0.evpkey);

    bool ok = Signer_save_pem(&s0, "ed25519.pem");
    TEST_ASSERT_TRUE(ok);

    Signer s1;
    ok = Signer_read_pem(&s1, "ed25519.pem");
    TEST_ASSERT_TRUE(ok);
    TEST_ASSERT_NOT_NULL(s1.evpkey);

    // Prepare clenup
    Signer_free(&s0);
    TEST_ASSERT_NULL(s0.evpkey);
    Signer_free(&s1);
    TEST_ASSERT_NULL(s1.evpkey);
}

static void test_encode_decode_public_address()
{
    // Prepare
    Signer s = Signer_new();
    TEST_ASSERT_NOT_NULL(s.evpkey);

    RawCryptoKey raw_key = Signer_get_public_key(&s);
    TEST_ASSERT_NOT_NULL(raw_key.buffer);
    TEST_ASSERT_EQUAL_UINT(32, raw_key.len); 

    // Test
    char *address = encode_address_from_raw(raw_key.buffer, raw_key.len);
    TEST_ASSERT_NOT_NULL(address);
    TEST_ASSERT_GREATER_OR_EQUAL_size_t(32, strlen(address));

    RawCryptoKey new_raw_key = (RawCryptoKey){ .buffer = NULL, .len = 0};
    new_raw_key.len = decode_address_to_raw(address, &new_raw_key.buffer);
    TEST_ASSERT_NOT_NULL(new_raw_key.buffer);
    TEST_ASSERT_EQUAL_UINT(32, new_raw_key.len);

    TEST_ASSERT_EQUAL_CHAR_ARRAY(raw_key.buffer, new_raw_key.buffer, raw_key.len);

    free(address);

    // Prepare cleanup
    RawCryptoKey_free(&raw_key);
    TEST_ASSERT_NULL(raw_key.buffer);
    TEST_ASSERT_EQUAL_UINT(0, raw_key.len);


    RawCryptoKey_free(&new_raw_key);
    TEST_ASSERT_NULL(new_raw_key.buffer);
    TEST_ASSERT_EQUAL_UINT(0, new_raw_key.len);

    Signer_free(&s);
    TEST_ASSERT_NULL(s.evpkey);
}

static void test_signer_sign(void)
{
    // Prepare
    Signer s = Signer_new();
    TEST_ASSERT_NOT_NULL(s.evpkey);

    // Test
    char msg[24] = "this is message to sing\0";
    Signature sig = Signer_sign(&s, (unsigned char*)msg, 24);
    TEST_ASSERT_NOT_NULL(sig.digest_buffer);
    TEST_ASSERT_EQUAL_UINT(32, sig.digest_len);
    TEST_ASSERT_NOT_NULL(sig.signature_buffer);
    TEST_ASSERT_EQUAL_UINT(64, sig.signature_len);

    Signature_free(&sig);

    // Prepare clenup
    Signer_free(&s);
    TEST_ASSERT_NULL(s.evpkey);
}

static void test_signer_verify_signature_success(void)
{
    // Prepare
    Signer s = Signer_new();
    TEST_ASSERT_NOT_NULL(s.evpkey);

    RawCryptoKey raw_pub_key = Signer_get_public_key(&s);
    TEST_ASSERT_NOT_NULL(raw_pub_key.buffer);
    TEST_ASSERT_EQUAL_UINT(32, raw_pub_key.len);
    
    // Test 
    char msg[24] = "this is message to sing\0";
    Signature sig = Signer_sign(&s, (unsigned char*)msg, 24);
    TEST_ASSERT_NOT_NULL(sig.digest_buffer);
    TEST_ASSERT_EQUAL_UINT(32, sig.digest_len);
    TEST_ASSERT_NOT_NULL(sig.signature_buffer);
    TEST_ASSERT_EQUAL_UINT(64, sig.signature_len);
    
    EVP_PKEY *pkey = RawCryptoKey_get_evp_public_key(&raw_pub_key);
    TEST_ASSERT_NOT_NULL(pkey);
    
    bool success = Signature_verify(&sig, pkey, (unsigned char *)msg, 24);
    TEST_ASSERT_TRUE(success);

    Signature_free(&sig);
    EVP_PKEY_free(pkey);

    // Prepare clenup
    RawCryptoKey_free(&raw_pub_key);
    TEST_ASSERT_NULL(raw_pub_key.buffer);
    TEST_ASSERT_EQUAL_UINT(0, raw_pub_key.len);
    Signer_free(&s);
    TEST_ASSERT_NULL(s.evpkey);
}

static void test_signer_verify_signature_failure_wrong_pub_key(void)
{
    // Prepare
    Signer s = Signer_new();
    TEST_ASSERT_NOT_NULL(s.evpkey);

    RawCryptoKey raw_pub_key = Signer_get_public_key(&s);
    TEST_ASSERT_NOT_NULL(raw_pub_key.buffer);
    TEST_ASSERT_EQUAL_UINT(32, raw_pub_key.len);

    Signer wrong_s = Signer_new();
    TEST_ASSERT_NOT_NULL(wrong_s.evpkey);
    RawCryptoKey wrong_raw_pub_key = Signer_get_public_key(&wrong_s);
    TEST_ASSERT_NOT_NULL(wrong_raw_pub_key.buffer);
    TEST_ASSERT_EQUAL_UINT(32, wrong_raw_pub_key.len);
    
    // Test 
    char msg[24] = "this is message to sing\0";
    Signature sig = Signer_sign(&s, (unsigned char*)msg, 24);
    TEST_ASSERT_NOT_NULL(sig.digest_buffer);
    TEST_ASSERT_EQUAL_UINT(32, sig.digest_len);
    TEST_ASSERT_NOT_NULL(sig.signature_buffer);
    TEST_ASSERT_EQUAL_UINT(64, sig.signature_len);
    
    EVP_PKEY *pkey = RawCryptoKey_get_evp_public_key(&wrong_raw_pub_key);
    TEST_ASSERT_NOT_NULL(pkey);
    
    bool success = Signature_verify(&sig, pkey, (unsigned char *)msg, 24);
    TEST_ASSERT_FALSE(success);

    Signature_free(&sig);
    EVP_PKEY_free(pkey);

    // Prepare clenup
    RawCryptoKey_free(&raw_pub_key);
    TEST_ASSERT_NULL(raw_pub_key.buffer);
    TEST_ASSERT_EQUAL_UINT(0, raw_pub_key.len);


    RawCryptoKey_free(&wrong_raw_pub_key);
    TEST_ASSERT_NULL(wrong_raw_pub_key.buffer);
    TEST_ASSERT_EQUAL_UINT(0, wrong_raw_pub_key.len);

    Signer_free(&s);
    TEST_ASSERT_NULL(s.evpkey);
    
    Signer_free(&wrong_s);
    TEST_ASSERT_NULL(wrong_s.evpkey);
}

static void test_signer_verify_signature_failure_corrupted_msg(void)
{
    // Prepare
    Signer s = Signer_new();
    TEST_ASSERT_NOT_NULL(s.evpkey);

    RawCryptoKey raw_pub_key = Signer_get_public_key(&s);
    TEST_ASSERT_NOT_NULL(raw_pub_key.buffer);
    TEST_ASSERT_EQUAL_UINT(32, raw_pub_key.len);
    
    // Test 
    char msg[24] = "this is message to sing\0";
    Signature sig = Signer_sign(&s, (unsigned char*)msg, 24);
    TEST_ASSERT_NOT_NULL(sig.digest_buffer);
    TEST_ASSERT_EQUAL_UINT(32, sig.digest_len);
    TEST_ASSERT_NOT_NULL(sig.signature_buffer);
    TEST_ASSERT_EQUAL_UINT(64, sig.signature_len);
    
    EVP_PKEY *pkey = RawCryptoKey_get_evp_public_key(&raw_pub_key);
    TEST_ASSERT_NOT_NULL(pkey);

    // corrupt message

    msg[3] = 'S';
    
    bool success = Signature_verify(&sig, pkey, (unsigned char *)msg, 24);
    TEST_ASSERT_FALSE(success);

    Signature_free(&sig);
    EVP_PKEY_free(pkey);

    // Prepare clenup
    RawCryptoKey_free(&raw_pub_key);
    TEST_ASSERT_NULL(raw_pub_key.buffer);
    TEST_ASSERT_EQUAL_UINT(0, raw_pub_key.len);
    Signer_free(&s);
    TEST_ASSERT_NULL(s.evpkey);
}

static void test_signer_verify_signature_failure_corrupted_digest(void)
{
    // Prepare
    Signer s = Signer_new();
    TEST_ASSERT_NOT_NULL(s.evpkey);

    RawCryptoKey raw_pub_key = Signer_get_public_key(&s);
    TEST_ASSERT_NOT_NULL(raw_pub_key.buffer);
    TEST_ASSERT_EQUAL_UINT(32, raw_pub_key.len);
    
    // Test 
    char msg[24] = "this is message to sing\0";
    Signature sig = Signer_sign(&s, (unsigned char*)msg, 24);
    TEST_ASSERT_NOT_NULL(sig.digest_buffer);
    TEST_ASSERT_EQUAL_UINT(32, sig.digest_len);
    TEST_ASSERT_NOT_NULL(sig.signature_buffer);
    TEST_ASSERT_EQUAL_UINT(64, sig.signature_len);
    
    EVP_PKEY *pkey = RawCryptoKey_get_evp_public_key(&raw_pub_key);
    TEST_ASSERT_NOT_NULL(pkey);

    // corrupt message

    sig.digest_buffer[3] = 'X';
    sig.digest_buffer[4] = 'X';
    
    bool success = Signature_verify(&sig, pkey, (unsigned char *)msg, 24);
    TEST_ASSERT_FALSE(success);

    Signature_free(&sig);
    EVP_PKEY_free(pkey);

    // Prepare clenup
    RawCryptoKey_free(&raw_pub_key);
    TEST_ASSERT_NULL(raw_pub_key.buffer);
    TEST_ASSERT_EQUAL_UINT(0, raw_pub_key.len);
    Signer_free(&s);
    TEST_ASSERT_NULL(s.evpkey);
}

int main(void)
{
    UnityBegin("test_client.c");

    RUN_TEST(test_dummy);
    RUN_TEST(test_signer_new);
    RUN_TEST(test_signer_public_key);
    RUN_TEST(test_signer_private_key);
    RUN_TEST(test_signer_save_read_pem);
    RUN_TEST(test_encode_decode_public_address);
    RUN_TEST(test_signer_sign);
    RUN_TEST(test_signer_verify_signature_success);
    RUN_TEST(test_signer_verify_signature_failure_wrong_pub_key);
    RUN_TEST(test_signer_verify_signature_failure_corrupted_msg);
    RUN_TEST(test_signer_verify_signature_failure_corrupted_digest);

    return UnityEnd();
}
