// cryptopasta - basic cryptography examples
//
// Written in 2015 by George Tankersley <george.tankersley@gmail.com>
//
// To the extent possible under law, the author(s) have dedicated all copyright
// and related and neighboring rights to this software to the public domain
// worldwide. This software is distributed without any warranty.
//
// You should have received a copy of the CC0 Public Domain Dedication along
// with this software. If not, see // <http://creativecommons.org/publicdomain/zero/1.0/>.

package crypto

import (
	"testing"
)

func TestPasswordHashing(t *testing.T) {
	bcryptTests := []struct {
		plaintext []byte
		hash      []byte
	}{
		{
			plaintext: []byte("password"),
			hash:      []byte("$2a$14$uALAQb/Lwl59oHVbuUa5m.xEFmQBc9ME/IiSgJK/VHtNJJXASCDoS"),
		},
	}

	for _, tt := range bcryptTests {
		hashed, err := HashPassword(tt.plaintext)
		if err != nil {
			t.Error(err)
		}

		if err = CheckPasswordHash(hashed, tt.plaintext); err != nil {
			t.Error(err)
		}
	}
}

func BenchmarkBcrypt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := HashPassword([]byte("thisisareallybadpassword"))
		if err != nil {
			b.Error(err)
			break
		}
	}
}
