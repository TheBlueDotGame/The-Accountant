package aeswrapper

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncryptDecryptSuccess(t *testing.T) {
	data := []byte("2a7afdd039c69497d6591f1f2aa3d72a9119f53d0166b6227feeb84f95b5020d1706f36d5f20197328e883f9e7048a4b5395953aec2633047f32e15cd834d627a5d985ef7299a5a91bd216c2eee8f4abc9147fae55e2abfc615041084a849c880a7e99e7c6c8f313ed125d0ba1bbdb0e7c18435d80016bbc67dffcfcd3c95167fb6da64df411553e4faeb4a880bd2d5ab14da54a29108c07d98aab2ed61f621087677dc310b98459192239373b2e38a186ec9a48558a35485e0e7671ea3a2c41ea750ec3026c14be8801c41d9c70cc3593ffb3e98f2026903c1b86401a4ae02844cf3ccf336ad5df7340173bd245d1662aa88201253b41dfbda4c5238f627dfd")
	keys := []string{
		"f5f9fb83df631c6746dcc7fe7b21de1e2e33b2584428b37b911cf818a7cd9d84",
		"a531345e49fc6047f780174cbe8958397a70ed9ac5f2cafa9ab6598732cc70db",
		"1d457fe37e4d95c7afaa266952541d52c2b9ec6115793df570ddb66f18613881",
	}

	for i, k := range keys {
		pass, err := hex.DecodeString(k)
		assert.Nil(t, err)
		t.Run(fmt.Sprintf("TestEncryptDecryptSuccess-%d-%d", i, len(pass)), func(t *testing.T) {
			h := New()
			enc, err := h.Encrypt(pass, data)
			assert.Nil(t, err)
			dec, err := h.Decrypt(pass, enc)
			assert.Nil(t, err)
			assert.Equal(t, data, dec)
		})
	}
}

func TestEncryptDecryptFailEncrypt(t *testing.T) {
	data := []byte("2a7afdd039c69497d6591f1f2aa3d72a9119f53d0166b6227feeb84f95b5020d1706f36d5f20197328e883f9e7048a4b5395953aec2633047f32e15cd834d627a5d985ef7299a5a91bd216c2eee8f4abc9147fae55e2abfc615041084a849c880a7e99e7c6c8f313ed125d0ba1bbdb0e7c18435d80016bbc67dffcfcd3c95167fb6da64df411553e4faeb4a880bd2d5ab14da54a29108c07d98aab2ed61f621087677dc310b98459192239373b2e38a186ec9a48558a35485e0e7671ea3a2c41ea750ec3026c14be8801c41d9c70cc3593ffb3e98f2026903c1b86401a4ae02844cf3ccf336ad5df7340173bd245d1662aa88201253b41dfbda4c5238f627dfd")
	keys := []string{
		"f5f9fb83df631c6746dcc7fe7b21de1e2e33b2584428b37b911cf818a7cd9d",
		"a531345e49fc6047f780174cbe8958397a70ed9ac5f2cafa9ab6598732cc70dbaa",
		"1d457fe37e4d95c7afaa266952541d52c2b9ec6115793df5",
	}

	for i, k := range keys {
		pass, err := hex.DecodeString(k)
		assert.Nil(t, err)
		t.Run(fmt.Sprintf("TestEncryptDecryptSuccess-%d-%d", i, len(pass)), func(t *testing.T) {
			h := New()
			_, err := h.Encrypt(pass, data)
			assert.NotNil(t, err)
		})
	}
}

func TestEncryptDecryptFailDecrypt(t *testing.T) {
	data := []byte("2a7afdd039c69497d6591f1f2aa3d72a9119f53d0166b6227feeb84f95b5020d1706f36d5f20197328e883f9e7048a4b5395953aec2633047f32e15cd834d627a5d985ef7299a5a91bd216c2eee8f4abc9147fae55e2abfc615041084a849c880a7e99e7c6c8f313ed125d0ba1bbdb0e7c18435d80016bbc67dffcfcd3c95167fb6da64df411553e4faeb4a880bd2d5ab14da54a29108c07d98aab2ed61f621087677dc310b98459192239373b2e38a186ec9a48558a35485e0e7671ea3a2c41ea750ec3026c14be8801c41d9c70cc3593ffb3e98f2026903c1b86401a4ae02844cf3ccf336ad5df7340173bd245d1662aa88201253b41dfbda4c5238f627dfd")
	keys := []string{
		"f5f9fb83df631c6746dcc7fe7b21de1e2e33b2584428b37b911cf818a7cd9d",
		"a531345e49fc6047f780174cbe8958397a70ed9ac5f2cafa9ab6598732cc70dbaa",
		"1d457fe37e4d95c7afaa266952541d52c2b9ec6115793df5",
	}

	for i, k := range keys {
		pass, err := hex.DecodeString(k)
		assert.Nil(t, err)
		t.Run(fmt.Sprintf("TestEncryptDecryptSuccess-%d-%d", i, len(pass)), func(t *testing.T) {
			h := New()
			_, err := h.Decrypt(pass, data)
			assert.NotNil(t, err)
		})
	}
}
