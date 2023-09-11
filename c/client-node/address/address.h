///
/// Copyright (C) 2023 by Computantis
///
/// Permission is hereby granted, free of charge, to any person obtaining a copy of this software and associated documentation files (the "Software"), to deal in the Software without restriction, including without l> imitation the rights to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of the Software, and to permit persons to whom the Software is furnished to do so, subject to the following conditions:
/// 
/// The above copyright notice and this permission notice shall be included in all copies or substantial portions of the Software.
///
/// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
///
#ifndef ADDRESS_H
#define ADDRESS_H
#define PUBLIC_KEY_LEN 32
#define ADDRESS_LEN 56
#define CHECKSUM_LEN 4

#include <ctype.h>
#include <stdbool.h>

///
/// encode_address_from_raw encodes new public address string from raw buffer.
/// Encoder uses base58 encoding algorithm and returns pointer to nullable string.
/// Caller takes responsibility of freeing the returned string. 
///
char *encode_address_from_raw(unsigned char wallet_version, unsigned char  *raw, size_t len);

/// 
/// decode_address_to_raw decodes nullable string to raw bytes.
/// unsigned char **raw bytes array represents the variable that points to the array of bytes that the key will be decoded to.
/// It will override the underlining unsigned char *raw bytes array.
/// Best practice is to pass unsigned char **raw as a pointer to NULL pointer.
/// Decoder uses base58 decoding algorithm.
/// Returns length of unsigned char *raw bytes array.
/// Caller takes the responsibility to free the unsigned char *raw bytes array.
///
int decode_address_to_raw(unsigned char wallet_version, char *str, unsigned char **raw);

#endif
