#include "test-framework/unity.h"
#include "client.h"

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

int main(void)
{
    UnityBegin("test_client.c");

    RUN_TEST(test_dummy);

    return UnityEnd();
}
