package state

import (
	"crypto/md5"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	unencryptedVersion4StateFile string = `{
		"version": 4,
		"terraform_version": "1.9.1",
		"serial": 1,
		"lineage": "fcf2c3a0-3a0f-703f-c611-0ca7776c06b8",
		"outputs": {},
		"resources": [],
		"check_results": null
	}`

	encryptedVersion4StateFile string = `{
		"serial": 1,
		"lineage": "2ce64317-825a-f8cd-b35e-84fa110b561f",
		"meta": {
			"key_provider.pbkdf2.local": "eyJzYWx0IjoiNW9WSGpZM3BGaHRaNncwUE1hSDkrcnc2UzR1cjE4Q05adUR4RXNWM1dmcz0iLCJpdGVyYXRpb25zIjo2MDAwMDAsImhhc2hfZnVuY3Rpb24iOiJzaGE1MTIiLCJrZXlfbGVuZ3RoIjozMn0="
		},
		"encrypted_data": "e9oQmRWzPOcELCgmi1uJd5vfKMSnO7oQe/Mpyo8pKEXyNwChoPs0HpDeFhf2h32diyUORX28rWQ+D+wotHseJ+9c6EeE/G3TNr+ilmvdy0nKqZ4ABw/YoD6Zwn+DKF4qviUK2pnXN3HOQqgHUMmiuzL/AqIz1gDBncDyKJLdzuAXjXI1NyFRtCNJmVJpJkyq0ZhQG/i1KxbxAQtCAySH8T3KPE/njy4X7E08vy29rHtGxF0=",
		"encryption_version": "v0"
	}`
)

func TestParseMetadata(t *testing.T) {
	t.Run("InvalidJSON", func(t *testing.T) {
		stateFile := []byte("This is not a JSON file")

		metadata, err := ParseMetadataFromStateFile(&stateFile, nil)
		assert.Nil(t, metadata)
		require.ErrorIs(t, err, ErrParseStateFile)
	})

	t.Run("ValidUnencryptedVersion4StateFile", func(t *testing.T) {
		stateFile := []byte(unencryptedVersion4StateFile)

		metadata, err := ParseMetadataFromStateFile(&stateFile, nil)
		assert.Equal(t, Metadata{
			ChecksumMD5:      md5.Sum(stateFile),
			Encrypted:        false,
			Lineage:          "fcf2c3a0-3a0f-703f-c611-0ca7776c06b8",
			OpenTofuVersion:  "1.9.1",
			Serial:           1,
			StateFileVersion: 4,
		}, *metadata)
		assert.Nil(t, err)
	})

	t.Run("ValidUnencryptedVersion4StateFileWithChecksum", func(t *testing.T) {
		stateFile := []byte(unencryptedVersion4StateFile)
		md5Checksum := md5.Sum(stateFile)

		metadata, err := ParseMetadataFromStateFile(&stateFile, &md5Checksum)
		assert.Equal(t, Metadata{
			ChecksumMD5:      md5.Sum(stateFile),
			Encrypted:        false,
			Lineage:          "fcf2c3a0-3a0f-703f-c611-0ca7776c06b8",
			OpenTofuVersion:  "1.9.1",
			Serial:           1,
			StateFileVersion: 4,
		}, *metadata)
		assert.Nil(t, err)
	})

	t.Run("ValidEncryptedV4StateFile", func(t *testing.T) {
		stateFile := []byte(encryptedVersion4StateFile)

		metadata, err := ParseMetadataFromStateFile(&stateFile, nil)
		assert.Equal(t, Metadata{
			ChecksumMD5: md5.Sum(stateFile),
			Encrypted:   true,
			Lineage:     "2ce64317-825a-f8cd-b35e-84fa110b561f",
			Serial:      1,
		}, *metadata)
		assert.Nil(t, err)
	})
}

func TestIsEncryptedPayload(t *testing.T) {
	t.Run("InvalidJSON", func(t *testing.T) {
		stateFile := []byte("This is not a JSON file")

		isEncrypted, err := isEncryptedPayload(&stateFile)
		assert.False(t, isEncrypted)
		require.ErrorIs(t, err, ErrParseEncryptionMetadata)
	})

	t.Run("MissingEncryptionVersion", func(t *testing.T) {
		stateFile := []byte(`{
			"serial": 1,
			"lineage": "2ce64317-825a-f8cd-b35e-84fa110b561f",
			"meta": {
				"key_provider.pbkdf2.local": "eyJzYWx0IjoiNW9WSGpZM3BGaHRaNncwUE1hSDkrcnc2UzR1cjE4Q05adUR4RXNWM1dmcz0iLCJpdGVyYXRpb25zIjo2MDAwMDAsImhhc2hfZnVuY3Rpb24iOiJzaGE1MTIiLCJrZXlfbGVuZ3RoIjozMn0="
			},
			"encrypted_data": "e9oQmRWzPOcELCgmi1uJd5vfKMSnO7oQe/Mpyo8pKEXyNwChoPs0HpDeFhf2h32diyUORX28rWQ+D+wotHseJ+9c6EeE/G3TNr+ilmvdy0nKqZ4ABw/YoD6Zwn+DKF4qviUK2pnXN3HOQqgHUMmiuzL/AqIz1gDBncDyKJLdzuAXjXI1NyFRtCNJmVJpJkyq0ZhQG/i1KxbxAQtCAySH8T3KPE/njy4X7E08vy29rHtGxF0="
		}`)

		isEncrypted, err := isEncryptedPayload(&stateFile)
		assert.False(t, isEncrypted)
		assert.Nil(t, err)
	})

	t.Run("ValidEncryptedVersion4StateFile", func(t *testing.T) {
		stateFile := []byte(encryptedVersion4StateFile)

		isEncrypted, err := isEncryptedPayload(&stateFile)
		assert.True(t, isEncrypted)
		assert.Nil(t, err)
	})
}
