package opentofu

const (
	// See https://opentofu.org/docs/language/state/
	OpenTofuStateFilename = "terraform.tfstate"
)

// Metadata represents the metadata of an OpenTofu/Terraform package
type Metadata struct {
	Version int `json:"version"`
}
