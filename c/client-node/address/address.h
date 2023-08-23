#ifndef ADDRESS_H
#define ADDRESS_H
#define PUBLIC_KEY_LEN 32
#define ADDRESS_LEN 51

#include <ctype.h>
#include <stdbool.h>

///
/// encode_address_from_raw encodes new public address string from raw buffer.
/// Encoder uses base58 encoding algorithm and returns nullable string.
///
char *encode_address_from_raw(unsigned char  *raw, size_t len);

/// 
/// decode_to_raw decodes nullable string to raw bytes.
/// It will override the underlining raw bytes array,
/// it is the best to pass the pointer to the NULL pointer.
/// Decoder uses base58 decoding algorithm.
/// Returns length of raw bytes array.
///
int decode_address_to_raw(char *str, unsigned char **raw);

#endif
