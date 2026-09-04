package crypto

import (
	"encoding/json"
	"testing"
	"time"
)

func FuzzCryptoVerify(f *testing.F) {
	kp, _ := GenerateKeyPair("test-key")
	kr := NewKeyRing()
	kr.RegisterKey("test-key", kp.PublicKey)

	b, _ := SignPayload(kp, "v1", 1*time.Hour, map[string]string{"foo": "bar"})
	validBytes, _ := json.Marshal(b)

	f.Add(validBytes)
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"version":"v1","keyId":"test-key","signature":"deadbeef"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var bundle SignedBundle
		if err := json.Unmarshal(data, &bundle); err != nil {
			return
		}
		_ = kr.Verify(&bundle)
	})
}
