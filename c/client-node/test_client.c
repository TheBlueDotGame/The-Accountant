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

static void test_signer_private_key(void)
{
       // Prepare
    Signer s = Signer_new();
    TEST_ASSERT_NOT_NULL(s.evpkey);
    
    // Test
    RawCryptoKey raw_key = Signer_get_private_key(&s);
    TEST_ASSERT_NOT_NULL(raw_key.buffer);
    TEST_ASSERT_EQUAL_UINT(32, raw_key.len);

    printf("Private key value is: [ ");
    for (size_t i = 0; i < raw_key.len; i ++)
    {
        printf("%u", raw_key.buffer[i]);
    }
    printf(" ]\n");
    // Test cleanup
    RawCryptoKey_free(&raw_key);
    TEST_ASSERT_NULL(raw_key.buffer);
    TEST_ASSERT_EQUAL_UINT(0, raw_key.len);


    // Prepare clenup
    Signer_free(&s);
    TEST_ASSERT_NULL(s.evpkey);
}

int main(void)
{
    UnityBegin("test_client.c");

    RUN_TEST(test_dummy);
    RUN_TEST(test_signer_new);
    RUN_TEST(test_signer_private_key);

    return UnityEnd();
}
