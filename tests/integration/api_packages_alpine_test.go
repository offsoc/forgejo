// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package integration

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"testing"

	"forgejo.org/models/db"
	"forgejo.org/models/packages"
	"forgejo.org/models/unittest"
	user_model "forgejo.org/models/user"
	alpine_module "forgejo.org/modules/packages/alpine"
	alpine_service "forgejo.org/services/packages/alpine"
	"forgejo.org/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPackageAlpine(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	packageName := "gitea-test"
	packageVersion := "1.4.1-r3"

	base64AlpinePackageContent := `H4sIAAAAAAACA9ML9nT30wsKdtTLzjNJzjYuckjPLElN1DUzMUxMNTa11CsqTtQrKE1ioAAYAIGZ
iQmYBgJ02hDENjQxMTAzMzQ1MTVjMDA0MTQ1ZlAwYKADKC0uSSxSUGAYoWDm4sZZtypv75+q2fVT
POD1bKkFB22ms+g1z+H4dk7AhC3HwUSj9EbT0Rk3Dn55dHxy/K7Q+Nl/i+L7Z036ypcRvvpZuMiN
s7wbZL/klqRGGshv9Gi0qHTgTZfw3HytnJdx9c3NTRp/PHn+Z50uq2pjkilzjtpfd+uzQMw1M7cY
i9RXJasnT2M+vDXCesLK7MilJt8sGplj4xUlLMUun9SzY+phFpxWxRXa06AseV9WvzH3jtGGoL5A
vQkea+VKPj5R+Cb461tIk97qpa9nJYsJujTNl2B/J1P52H/D2rPr/j19uU8p7cMSq5tmXk51ReXl
F/Yddr9XsMpEwFKlXSPo3QSGwnCOG8y2uadjm6ui998WYXNYubjg78N3a7bnXjhrl5fB8voI++LI
1FP5W44e2xf4Ou2wrtyic1Onz7MzMV5ksuno2V/LVG4eN/15X/n2/2vJ2VV+T68aT327dOrhd6e6
q5Y0V82Y83tdqkFa8TW2BvGCZ0ds/iibHVpzKuPcuSULO63/bNmfrnhjWqXzhMSXTb5Cv4vPaxSL
8LFMdqmxbN7+Y+Yi0ZyZhz4UxexLuHHFd1VFvk+kwvniq3P+f9rh52InWnL8Lpvedcecoh1GFSc5
xZ9VBGex2V269HZfwxSVCvP35wQfi2xKX+lYMXtF48n1R65O2PLWpm69RdESMa79dlrTGazsZacu
MbMLeSSScPORZde76/MBV6SFJAAEAAAfiwgAAAAAAAID7VRLaxsxEN6zfoUgZ++OVq+1aUIhUDeY
pKa49FhmJdkW3ofRysXpr69220t9SCk0gZJ+IGaY56eBmbxY4/m9Q+vCUOTr1fLu4d2H7O8CEpQQ
k0y4lAClypgQoBSTQqoMGBMgMnrOXgCnIWJIVLLXCcaoib5110CSij/V7D9eCZ5p5f9o/5VkF/tf
MqUzCi+5/6Hv41Nxv/Nffu4fwRVdus4FjM7S+pFiffKNpTxnkMMsALmin5PnHgMtS8rkgvGFBPpp
c0tLKDk5HnYdto5e052PDmfRDXE0fnUh2VgucjYLU5h1g0mm5RhGNymMrtEccOfIKTTJsY/xOCyK
YqqT+74gExWbmI2VlJ6LeQUcyPFH2lh/9SBuV/wjfXPohDnw8HZKviGD/zYmCZgrgsHsk36u1Bcl
SB/8zne/0jV92/qYbKRF38X0niiemN2QxhvXDWOL+7tNGhGeYt+m22mwaR6pddGZNM8FSeRxj8PY
X7PaqdqAVlqWXHKnmQGmK43VlqNlILRilbBSMI2jV5Vbu5XGSVsDyGc7yd8B/gK2qgAIAAAfiwgA
AAAAAAID7dNNSgMxGAbg7MSCOxcu5wJOv0x+OlkU7K5QoYXqVsxMMihlKMwP1Fu48QQewCN4DfEQ
egUz4sYuFKEtFN9n870hWSSQN+7P7GrsrfNV3Y9dW5Z3bNMo0FJ+zmB9EhcJ41KS1lxJpRnxbsWi
FduBtm5sFa7C/ifOo7y5Lf2QeiHar6jTaDSbnF5Mp+fzOL/x+aJuy3g+HvGhs8JY4b3yOpMZOZEo
lRW+MEoTTw3ZwqU0INNjsAe2VPk/9b/L3/s/kIKzqOtk+IbJGTtmr+bx7WoxOUoun98frk/un14O
Djfa/2q5bH4699v++uMAAAAAAAAAAAAAAAAAAAAAAHbgA/eXQh8AKAAA`
	content, err := base64.StdEncoding.DecodeString(base64AlpinePackageContent)
	require.NoError(t, err)

	branches := []string{"v3.16", "v3.17", "v3.18"}
	repositories := []string{"main", "testing"}

	rootURL := fmt.Sprintf("/api/packages/%s/alpine", user.Name)

	t.Run("RepositoryKey", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", rootURL+"/key")
		resp := MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, "application/x-pem-file", resp.Header().Get("Content-Type"))
		assert.Contains(t, resp.Body.String(), "-----BEGIN PUBLIC KEY-----")
	})

	for _, branch := range branches {
		for _, repository := range repositories {
			t.Run(fmt.Sprintf("[Branch:%s,Repository:%s]", branch, repository), func(t *testing.T) {
				t.Run("Upload", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					uploadURL := fmt.Sprintf("%s/%s/%s", rootURL, branch, repository)

					req := NewRequestWithBody(t, "PUT", uploadURL, bytes.NewReader([]byte{}))
					MakeRequest(t, req, http.StatusUnauthorized)

					req = NewRequestWithBody(t, "PUT", uploadURL, bytes.NewReader([]byte{})).
						AddBasicAuth(user.Name)
					MakeRequest(t, req, http.StatusBadRequest)

					req = NewRequestWithBody(t, "PUT", uploadURL, bytes.NewReader(content)).
						AddBasicAuth(user.Name)
					MakeRequest(t, req, http.StatusCreated)

					pvs, err := packages.GetVersionsByPackageType(db.DefaultContext, user.ID, packages.TypeAlpine)
					require.NoError(t, err)
					assert.Len(t, pvs, 1)

					pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
					require.NoError(t, err)
					assert.Nil(t, pd.SemVer)
					assert.IsType(t, &alpine_module.VersionMetadata{}, pd.Metadata)
					assert.Equal(t, packageName, pd.Package.Name)
					assert.Equal(t, packageVersion, pd.Version.Version)

					pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
					require.NoError(t, err)
					assert.NotEmpty(t, pfs)
					assert.Condition(t, func() bool {
						seen := false
						expectedFilename := fmt.Sprintf("%s-%s.apk", packageName, packageVersion)
						expectedCompositeKey := fmt.Sprintf("%s|%s|x86_64", branch, repository)
						for _, pf := range pfs {
							if pf.Name == expectedFilename && pf.CompositeKey == expectedCompositeKey {
								if seen {
									return false
								}
								seen = true

								assert.True(t, pf.IsLead)

								pfps, err := packages.GetProperties(db.DefaultContext, packages.PropertyTypeFile, pf.ID)
								require.NoError(t, err)

								for _, pfp := range pfps {
									switch pfp.Name {
									case alpine_module.PropertyBranch:
										assert.Equal(t, branch, pfp.Value)
									case alpine_module.PropertyRepository:
										assert.Equal(t, repository, pfp.Value)
									case alpine_module.PropertyArchitecture:
										assert.Equal(t, "x86_64", pfp.Value)
									}
								}
							}
						}
						return seen
					})
				})

				readIndexContent := func(r io.Reader) (string, error) {
					br := bufio.NewReader(r)

					gzr, err := gzip.NewReader(br)
					if err != nil {
						return "", err
					}

					for {
						gzr.Multistream(false)

						tr := tar.NewReader(gzr)
						for {
							hd, err := tr.Next()
							if err == io.EOF {
								break
							}
							if err != nil {
								return "", err
							}

							if hd.Name == alpine_service.IndexFilename {
								buf, err := io.ReadAll(tr)
								if err != nil {
									return "", err
								}

								return string(buf), nil
							}
						}

						err = gzr.Reset(br)
						if err == io.EOF {
							break
						}
						if err != nil {
							return "", err
						}
					}

					return "", io.EOF
				}

				t.Run("Index", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					url := fmt.Sprintf("%s/%s/%s/x86_64/APKINDEX.tar.gz", rootURL, branch, repository)

					req := NewRequest(t, "GET", url)
					resp := MakeRequest(t, req, http.StatusOK)

					content, err := readIndexContent(resp.Body)
					require.NoError(t, err)

					assert.Contains(t, content, "C:Q1/se1PjO94hYXbfpNR1/61hVORIc=\n")
					assert.Contains(t, content, "P:"+packageName+"\n")
					assert.Contains(t, content, "V:"+packageVersion+"\n")
					assert.Contains(t, content, "A:x86_64\n")
					assert.NotContains(t, content, "A:noarch\n")
					assert.Contains(t, content, "T:Gitea Test Package\n")
					assert.Contains(t, content, "U:https://gitea.io/\n")
					assert.Contains(t, content, "L:MIT\n")
					assert.Contains(t, content, "S:1353\n")
					assert.Contains(t, content, "I:4096\n")
					assert.Contains(t, content, "o:gitea-test\n")
					assert.Contains(t, content, "m:KN4CK3R <kn4ck3r@gitea.io>\n")
					assert.Contains(t, content, "t:1679498030\n")
				})

				t.Run("Download", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", fmt.Sprintf("%s/%s/%s/x86_64/%s-%s.apk", rootURL, branch, repository, packageName, packageVersion))
					MakeRequest(t, req, http.StatusOK)
				})
			})
		}
	}

	t.Run("Delete", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		for _, branch := range branches {
			for _, repository := range repositories {
				req := NewRequest(t, "DELETE", fmt.Sprintf("%s/%s/%s/x86_64/%s-%s.apk", rootURL, branch, repository, packageName, packageVersion))
				MakeRequest(t, req, http.StatusUnauthorized)

				req = NewRequest(t, "DELETE", fmt.Sprintf("%s/%s/%s/x86_64/%s-%s.apk", rootURL, branch, repository, packageName, packageVersion)).
					AddBasicAuth(user.Name)
				MakeRequest(t, req, http.StatusNoContent)

				// Deleting the last file of an architecture should remove that index
				req = NewRequest(t, "GET", fmt.Sprintf("%s/%s/%s/x86_64/APKINDEX.tar.gz", rootURL, branch, repository))
				MakeRequest(t, req, http.StatusNotFound)
			}
		}
	})
}

func TestPackageAlpineNoArch(t *testing.T) {
	defer tests.PrepareTestEnv(t)()

	user := unittest.AssertExistsAndLoadBean(t, &user_model.User{ID: 2})

	packageNames := []string{"forgejo-noarch-test", "forgejo-noarch-test-openrc"}
	packageVersion := "1.0.0-r0"

	base64AlpinePackageContent := `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
MoxKyDAAAAAAAAAAAADA2noDOINxQwAoAAA=`
	content, err := base64.StdEncoding.DecodeString(base64AlpinePackageContent)
	require.NoError(t, err)

	packageContents := map[string][]byte{}
	packageContents["forgejo-noarch-test"] = content

	base64AlpinePackageContent = `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`
	content, err = base64.StdEncoding.DecodeString(base64AlpinePackageContent)
	require.NoError(t, err)

	packageContents["forgejo-noarch-test-openrc"] = content

	branches := []string{"v3.16", "v3.17", "v3.18"}
	repositories := []string{"main", "testing"}

	rootURL := fmt.Sprintf("/api/packages/%s/alpine", user.Name)

	t.Run("RepositoryKey", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		req := NewRequest(t, "GET", rootURL+"/key")
		resp := MakeRequest(t, req, http.StatusOK)

		assert.Equal(t, "application/x-pem-file", resp.Header().Get("Content-Type"))
		assert.Contains(t, resp.Body.String(), "-----BEGIN PUBLIC KEY-----")
	})

	for _, branch := range branches {
		for _, repository := range repositories {
			t.Run(fmt.Sprintf("[Branch:%s,Repository:%s]", branch, repository), func(t *testing.T) {
				for _, pkg := range packageNames {
					t.Run(fmt.Sprintf("Upload[Package:%s]", pkg), func(t *testing.T) {
						defer tests.PrintCurrentTest(t)()

						uploadURL := fmt.Sprintf("%s/%s/%s", rootURL, branch, repository)

						req := NewRequestWithBody(t, "PUT", uploadURL, bytes.NewReader([]byte{}))
						MakeRequest(t, req, http.StatusUnauthorized)

						req = NewRequestWithBody(t, "PUT", uploadURL, bytes.NewReader([]byte{})).
							AddBasicAuth(user.Name)
						MakeRequest(t, req, http.StatusBadRequest)

						req = NewRequestWithBody(t, "PUT", uploadURL, bytes.NewReader(packageContents[pkg])).
							AddBasicAuth(user.Name)
						MakeRequest(t, req, http.StatusCreated)

						pvs, err := packages.GetVersionsByPackageName(db.DefaultContext, user.ID, packages.TypeAlpine, pkg)
						require.NoError(t, err)
						assert.Len(t, pvs, 1)

						pd, err := packages.GetPackageDescriptor(db.DefaultContext, pvs[0])
						require.NoError(t, err)
						assert.Nil(t, pd.SemVer)
						assert.IsType(t, &alpine_module.VersionMetadata{}, pd.Metadata)
						assert.Equal(t, pkg, pd.Package.Name)
						assert.Equal(t, packageVersion, pd.Version.Version)

						pfs, err := packages.GetFilesByVersionID(db.DefaultContext, pvs[0].ID)
						require.NoError(t, err)
						assert.NotEmpty(t, pfs)
						assert.Condition(t, func() bool {
							seen := false
							expectedFilename := fmt.Sprintf("%s-%s.apk", pkg, packageVersion)
							expectedCompositeKey := fmt.Sprintf("%s|%s|x86_64", branch, repository)
							for _, pf := range pfs {
								if pf.Name == expectedFilename && pf.CompositeKey == expectedCompositeKey {
									if seen {
										return false
									}
									seen = true

									assert.True(t, pf.IsLead)

									pfps, err := packages.GetProperties(db.DefaultContext, packages.PropertyTypeFile, pf.ID)
									require.NoError(t, err)

									for _, pfp := range pfps {
										switch pfp.Name {
										case alpine_module.PropertyBranch:
											assert.Equal(t, branch, pfp.Value)
										case alpine_module.PropertyRepository:
											assert.Equal(t, repository, pfp.Value)
										case alpine_module.PropertyArchitecture:
											assert.Equal(t, "x86_64", pfp.Value)
										}
									}
								}
							}
							return seen
						})
					})
				}

				t.Run("Index", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					url := fmt.Sprintf("%s/%s/%s/x86_64/APKINDEX.tar.gz", rootURL, branch, repository)

					req := NewRequest(t, "GET", url)
					resp := MakeRequest(t, req, http.StatusOK)

					assert.Condition(t, func() bool {
						br := bufio.NewReader(resp.Body)

						gzr, err := gzip.NewReader(br)
						require.NoError(t, err)

						for {
							gzr.Multistream(false)

							tr := tar.NewReader(gzr)
							for {
								hd, err := tr.Next()
								if err == io.EOF {
									break
								}
								require.NoError(t, err)

								if hd.Name == "APKINDEX" {
									buf, err := io.ReadAll(tr)
									require.NoError(t, err)

									s := string(buf)

									assert.Contains(t, s, "C:Q14rbX8G4tErQO98k5J4uHsNaoiqk=\n")
									assert.Contains(t, s, "P:"+packageNames[0]+"\n")
									assert.Contains(t, s, "V:"+packageVersion+"\n")
									assert.Contains(t, s, "A:x86_64\n")
									assert.Contains(t, s, "T:Forgejo #2173 reproduction\n")
									assert.Contains(t, s, "U:https://forgejo.org\n")
									assert.Contains(t, s, "L:GPLv3\n")
									assert.Contains(t, s, "S:1508\n")
									assert.Contains(t, s, "I:20480\n")
									assert.Contains(t, s, "o:forgejo-noarch-test\n")
									assert.Contains(t, s, "m:Alexandre Almeida <git@aoalmeida.com>\n")
									assert.Contains(t, s, "t:1707660311\n")
									assert.Contains(t, s, "p:cmd:forgejo_2173=1.0.0-r0")

									assert.Contains(t, s, "C:Q1zTXZP03UbSled31mi4MXmsrgNQ4=\n")
									assert.Contains(t, s, "P:"+packageNames[1]+"\n")
									assert.Contains(t, s, "V:"+packageVersion+"\n")
									assert.Contains(t, s, "A:x86_64\n")
									assert.Contains(t, s, "T:Forgejo #2173 reproduction (OpenRC init scripts)\n")
									assert.Contains(t, s, "U:https://forgejo.org\n")
									assert.Contains(t, s, "L:GPLv3\n")
									assert.Contains(t, s, "S:1569\n")
									assert.Contains(t, s, "I:16384\n")
									assert.Contains(t, s, "o:forgejo-noarch-test\n")
									assert.Contains(t, s, "m:Alexandre Almeida <git@aoalmeida.com>\n")
									assert.Contains(t, s, "t:1707660311\n")
									assert.Contains(t, s, "i:openrc forgejo-noarch-test=1.0.0-r0")

									return true
								}
							}

							err = gzr.Reset(br)
							if err == io.EOF {
								break
							}
							require.NoError(t, err)
						}

						return false
					})
				})

				t.Run("Download", func(t *testing.T) {
					defer tests.PrintCurrentTest(t)()

					req := NewRequest(t, "GET", fmt.Sprintf("%s/%s/%s/x86_64/%s-%s.apk", rootURL, branch, repository, packageNames[0], packageVersion))
					MakeRequest(t, req, http.StatusOK)
				})
			})
		}
	}

	t.Run("Delete", func(t *testing.T) {
		defer tests.PrintCurrentTest(t)()

		for _, branch := range branches {
			for _, repository := range repositories {
				for _, pkg := range packageNames {
					req := NewRequest(t, "DELETE", fmt.Sprintf("%s/%s/%s/x86_64/%s-%s.apk", rootURL, branch, repository, pkg, packageVersion))
					MakeRequest(t, req, http.StatusUnauthorized)

					req = NewRequest(t, "DELETE", fmt.Sprintf("%s/%s/%s/x86_64/%s-%s.apk", rootURL, branch, repository, pkg, packageVersion)).
						AddBasicAuth(user.Name)
					MakeRequest(t, req, http.StatusNoContent)
				}
				// Deleting the last file of an architecture should remove that index
				req := NewRequest(t, "GET", fmt.Sprintf("%s/%s/%s/x86_64/APKINDEX.tar.gz", rootURL, branch, repository))
				MakeRequest(t, req, http.StatusNotFound)
			}
		}
	})
}
