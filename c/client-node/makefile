### If you wish to use extra libraries (math.h for instance),
### add their flags here (-lm in our case) in the "LIBS" variable.

LIBS = -lm -lssl -UOPENSSL_NO_EC -lcrypto

###
CFLAGS  = -std=c99
CFLAGS += -g
CFLAGS += -Wall
CFLAGS += -Wextra
CFLAGS += -pedantic
CFLAGS += -Werror
CFLAGS += -Wmissing-declarations
CFLAGS += -DUNITY_SUPPORT_64 -DUNITY_OUTPUT_COLOR

ASANFLAGS  = -fsanitize=address
ASANFLAGS += -fno-common
ASANFLAGS += -fno-omit-frame-pointer

TEST_PACKAGES = test-framework/unity.c
PACKAGES = ./signer/*.c
PACKAGES += ./address/*.c
PACKAGES += ./signature/*.c
PACKAGES += ./config/*.c
PACKAGES += ./transaction/*.c
PACKAGES += ./*.c

# Used to test static functions.
UTFLAGS = -DUNIT_TEST

.PHONY: test
test: tests.out
	@./tests.out

.PHONY: memcheck
memcheck: ./*.c ./*.h
	@echo Compiling $@
	@$(CC) $(ASANFLAGS) $(CFLAGS) $(UTFLAGS) $(TEST_PACKAGES) $(PACKAGES) -o memcheck.out $(LIBS)
	@./memcheck.out
	@echo "Memory check passed"

.PHONY: clean
clean:
	rm -rf *.o *.out *.out.dSYM *.pem

tests.out: ./*.c ./*.h
	@echo Compiling $@
	@$(CC) $(CFLAGS) $(UTFLAGS) $(TEST_PACKAGES) $(PACKAGES) -o tests.out $(LIBS)

