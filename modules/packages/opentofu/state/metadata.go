package state

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	// See https://opentofu.org/docs/language/state/
	OpenTofuStateFilename = "terraform.tfstate"
)

var (
	ErrParseStateFile          = errors.New("Failed to parse the state file")
	ErrParseEncryptionMetadata = errors.New("Failed to parse the encryption metadata")
)

// Metadata represents the metadata of an OpenTofu/Terraform package.
type Metadata struct {
	// The MD5 checksum of the state file.
	ChecksumMD5 [16]byte

	// Whether or not the state file is encrypted.
	Encrypted bool

	// The "lineage" is a unique ID assigned to a state when it is created. If a
	// lineage is different, then it means the states were created at different
	// times and its very likely you're modifying a different state.
	//
	// See https://opentofu.org/docs/language/state/backends/#manual-state-pullpush.
	Lineage string `json:"lineage"`

	// OpenTofu/Terraform version used to generate the state file.
	//
	// Absent from encrypted state files.
	OpenTofuVersion string `json:"terraform_version,omitempty"`

	// Every state has a monotonically increasing "serial" number. If the
	// destination state has a higher serial, OpenTofu will not allow you to write
	// it since it means that changes have occurred since the state you're
	// attempting to write.
	//
	// See https://opentofu.org/docs/language/state/backends/#manual-state-pullpush.
	Serial uint64 `json:"serial"`

	// Version of the state file format.
	//
	// Absent from encrypted state files.
	StateFileVersion uint64 `json:"version,omitempty"`
}

// ParseMetadataFromStateFile extracts the package metadata from a state file.
//
// If md5Checksum is present, the MD5 hash will not be computed again but store
// directly in the metadata.
func ParseMetadataFromStateFile(stateFile *[]byte, md5Checksum *[16]byte) (*Metadata, error) {
	var metadata Metadata
	if err := json.Unmarshal(*stateFile, &metadata); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseStateFile, err)
	}

	isEncrypted, err := isEncryptedPayload(stateFile)
	if err != nil {
		return nil, err
	}
	metadata.Encrypted = isEncrypted

	if md5Checksum != nil {
		metadata.ChecksumMD5 = *md5Checksum

	} else {
		metadata.ChecksumMD5 = md5.Sum(*stateFile)
	}

	return &metadata, nil
}

// isEncryptedPayload detects whether or not the OpenTofu/Terraform state file's
// payload is encrypted.
func isEncryptedPayload(data *[]byte) (bool, error) {
	type EncryptionMetadata struct {
		Data    []byte            `json:"encrypted_data"`
		Meta    map[string][]byte `json:"meta"`
		Version string            `json:"encryption_version"`
	}

	var encryptionMetadata EncryptionMetadata
	if err := json.Unmarshal(*data, &encryptionMetadata); err != nil {
		return false, fmt.Errorf("%w: %v", ErrParseEncryptionMetadata, err)
	}

	return encryptionMetadata.Version != "", nil
}
