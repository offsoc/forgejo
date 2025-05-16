package integration

import (
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	opentofu_state_module "forgejo.org/modules/packages/opentofu/state"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageOpenTofuHttpBackend(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	rootUrl := fmt.Sprintf("/api/packages/%s/opentofu/http/state", user.Name)

	t.Run("Upload", func(t *testing.T) {
		unencryptedVersion4StateFile := `{
			"version": 4,
			"terraform_version": "1.9.0",
			"serial": 1,
			"lineage": "fcf2c3a0-3a0f-703f-c611-0ca7776c06b8",
			"outputs": {},
			"resources": [],
			"check_results": null
		}`

		encryptedVersion4StateFile := `{
			"serial": 1,
			"lineage": "2ce64317-825a-f8cd-b35e-84fa110b561f",
			"meta": {
				"key_provider.pbkdf2.local": "eyJzYWx0IjoiNW9WSGpZM3BGaHRaNncwUE1hSDkrcnc2UzR1cjE4Q05adUR4RXNWM1dmcz0iLCJpdGVyYXRpb25zIjo2MDAwMDAsImhhc2hfZnVuY3Rpb24iOiJzaGE1MTIiLCJrZXlfbGVuZ3RoIjozMn0="
			},
			"encrypted_data": "e9oQmRWzPOcELCgmi1uJd5vfKMSnO7oQe/Mpyo8pKEXyNwChoPs0HpDeFhf2h32diyUORX28rWQ+D+wotHseJ+9c6EeE/G3TNr+ilmvdy0nKqZ4ABw/YoD6Zwn+DKF4qviUK2pnXN3HOQqgHUMmiuzL/AqIz1gDBncDyKJLdzuAXjXI1NyFRtCNJmVJpJkyq0ZhQG/i1KxbxAQtCAySH8T3KPE/njy4X7E08vy29rHtGxF0=",
			"encryption_version": "v0"
		}`

		// Sends a state upload request without being authenticated.
		t.Run("UnauthenticatedUploadRequest", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, "POST", rootUrl+"/unauthenticated", strings.NewReader(unencryptedVersion4StateFile)).SetHeader("Content-Type", "application/json")
			MakeRequest(t, req, http.StatusUnauthorized)
		})

		// Sends a state upload request with an invalid JSON payload.
		t.Run("InvalidJSON", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, "POST", rootUrl+"/invalid-json", strings.NewReader("This is not a valid JSON payload")).SetHeader("Content-Type", "application/json").AddBasicAuth(user.Name)
			MakeRequest(t, req, http.StatusBadRequest)
		})

		// Sends a state upload request with an invalid MD5 checksum as HTTP header.
		t.Run("InvalidMD5Checksum", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			req := NewRequestWithBody(t, "POST", rootUrl+"/invalid-md5", strings.NewReader(unencryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").SetHeader("Content-MD5", "SW52YWxpZCBtZDUgY2hlY2tzdW0K").AddBasicAuth(user.Name)
			MakeRequest(t, req, http.StatusBadRequest)
		})

		// Sends a valid unencrypted version 4 state file.
		t.Run("UploadValidUnencryptedVersion4StateFile", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			packageName := "v4-unencrypted"
			packageVersion := "1"

			pvs, err := packages.GetVersionsByPackageName(db.DefaultContext, user.ID, packages.TypeOpenTofuState, packageName)
			require.NoError(t, err)
			assert.Len(t, pvs, 0)

			md5Hash := md5.Sum([]byte(unencryptedVersion4StateFile))
			md5Base64 := base64.StdEncoding.EncodeToString(md5Hash[:])

			req := NewRequestWithBody(t, "POST", rootUrl+"/"+packageName, strings.NewReader(unencryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").SetHeader("Content-MD5", md5Base64).AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusOK)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NotEmpty(t, bodyBytes)

			pvs, err = packages.GetVersionsByPackageName(db.DefaultContext, user.ID, packages.TypeOpenTofuState, packageName)
			require.NoError(t, err)
			assert.Len(t, pvs, 1)

			pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
			require.NoError(t, err)
			assert.Equal(t, packageName, pd.Package.Name)
			assert.Equal(t, packageVersion, pd.Version.Version)
			assert.IsType(t, &opentofu_state_module.Metadata{}, pd.Metadata)
			assert.False(t, pd.Metadata.(*opentofu_state_module.Metadata).Encrypted)

			pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
			require.NoError(t, err)
			assert.Len(t, pfs, 1)
			assert.Equal(t, opentofu_state_module.OpenTofuStateFilename, pfs[0].Name)
			assert.True(t, pfs[0].IsLead)

			pb, err := packages.GetBlobByID(db.DefaultContext, pfs[0].BlobID)
			require.NoError(t, err)
			assert.Equal(t, int64(len(unencryptedVersion4StateFile)), pb.Size)
			assert.Equal(t, pd.Metadata.(*opentofu_state_module.Metadata).ChecksumMD5, md5.Sum([]byte(unencryptedVersion4StateFile)))

			req = NewRequestWithBody(t, "POST", rootUrl+"/"+packageName, strings.NewReader(unencryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").SetHeader("Content-MD5", md5Base64).AddBasicAuth(user.Name)
			MakeRequest(t, req, http.StatusConflict)
		})

		// Sends a valid encrypted version 4 state file.
		t.Run("UploadValidEncryptedVersion4StateFile", func(t *testing.T) {
			defer tests.PrintCurrentTest(t)()

			packageName := "v4-encrypted"
			packageVersion := "1"

			pvs, err := packages.GetVersionsByPackageName(db.DefaultContext, user.ID, packages.TypeOpenTofuState, packageName)
			require.NoError(t, err)
			assert.Len(t, pvs, 0)

			md5Hash := md5.Sum([]byte(encryptedVersion4StateFile))
			md5Base64 := base64.StdEncoding.EncodeToString(md5Hash[:])

			req := NewRequestWithBody(t, "POST", rootUrl+"/"+packageName, strings.NewReader(encryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").SetHeader("Content-MD5", md5Base64).AddBasicAuth(user.Name)
			resp := MakeRequest(t, req, http.StatusOK)
			assert.Contains(t, resp.Header().Get("Content-Type"), "application/json")

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NotEmpty(t, bodyBytes)

			pvs, err = packages.GetVersionsByPackageName(db.DefaultContext, user.ID, packages.TypeOpenTofuState, packageName)
			require.NoError(t, err)
			assert.Len(t, pvs, 1)

			pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
			require.NoError(t, err)
			assert.Equal(t, packageName, pd.Package.Name)
			assert.Equal(t, packageVersion, pd.Version.Version)
			assert.IsType(t, &opentofu_state_module.Metadata{}, pd.Metadata)
			assert.True(t, pd.Metadata.(*opentofu_state_module.Metadata).Encrypted)

			pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
			require.NoError(t, err)
			assert.Len(t, pfs, 1)
			assert.Equal(t, opentofu_state_module.OpenTofuStateFilename, pfs[0].Name)
			assert.True(t, pfs[0].IsLead)

			pb, err := packages.GetBlobByID(db.DefaultContext, pfs[0].BlobID)
			require.NoError(t, err)
			assert.Equal(t, int64(len(encryptedVersion4StateFile)), pb.Size)
			assert.Equal(t, pd.Metadata.(*opentofu_state_module.Metadata).ChecksumMD5, md5.Sum([]byte(encryptedVersion4StateFile)))

			req = NewRequestWithBody(t, "POST", rootUrl+"/"+packageName, strings.NewReader(encryptedVersion4StateFile)).SetHeader("Content-Type", "application/json").SetHeader("Content-MD5", md5Base64).AddBasicAuth(user.Name)
			MakeRequest(t, req, http.StatusConflict)
		})
	})
}
