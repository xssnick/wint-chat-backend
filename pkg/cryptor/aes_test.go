package cryptor

import (
	"reflect"
	"testing"
	"time"
)

func TestSame(t *testing.T) {
	a, err := NewAESGCM(make([]byte, 32))
	if err != nil {
		t.Errorf("a err %v", err)
		return
	}

	time.Sleep(50 * time.Millisecond)
	b, err := NewAESGCM(make([]byte, 32))
	if err != nil {
		t.Errorf("b err")
		return
	}

	enc, _ := a.Encrypt([]byte("sometext"))

	dec, err := b.Decrypt(enc)
	if err != nil {
		t.Errorf("Decrypt err")
	}

	if !reflect.DeepEqual(dec, []byte("sometext")) {
		t.Errorf("Not equal result")
	}

	if !reflect.DeepEqual(a, b) {
		t.Errorf("Not equal instances")
	}

	c, _ := NewAESGCM(append(make([]byte, 31), 0x01))
	if reflect.DeepEqual(a, c) {
		t.Errorf("Equal but diff key")
	}
}

func Test_aes_Decrypt(t *testing.T) {
	type args struct {
		key  []byte
		text []byte
	}

	// result of Encrypt(make([]byte, 32), []byte{0xAA,0xBB})
	dec := []byte{0xAA, 0xBB}
	enced := []byte{152, 186, 159, 178, 5, 93, 217, 0, 224, 215, 147, 213, 115, 181, 195, 7, 7, 13, 132, 167, 217, 26, 123, 237, 238, 241, 57, 4, 8, 189}

	tests := []struct {
		name          string
		args          args
		wantDecrypted []byte
		wantErr       bool
		wantErrInit   bool
	}{
		{
			name: "bad key under size",
			args: args{
				key:  make([]byte, 11),
				text: enced,
			},
			wantDecrypted: nil,
			wantErrInit:   true,
		},
		{
			name: "bad key over size",
			args: args{
				key:  make([]byte, 33),
				text: enced,
			},
			wantDecrypted: nil,
			wantErrInit:   true,
		},
		{
			name: "bad text",
			args: args{
				key:  make([]byte, 32),
				text: append(enced[1:], 0xFF),
			},
			wantDecrypted: nil,
			wantErr:       true,
		},
		{
			name: "empty text",
			args: args{
				key:  make([]byte, 32),
				text: []byte{},
			},
			wantDecrypted: nil,
			wantErr:       true,
		},
		{
			name: "nil text",
			args: args{
				key:  make([]byte, 32),
				text: nil,
			},
			wantDecrypted: nil,
			wantErr:       true,
		},
		{
			name: "empty key",
			args: args{
				key:  []byte{},
				text: enced,
			},
			wantDecrypted: nil,
			wantErrInit:   true,
		},
		{
			name: "wrong key",
			args: args{
				key:  append([]byte{0x05}, make([]byte, 31)...),
				text: enced,
			},
			wantDecrypted: nil,
			wantErr:       true,
		},
		{
			name: "ok",
			args: args{
				key:  make([]byte, 32),
				text: enced,
			},
			wantDecrypted: dec,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := NewAESGCM(tt.args.key)
			if (err != nil) != tt.wantErrInit {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			gotDecrypted, err := a.Decrypt(tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotDecrypted, tt.wantDecrypted) {
				t.Errorf("Decrypt() gotDecrypted = %v, want %v", gotDecrypted, tt.wantDecrypted)
			}
		})
	}
}

func Test_aes_Encrypt(t *testing.T) {
	type args struct {
		key  []byte
		text []byte
	}

	dec := []byte{0xAA, 0xBB}

	ap, _ := NewAESGCM(make([]byte, 32))
	preenc, _ := ap.Encrypt([]byte{0xAA, 0xBB})

	tests := []struct {
		name             string
		args             args
		notWantEncrypted []byte
		wantEncrypted    []byte
		wantErr          bool
		wantErrInit      bool
	}{
		{
			name: "bad key under size",
			args: args{
				key:  make([]byte, 11),
				text: dec,
			},
			wantErrInit: true,
		},
		{
			name: "bad key over size",
			args: args{
				key:  make([]byte, 33),
				text: dec,
			},
			wantErrInit: true,
		},
		{
			name: "empty text",
			args: args{
				key:  make([]byte, 32),
				text: []byte{},
			},
			wantErr: true,
		},
		{
			name: "nil text",
			args: args{
				key:  make([]byte, 32),
				text: nil,
			},
			wantErr: true,
		},
		{
			name: "empty key",
			args: args{
				key:  []byte{},
				text: dec,
			},
			wantErrInit: true,
		},
		{
			name: "reuse iv",
			args: args{
				key:  make([]byte, 32),
				text: dec,
			},
			notWantEncrypted: preenc,
			wantErr:          false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, err := NewAESGCM(tt.args.key)
			if (err != nil) != tt.wantErrInit {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			gotEncrypted, err := a.Encrypt(tt.args.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.name == "reuse iv" {
				//must not equal
				if reflect.DeepEqual(gotEncrypted, tt.notWantEncrypted) {
					t.Errorf("Encrypt() gotEncrypted = %v, want %v", gotEncrypted, tt.notWantEncrypted)
				}

				return
			}

			if !reflect.DeepEqual(gotEncrypted, tt.wantEncrypted) {
				t.Errorf("Encrypt() gotEncrypted = %v, want %v", gotEncrypted, tt.wantEncrypted)
			}
		})
	}
}
