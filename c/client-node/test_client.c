#include "test-framework/unity.h"
#include "client.h"
#include "./signer/signer.h"

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

static void test_signer_create_and_free(void){
    Signer s = Signer_new();
    TEST_ASSERT_NOT_NULL(s.evpkey);

    Signer_free(&s);
    TEST_ASSERT_NULL(s.evpkey);
}

int main(void)
{
    UnityBegin("test_client.c");

    RUN_TEST(test_dummy);
    RUN_TEST(test_signer_create_and_free);

    return UnityEnd();
}
