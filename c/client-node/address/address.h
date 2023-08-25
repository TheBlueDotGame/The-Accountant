#ifndef ADDRESS_H
#define ADDRESS_H
#define PUBLIC_KEY_LEN 32
#define ADDRESS_LEN 51

#include <ctype.h>
#include <stdbool.h>

///
/// encode_address_from_raw encodes new public address string from raw buffer.
/// Encoder uses base58 encoding algorithm and returns pointer to nullable string.
/// Caller takes responsibility of freeing the returned string. 
///
char *encode_address_from_raw(unsigned char  *raw, size_t len);

/// 
/// decode_address_to_raw decodes nullable string to raw bytes.
/// unsigned char **raw bytes array represents the variable that points to the array of bytes that the key will be decoded to.
/// It will override the underlining unsigned char *raw bytes array.
/// Best practice is to pass unsigned char **raw as a pointer to NULL pointer.
/// Decoder uses base58 decoding algorithm.
/// Returns length of unsigned char *raw bytes array.
/// Caller takes the responsibility to free the unsigned char *raw bytes array.
///
int decode_address_to_raw(char *str, unsigned char **raw);

#endif
