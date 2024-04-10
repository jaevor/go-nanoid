/*
Copyright 2024 jaevor.
License can be found in the LICENSE file.

Original reference: https://github.com/ai/nanoid
*/

package nanoid

import (
	crand "crypto/rand"
	"errors"
	"math/bits"
	"sync"
	"unicode"
)

type generator = func() string

// `A-Za-z0-9_-`.
var standardAlphabet = [64]byte{
	'A', 'B', 'C', 'D', 'E',
	'F', 'G', 'H', 'I', 'J',
	'K', 'L', 'M', 'N', 'O',
	'P', 'Q', 'R', 'S', 'T',
	'U', 'V', 'W', 'X', 'Y',
	'Z', 'a', 'b', 'c', 'd',
	'e', 'f', 'g', 'h', 'i',
	'j', 'k', 'l', 'm', 'n',
	'o', 'p', 'q', 'r', 's',
	't', 'u', 'v', 'w', 'x',
	'y', 'z', '0', '1', '2',
	'3', '4', '5', '6', '7',
	'8', '9', '-', '_',
}

var asciiAlphabet = [90]byte{
	'A', 'B', 'C', 'D', 'E',
	'F', 'G', 'H', 'I', 'J',
	'K', 'L', 'M', 'N', 'O',
	'P', 'Q', 'R', 'S', 'T',
	'U', 'V', 'W', 'X', 'Y',
	'Z', 'a', 'b', 'c', 'd',
	'e', 'f', 'g', 'h', 'i',
	'j', 'k', 'l', 'm', 'n',
	'o', 'p', 'q', 'r', 's',
	't', 'u', 'v', 'w', 'x',
	'y', 'z', '0', '1', '2',
	'3', '4', '5', '6', '7',
	'8', '9', '-', '_', '!',
	'#', '$', '%', '&', '(',
	')', '*', '+', ',', '.',
	':', ';', '<', '=', '>',
	'?', '@', '[', ']', '^',
	'`', '{', '|', '}', '~',
}

/*
Returns a mutexed buffered NanoID generator.

Errors if length is not within 2-255 (incl).
*/
func Standard(length int) (generator, error) {
	if invalidLength(length) {
		return nil, ErrInvalidLength
	}

	// Multiplying to increase the 'buffer' so that .Read()
	// has to be called less which is more efficient in the
	// longrun but requires more memory.
	size := length * length * 7
	// b holds the random crypto bytes.
	b := make([]byte, size)
	crand.Read(b)

	offset := 0

	// The standard alphabet is ASCII which goes up to 128 so we use bytes instead of runes.
	id := make([]byte, length)

	var mu sync.Mutex

	return func() string {
		mu.Lock()
		defer mu.Unlock()

		// Refill if all the bytes have been used.
		if offset == size {
			crand.Read(b)
			offset = 0
		}

		for i := 0; i < length; i++ {
			/*
				"It is incorrect to use bytes exceeding the alphabet size.
				The following mask reduces the random byte in the 0-255 value
				range to the 0-63 value range. Therefore, adding hacks such
				as empty string fallback or magic numbers is unneccessary because
				the bitmask trims bytes down to the alphabet size (64)."
			*/
			id[i] = standardAlphabet[b[i+offset]&63]
		}

		offset += length

		return string(id)
	}, nil
}

/*
Returns a standard NanoID generator with canonic length (21)

i.e., nanoid.Standard(21)
*/
func Canonic() (generator, error) {
	return Standard(21)
}

/*
Deprecated; use nanoid.CustomUnicode.
*/
func Custom(alphabet string, length int) (generator, error) {
	return CustomUnicode(alphabet, length)
}

/*
Returns a mutexed buffered NanoID generator which uses a custom alphabet that can contain non-ASCII (unicode).

Uses more memory by supporting unicode.
For ASCII-only, use nanoid.CustomASCII.

🟡 Errors if length is within 2-255 (incl).
*/
func CustomUnicode(alphabet string, length int) (generator, error) {
	if invalidLength(length) {
		return nil, ErrInvalidLength
	}

	alphabetLen := len(alphabet)
	// Runes to support unicode.
	runes := []rune(alphabet)

	// Because the custom alphabet is not guaranteed to have
	// 64 chars to utilise, we have to calculate a suitable mask.
	x := uint32(alphabetLen) - 1
	clz := bits.LeadingZeros32(x | 1)
	mask := (2 << (31 - clz)) - 1
	step := (length / 5) * 8

	b := make([]byte, step)
	id := make([]rune, length)

	j, idx := 0, 0

	var mu sync.Mutex

	return func() string {
		mu.Lock()
		defer mu.Unlock()

		for {
			crand.Read(b)
			for i := 0; i < step; i++ {
				idx = int(b[i]) & mask
				if idx < alphabetLen {
					id[j] = runes[idx]
					j++
					if j == length {
						j = 0
						return string(id)
					}
				}
			}
		}
	}, nil
}

// MustCustomASCII is a wrapper around CustomASCII but panics if any initialization error occurs.
func MustCustomASCII(alphabet string, length int) generator {
	g, err := CustomASCII(alphabet, length)
	if err != nil {
		panic(err.Error())
	}
	return g
}

/*
Returns a Nano ID generator which uses a custom ASCII alphabet.

Uses less memory than CustomUnicode by only supporting ASCII.
For unicode support use nanoid.CustomUnicode.

Errors if alphabet is not valid ASCII or if length is not within 2-255 (incl).
*/
func CustomASCII(alphabet string, length int) (generator, error) {
	if invalidLength(length) {
		return nil, ErrInvalidLength
	}

	alphabetLen := len(alphabet)

	for i := 0; i < alphabetLen; i++ {
		if alphabet[i] > unicode.MaxASCII {
			return nil, errors.New("not valid ascii")
		}
	}

	ab := []byte(alphabet)

	x := uint32(alphabetLen) - 1
	clz := bits.LeadingZeros32(x | 1)
	mask := (2 << (31 - clz)) - 1
	step := (length / 5) * 8

	b := make([]byte, step)
	id := make([]byte, length)

	j, idx := 0, 0

	var mu sync.Mutex

	return func() string {
		mu.Lock()
		defer mu.Unlock()

		for {
			crand.Read(b)
			for i := 0; i < step; i++ {
				idx = int(b[i]) & mask
				if idx < alphabetLen {
					id[j] = ab[idx]
					j++
					if j == length {
						j = 0
						return string(id)
					}
				}
			}
		}
	}, nil
}

/*
Returns a mutexed buffereed NanoID generator that uses an alphabet of ASCII characters 40-126 inclusive.

Errors if length is not within 2-255 (incl).
*/
func ASCII(length int) (generator, error) {
	if invalidLength(length) {
		return nil, ErrInvalidLength
	}

	size := length * length * 7
	b := make([]byte, size)
	crand.Read(b)

	offset := 0

	id := make([]byte, length)

	var mu sync.Mutex

	return func() string {
		mu.Lock()
		defer mu.Unlock()

		if offset == size {
			crand.Read(b)
			offset = 0
		}

		for i := 0; i < length; i++ {
			id[i] = asciiAlphabet[b[i+offset]&89]
		}

		offset += length

		return string(id)

	}, nil
}

var ErrInvalidLength = errors.New("nanoid: length for ID is invalid (must be within 2-255)")

func invalidLength(length int) bool {
	return length < 2 || length > 255
}
