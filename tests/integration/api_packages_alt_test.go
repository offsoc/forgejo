// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	packages_module "forgejo.org/modules/packages"
	rpm_module "forgejo.org/modules/packages/rpm"
	"forgejo.org/modules/setting"
	"forgejo.org/modules/util"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ulikunitz/xz"
)

func TestPackageAlt(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	packageName := "gitea-test"
	packageVersion := "1.0.2-1"
	packageArchitecture := "x86_64"

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	base64RpmPackageContent := `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`
	rpmPackageContent, err := base64.StdEncoding.DecodeString(base64RpmPackageContent)
	require.NoError(t, err)

	zr, err := gzip.NewReader(bytes.NewReader(rpmPackageContent))
	require.NoError(t, err)

	content, err := io.ReadAll(zr)
	require.NoError(t, err)

	rootURL := fmt.Sprintf("/api/packages/%s/alt", user.Name)

	for _, group := range []string{"", "el9", "el9/stable"} {
		t.Run(fmt.Sprintf("[Group:%s]", group), func(t *testing.T) {
			var groupParts []string
			uploadURL := rootURL
			if group != "" {
				groupParts = strings.Split(group, "/")
				uploadURL = strings.Join(append([]string{rootURL}, groupParts...), "/")
			} else {
				groupParts = strings.Split("alt", "/")
			}
			groupURL := strings.Join(append([]string{rootURL}, groupParts...), "/")

			t.Run("RepositoryConfig", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", groupURL+".repo")
				resp := MakeRequest(t, req, http.StatusOK)

				expected := fmt.Sprintf(`[gitea-%s]
name=%s
baseurl=%s
enabled=1`,
					strings.Join(append([]string{user.LowerName}, groupParts...), "-"),
					strings.Join(append([]string{user.Name, setting.AppName}, groupParts...), " - "),
					util.URLJoin(setting.AppURL, groupURL),
				)

				assert.Equal(t, expected, resp.Body.String())
			})

			t.Run("Upload", func(t *testing.T) {
				url := uploadURL + "/upload"

				req := NewRequestWithBody(t, "PUT", url, bytes.NewReader(content))
				MakeRequest(t, req, http.StatusUnauthorized)

				req = NewRequestWithBody(t, "PUT", url, bytes.NewReader(content)).
					AddBasicAuth(user.Name)
				MakeRequest(t, req, http.StatusCreated)

				pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeAlt)
				require.NoError(t, err)
				assert.Len(t, pvs, 1)

				pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
				require.NoError(t, err)
				assert.Nil(t, pd.SemVer)
				assert.IsType(t, &rpm_module.VersionMetadata{}, pd.Metadata)
				assert.Equal(t, packageName, pd.Package.Name)
				assert.Equal(t, packageVersion, pd.Version.Version)

				pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
				require.NoError(t, err)
				assert.Len(t, pfs, 1)
				assert.Equal(t, fmt.Sprintf("%s-%s.%s.rpm", packageName, packageVersion, packageArchitecture), pfs[0].Name)
				assert.True(t, pfs[0].IsLead)

				pb, err := packages.GetBlobByID(db.DefaultContext, pfs[0].BlobID)
				require.NoError(t, err)
				assert.Equal(t, int64(len(content)), pb.Size)

				req = NewRequestWithBody(t, "PUT", url, bytes.NewReader(content)).
					AddBasicAuth(user.Name)
				MakeRequest(t, req, http.StatusConflict)
			})

			t.Run("Download", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "GET", fmt.Sprintf("%s.repo/%s/RPMS.classic/%s-%s.%s.rpm", groupURL, packageArchitecture, packageName, packageVersion, packageArchitecture))
				resp := MakeRequest(t, req, http.StatusOK)

				assert.Equal(t, content, resp.Body.Bytes())
			})

			t.Run("Repository", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				url := fmt.Sprintf("%s.repo/%s/base", groupURL, packageArchitecture)

				req := NewRequest(t, "HEAD", url+"/dummy.xml")
				MakeRequest(t, req, http.StatusNotFound)

				req = NewRequest(t, "GET", url+"/dummy.xml")
				MakeRequest(t, req, http.StatusNotFound)

				t.Run("release.classic", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req = NewRequest(t, "HEAD", url+"/release.classic")
					MakeRequest(t, req, http.StatusOK)

					req = NewRequest(t, "GET", url+"/release.classic")
					resp := MakeRequest(t, req, http.StatusOK).Body.String()

					type ReleaseClassic struct {
						Archive      string
						Component    string
						Origin       string
						Label        string
						Architecture string
						NotAutomatic bool
					}

					var result ReleaseClassic

					lines := strings.Split(resp, "\n")

					for _, line := range lines {
						parts := strings.SplitN(line, ": ", 2)
						if len(parts) < 2 {
							continue
						}

						switch parts[0] {
						case "Archive":
							result.Archive = parts[1]
						case "Component":
							result.Component = parts[1]
						case "Origin":
							result.Origin = parts[1]
						case "Label":
							result.Label = parts[1]
						case "Architecture":
							result.Architecture = parts[1]
						case "NotAutomatic":
							notAuto, err := strconv.ParseBool(parts[1])
							if err != nil {
								require.NoError(t, err)
							}
							result.NotAutomatic = notAuto
						}
					}

					assert.Equal(t, "classic", result.Component)
					assert.Equal(t, "Alt Linux Team", result.Origin)
					assert.Equal(t, "Forgejo", result.Label)
					assert.Equal(t, "x86_64", result.Architecture)
					assert.False(t, result.NotAutomatic)
					assert.NotEmpty(t, result.Archive)
				})

				t.Run("release", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req = NewRequest(t, "HEAD", url+"/release")
					MakeRequest(t, req, http.StatusOK)

					req = NewRequest(t, "GET", url+"/release")
					resp := MakeRequest(t, req, http.StatusOK).Body.String()

					type Checksum struct {
						Hash string
						Size int
						File string
					}

					type Release struct {
						Origin        string
						Label         string
						Suite         string
						Architectures string
						MD5Sum        []Checksum
						BLAKE2B       []Checksum
					}

					var result Release

					lines := strings.Split(resp, "\n")

					var isMD5Sum, isBLAKE2b bool

					for _, line := range lines {
						line = strings.TrimSpace(line)

						if line == "" {
							continue
						}
						switch {
						case strings.HasPrefix(line, "Origin:"):
							result.Origin = strings.TrimSpace(strings.TrimPrefix(line, "Origin:"))
						case strings.HasPrefix(line, "Label:"):
							result.Label = strings.TrimSpace(strings.TrimPrefix(line, "Label:"))
						case strings.HasPrefix(line, "Suite:"):
							result.Suite = strings.TrimSpace(strings.TrimPrefix(line, "Suite:"))
						case strings.HasPrefix(line, "Architectures:"):
							result.Architectures = strings.TrimSpace(strings.TrimPrefix(line, "Architectures:"))
						case line == "MD5Sum:":
							isMD5Sum = true
							isBLAKE2b = false
						case line == "BLAKE2b:":
							isBLAKE2b = true
							isMD5Sum = false
						case isMD5Sum || isBLAKE2b:
							parts := strings.Fields(line)
							if len(parts) >= 3 {
								hash := parts[0]
								size, err := strconv.Atoi(parts[1])
								if err != nil {
									continue
								}
								file := parts[2]

								checksum := Checksum{
									Hash: hash,
									Size: size,
									File: file,
								}

								if isMD5Sum {
									result.MD5Sum = append(result.MD5Sum, checksum)
								} else if isBLAKE2b {
									result.BLAKE2B = append(result.BLAKE2B, checksum)
								}
							}
						}
					}

					assert.Equal(t, "Forgejo", result.Origin)
					assert.Equal(t, "Forgejo", result.Label)
					assert.Equal(t, "Sisyphus", result.Suite)
					assert.Equal(t, "x86_64", result.Architectures)

					assert.Len(t, result.MD5Sum, 3)
					assert.Equal(t, "bbf7ae6b2f540673ed1cfc0266b5f319", result.MD5Sum[0].Hash)
					assert.Equal(t, 1003, result.MD5Sum[0].Size)
					assert.Equal(t, "base/pkglist.classic", result.MD5Sum[0].File)

					assert.Len(t, result.BLAKE2B, 3)
					assert.Equal(t, "b527bf038895ce29107ec3a6d2eebd7c365e8ce5ab767276eeddd7c549a159025225cb0ecfdbf7b71da13db7e865e77bcb0e2dae4d21335df01a4a17e0056a70", result.BLAKE2B[0].Hash)
					assert.Equal(t, 1003, result.BLAKE2B[0].Size)
					assert.Equal(t, "base/pkglist.classic", result.BLAKE2B[0].File)
				})

				t.Run("pkglist.classic", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req = NewRequest(t, "GET", url+"/pkglist.classic")
					resp := MakeRequest(t, req, http.StatusOK)

					body := resp.Body
					defer body.Reset()

					type RpmHeader struct {
						Magic  [8]byte
						Nindex uint32
						Hsize  uint32
					}

					type RpmHdrIndex struct {
						Tag    uint32
						Type   uint32
						Offset uint32
						Count  uint32
					}

					type Metadata struct {
						Name                    string
						Version                 string
						Release                 string
						Summary                 []string
						Description             []string
						BuildTime               int
						Size                    int
						License                 string
						Packager                string
						Group                   []string
						URL                     string
						Arch                    string
						SourceRpm               string
						ProvideNames            []string
						RequireFlags            []int
						RequireNames            []string
						RequireVersions         []string
						ChangeLogTimes          []int
						ChangeLogNames          []string
						ChangeLogTexts          []string
						ProvideFlags            []int
						ProvideVersions         []string
						DirIndexes              []int
						BaseNames               []string
						DirNames                []string
						DistTag                 string
						AptIndexLegacyFileName  string
						AptIndexLegacyFileSize  int
						MD5Sum                  string
						BLAKE2B                 string
						AptIndexLegacyDirectory string
					}

					var result Metadata

					const rpmHeaderMagic = "\x8e\xad\xe8\x01\x00\x00\x00\x00"

					var hdr RpmHeader
					for {
						if err := binary.Read(body, binary.BigEndian, &hdr); err != nil {
							if err == io.EOF {
								break
							}
							require.NoError(t, err)
						}

						if !bytes.Equal(hdr.Magic[:], []byte(rpmHeaderMagic)) {
							require.NoError(t, err)
						}

						nindex := hdr.Nindex
						index := make([]RpmHdrIndex, nindex)
						if err := binary.Read(body, binary.BigEndian, &index); err != nil {
							require.NoError(t, err)
						}

						data := make([]byte, hdr.Hsize)
						if err := binary.Read(body, binary.BigEndian, &data); err != nil {
							require.NoError(t, err)
						}

						var indexPtrs []*RpmHdrIndex
						for i := range index {
							indexPtrs = append(indexPtrs, &index[i])
						}

						for _, idx := range indexPtrs {
							tag := binary.BigEndian.Uint32([]byte{byte(idx.Tag >> 24), byte(idx.Tag >> 16), byte(idx.Tag >> 8), byte(idx.Tag)})
							typ := binary.BigEndian.Uint32([]byte{byte(idx.Type >> 24), byte(idx.Type >> 16), byte(idx.Type >> 8), byte(idx.Type)})
							offset := binary.BigEndian.Uint32([]byte{byte(idx.Offset >> 24), byte(idx.Offset >> 16), byte(idx.Offset >> 8), byte(idx.Offset)})
							count := binary.BigEndian.Uint32([]byte{byte(idx.Count >> 24), byte(idx.Count >> 16), byte(idx.Count >> 8), byte(idx.Count)})

							if typ == 6 || typ == 8 || typ == 9 {
								elem := data[offset:]
								for j := uint32(0); j < count; j++ {
									strEnd := bytes.IndexByte(elem, 0)
									if strEnd == -1 {
										require.NoError(t, err)
									}
									switch tag {
									case 1000:
										result.Name = string(elem[:strEnd])
									case 1001:
										result.Version = string(elem[:strEnd])
									case 1002:
										result.Release = string(elem[:strEnd])
									case 1004:
										var summaries []string
										for i := uint32(0); i < count; i++ {
											summaries = append(summaries, string(elem[:strEnd]))
										}
										result.Summary = summaries
									case 1005:
										var descriptions []string
										for i := uint32(0); i < count; i++ {
											descriptions = append(descriptions, string(elem[:strEnd]))
										}
										result.Description = descriptions
									case 1014:
										result.License = string(elem[:strEnd])
									case 1015:
										result.Packager = string(elem[:strEnd])
									case 1016:
										var groups []string
										for i := uint32(0); i < count; i++ {
											groups = append(groups, string(elem[:strEnd]))
										}
										result.Group = groups
									case 1020:
										result.URL = string(elem[:strEnd])
									case 1022:
										result.Arch = string(elem[:strEnd])
									case 1044:
										result.SourceRpm = string(elem[:strEnd])
									case 1047:
										var provideNames []string
										for i := uint32(0); i < count; i++ {
											provideNames = append(provideNames, string(elem[:strEnd]))
										}
										result.ProvideNames = provideNames
									case 1049:
										var requireNames []string
										for i := uint32(0); i < count; i++ {
											requireNames = append(requireNames, string(elem[:strEnd]))
										}
										result.RequireNames = requireNames
									case 1050:
										var requireVersions []string
										for i := uint32(0); i < count; i++ {
											requireVersions = append(requireVersions, string(elem[:strEnd]))
										}
										result.RequireVersions = requireVersions
									case 1081:
										var changeLogNames []string
										for i := uint32(0); i < count; i++ {
											changeLogNames = append(changeLogNames, string(elem[:strEnd]))
										}
										result.ChangeLogNames = changeLogNames
									case 1082:
										var changeLogTexts []string
										for i := uint32(0); i < count; i++ {
											changeLogTexts = append(changeLogTexts, string(elem[:strEnd]))
										}
										result.ChangeLogTexts = changeLogTexts
									case 1113:
										var provideVersions []string
										for i := uint32(0); i < count; i++ {
											provideVersions = append(provideVersions, string(elem[:strEnd]))
										}
										result.ProvideVersions = provideVersions
									case 1117:
										var baseNames []string
										for i := uint32(0); i < count; i++ {
											baseNames = append(baseNames, string(elem[:strEnd]))
										}
										result.BaseNames = baseNames
									case 1118:
										var dirNames []string
										for i := uint32(0); i < count; i++ {
											dirNames = append(dirNames, string(elem[:strEnd]))
										}
										result.DirNames = dirNames
									case 1155:
										result.DistTag = string(elem[:strEnd])
									case 1000000:
										result.AptIndexLegacyFileName = string(elem[:strEnd])
									case 1000005:
										result.MD5Sum = string(elem[:strEnd])
									case 1000009:
										result.BLAKE2B = string(elem[:strEnd])
									case 1000010:
										result.AptIndexLegacyDirectory = string(elem[:strEnd])
									}
									elem = elem[strEnd+1:]
								}
							} else if typ == 4 {
								elem := data[offset:]
								for j := uint32(0); j < count; j++ {
									val := binary.BigEndian.Uint32(elem)
									switch tag {
									case 1006:
										result.BuildTime = int(val)
									case 1009:
										result.Size = int(val)
									case 1048:
										var requireFlags []int
										for i := uint32(0); i < count; i++ {
											requireFlags = append(requireFlags, int(val))
										}
										result.RequireFlags = requireFlags
									case 1080:
										var changeLogTimes []int
										for i := uint32(0); i < count; i++ {
											changeLogTimes = append(changeLogTimes, int(val))
										}
										result.ChangeLogTimes = changeLogTimes
									case 1112:
										var provideFlags []int
										for i := uint32(0); i < count; i++ {
											provideFlags = append(provideFlags, int(val))
										}
										result.ProvideFlags = provideFlags
									case 1116:
										var dirIndexes []int
										for i := uint32(0); i < count; i++ {
											dirIndexes = append(dirIndexes, int(val))
										}
										result.DirIndexes = dirIndexes
									case 1000001:
										result.AptIndexLegacyFileSize = int(val)
									}
									elem = elem[4:]
								}
							} else {
								require.NoError(t, err)
							}
						}
					}
					assert.Equal(t, "gitea-test", result.Name)
					assert.Equal(t, "1.0.2", result.Version)
					assert.Equal(t, "1", result.Release)
					assert.Equal(t, []string{"RPM package summary"}, result.Summary)
					assert.Equal(t, []string{"RPM package description"}, result.Description)
					assert.Equal(t, 1678225964, result.BuildTime)
					assert.Equal(t, 13, result.Size)
					assert.Equal(t, "MIT", result.License)
					assert.Equal(t, "KN4CK3R", result.Packager)
					assert.Equal(t, []string{"System"}, result.Group)
					assert.Equal(t, "https://gitea.io", result.URL)
					assert.Equal(t, "x86_64", result.Arch)
					assert.Equal(t, "gitea-test-1.0.2-1.src.rpm", result.SourceRpm)
					assert.Equal(t, []string{"", ""}, result.ProvideNames)
					assert.Equal(t, []int{16777226, 16777226, 16777226, 16777226, 16777226, 16777226, 16777226}, result.RequireFlags)
					assert.Equal(t, []string{"", "", "", "", "", "", ""}, result.RequireNames)
					assert.Equal(t, []string{"5.2-1", "5.2-1", "5.2-1", "5.2-1", "5.2-1", "5.2-1", "5.2-1"}, result.RequireVersions)
					assert.Equal(t, []int{1678276800}, result.ChangeLogTimes)
					assert.Equal(t, []string{"KN4CK3R <dummy@gitea.io>"}, result.ChangeLogNames)
					assert.Equal(t, []string{"- Changelog message."}, result.ChangeLogTexts)
					assert.Equal(t, []int{8, 8}, result.ProvideFlags)
					assert.Equal(t, []string{"1.0.2-1", "1.0.2-1"}, result.ProvideVersions)
					assert.Equal(t, []int(nil), result.DirIndexes)
					assert.Equal(t, []string{"hello"}, result.BaseNames)
					assert.Equal(t, []string{"/usr/local/bin/"}, result.DirNames)
					assert.Empty(t, result.DistTag)
					assert.Equal(t, "gitea-test-1.0.2-1.x86_64.rpm", result.AptIndexLegacyFileName)
					assert.Equal(t, 7116, result.AptIndexLegacyFileSize)
					assert.Equal(t, "9ea82dd62968719aea19c08cd2ced79a", result.MD5Sum)
					assert.Equal(t, "8ba7f1f52a47b23997aa2de21b305cc71974d51f0c54fb53cb927156284dafdcc233d514a46c020e4a0666e218529e0284933c5873d24c2555830d7627140f7d", result.BLAKE2B)
					assert.Equal(t, "RPMS.classic", result.AptIndexLegacyDirectory)
				})

				t.Run("pkglist.classic.xz", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", url+"/pkglist.classic.xz")
					pkglistXZResp := MakeRequest(t, req, http.StatusOK)
					pkglistXZ := pkglistXZResp.Body
					defer pkglistXZ.Reset()

					req2 := NewRequest(t, "GET", url+"/pkglist.classic")
					pkglistResp := MakeRequest(t, req2, http.StatusOK)
					pkglist := pkglistResp.Body
					defer pkglist.Reset()

					assert.Less(t, pkglistXZ.Len(), pkglist.Len())

					xzReader, err := xz.NewReader(pkglistXZ)
					require.NoError(t, err)

					var unxzData bytes.Buffer
					_, err = io.Copy(&unxzData, xzReader)
					require.NoError(t, err)

					assert.Equal(t, unxzData.Len(), pkglist.Len())

					content, _ := packages_module.NewHashedBuffer()
					defer content.Close()

					h := sha256.New()
					w := io.MultiWriter(content, h)

					_, err = io.Copy(w, pkglist)
					require.NoError(t, err)

					hashMD5Classic, _, hashSHA256Classic, _, hashBlake2bClassic := content.Sums()

					contentUnxz, _ := packages_module.NewHashedBuffer()
					defer contentUnxz.Close()

					_, err = io.Copy(io.MultiWriter(contentUnxz, sha256.New()), &unxzData)
					require.NoError(t, err)

					hashMD5Unxz, _, hashSHA256Unxz, _, hashBlake2bUnxz := contentUnxz.Sums()

					assert.Equal(t, fmt.Sprintf("%x", hashSHA256Classic), fmt.Sprintf("%x", hashSHA256Unxz))
					assert.Equal(t, fmt.Sprintf("%x", hashBlake2bClassic), fmt.Sprintf("%x", hashBlake2bUnxz))
					assert.Equal(t, fmt.Sprintf("%x", hashMD5Classic), fmt.Sprintf("%x", hashMD5Unxz))
				})
			})

			t.Run("Delete", func(t *testing.T) {
				defer tests.PrintCurrentTest(t)()

				req := NewRequest(t, "DELETE", fmt.Sprintf("%s.repo/%s/RPMS.classic/%s-%s.%s.rpm", groupURL, packageArchitecture, packageName, packageVersion, packageArchitecture))
				MakeRequest(t, req, http.StatusUnauthorized)

				req = NewRequest(t, "DELETE", fmt.Sprintf("%s.repo/%s/RPMS.classic/%s-%s.%s.rpm", groupURL, packageArchitecture, packageName, packageVersion, packageArchitecture)).
					AddBasicAuth(user.Name)
				MakeRequest(t, req, http.StatusNoContent)

				pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeAlt)
				require.NoError(t, err)
				assert.Empty(t, pvs)
				req = NewRequest(t, "DELETE", fmt.Sprintf("%s.repo/%s/RPMS.classic/%s-%s.%s.rpm", groupURL, packageArchitecture, packageName, packageVersion, packageArchitecture)).
					AddBasicAuth(user.Name)
				MakeRequest(t, req, http.StatusNotFound)
			})
		})
	}
}
