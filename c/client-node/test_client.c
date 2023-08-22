#include "test-framework/unity.h"
#include "client.h"
#include "./signer/signer.h"
#include <stdbool.h>
#include <stdio.h>

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

int main(void)
{
    UnityBegin("test_client.c");

    RUN_TEST(test_dummy);
    RUN_TEST(test_signer_new);
    RUN_TEST(test_signer_public_key);
    RUN_TEST(test_signer_private_key);
    RUN_TEST(test_signer_save_read_pem);

    return UnityEnd();
}
