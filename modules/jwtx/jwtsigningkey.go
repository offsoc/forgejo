// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package jwtx

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"code.gitea.io/gitea/modules/log"
	"code.gitea.io/gitea/modules/util"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidAlgorithmType represents an invalid algorithm error.
type ErrInvalidAlgorithmType struct {
	Algorithm string
}

func (err ErrInvalidAlgorithmType) Error() string {
	return fmt.Sprintf("JWT signing algorithm is not supported: %s", err.Algorithm)
}

// JWTSigningKey represents a algorithm/key pair to sign JWTs
type JWTSigningKey interface {
	IsSymmetric() bool
	SigningMethod() jwt.SigningMethod
	SignKey() any
	VerifyKey() any
	ToJWK() (map[string]string, error)
	KID() string
	PreProcessToken(*jwt.Token)
}

type hmacSigningKey struct {
	signingMethod jwt.SigningMethod
	secret        []byte
}

func (key hmacSigningKey) IsSymmetric() bool {
	return true
}

func (key hmacSigningKey) SigningMethod() jwt.SigningMethod {
	return key.signingMethod
}

func (key hmacSigningKey) SignKey() any {
	return key.secret
}

func (key hmacSigningKey) VerifyKey() any {
	return key.secret
}

func (key hmacSigningKey) ToJWK() (map[string]string, error) {
	return map[string]string{
		"kty": "oct",
		"alg": key.SigningMethod().Alg(),
	}, nil
}

func (key hmacSigningKey) KID() string {
	return ""
}

func (key hmacSigningKey) PreProcessToken(*jwt.Token) {}

type rsaSigningKey struct {
	signingMethod jwt.SigningMethod
	key           *rsa.PrivateKey
	id            string
}

func newRSASigningKey(signingMethod jwt.SigningMethod, key *rsa.PrivateKey) (rsaSigningKey, error) {
	kid, err := util.CreatePublicKeyFingerprint(key.Public().(*rsa.PublicKey))
	if err != nil {
		return rsaSigningKey{}, err
	}

	return rsaSigningKey{
		signingMethod,
		key,
		base64.RawURLEncoding.EncodeToString(kid),
	}, nil
}

func (key rsaSigningKey) IsSymmetric() bool {
	return false
}

func (key rsaSigningKey) SigningMethod() jwt.SigningMethod {
	return key.signingMethod
}

func (key rsaSigningKey) SignKey() any {
	return key.key
}

func (key rsaSigningKey) VerifyKey() any {
	return key.key.Public()
}

func (key rsaSigningKey) ToJWK() (map[string]string, error) {
	pubKey := key.key.Public().(*rsa.PublicKey)

	return map[string]string{
		"kty": "RSA",
		"alg": key.SigningMethod().Alg(),
		"kid": key.id,
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pubKey.E)).Bytes()),
		"n":   base64.RawURLEncoding.EncodeToString(pubKey.N.Bytes()),
	}, nil
}

func (key rsaSigningKey) KID() string {
	return key.id
}

func (key rsaSigningKey) PreProcessToken(token *jwt.Token) {
	token.Header["kid"] = key.id
}

type eddsaSigningKey struct {
	signingMethod jwt.SigningMethod
	key           ed25519.PrivateKey
	id            string
}

func newEdDSASigningKey(signingMethod jwt.SigningMethod, key ed25519.PrivateKey) (eddsaSigningKey, error) {
	kid, err := util.CreatePublicKeyFingerprint(key.Public().(ed25519.PublicKey))
	if err != nil {
		return eddsaSigningKey{}, err
	}

	return eddsaSigningKey{
		signingMethod,
		key,
		base64.RawURLEncoding.EncodeToString(kid),
	}, nil
}

func (key eddsaSigningKey) IsSymmetric() bool {
	return false
}

func (key eddsaSigningKey) SigningMethod() jwt.SigningMethod {
	return key.signingMethod
}

func (key eddsaSigningKey) SignKey() any {
	return key.key
}

func (key eddsaSigningKey) VerifyKey() any {
	return key.key.Public()
}

func (key eddsaSigningKey) ToJWK() (map[string]string, error) {
	pubKey := key.key.Public().(ed25519.PublicKey)

	return map[string]string{
		"alg": key.SigningMethod().Alg(),
		"kid": key.id,
		"kty": "OKP",
		"crv": "Ed25519",
		"x":   base64.RawURLEncoding.EncodeToString(pubKey),
	}, nil
}

func (key eddsaSigningKey) KID() string {
	return key.id
}

func (key eddsaSigningKey) PreProcessToken(token *jwt.Token) {
	token.Header["kid"] = key.id
}

type ecdsaSigningKey struct {
	signingMethod jwt.SigningMethod
	key           *ecdsa.PrivateKey
	id            string
}

func newECDSASigningKey(signingMethod jwt.SigningMethod, key *ecdsa.PrivateKey) (ecdsaSigningKey, error) {
	kid, err := util.CreatePublicKeyFingerprint(key.Public().(*ecdsa.PublicKey))
	if err != nil {
		return ecdsaSigningKey{}, err
	}

	return ecdsaSigningKey{
		signingMethod,
		key,
		base64.RawURLEncoding.EncodeToString(kid),
	}, nil
}

func (key ecdsaSigningKey) IsSymmetric() bool {
	return false
}

func (key ecdsaSigningKey) SigningMethod() jwt.SigningMethod {
	return key.signingMethod
}

func (key ecdsaSigningKey) SignKey() any {
	return key.key
}

func (key ecdsaSigningKey) VerifyKey() any {
	return key.key.Public()
}

func (key ecdsaSigningKey) ToJWK() (map[string]string, error) {
	pubKey := key.key.Public().(*ecdsa.PublicKey)

	return map[string]string{
		"kty": "EC",
		"alg": key.SigningMethod().Alg(),
		"kid": key.id,
		"crv": pubKey.Params().Name,
		"x":   base64.RawURLEncoding.EncodeToString(pubKey.X.Bytes()),
		"y":   base64.RawURLEncoding.EncodeToString(pubKey.Y.Bytes()),
	}, nil
}

func (key ecdsaSigningKey) KID() string {
	return key.id
}

func (key ecdsaSigningKey) PreProcessToken(token *jwt.Token) {
	token.Header["kid"] = key.id
}

// CreateJWTSigningKey creates a signing key from an algorithm / key pair.
func CreateJWTSigningKey(algorithm string, key any) (JWTSigningKey, error) {
	var signingMethod jwt.SigningMethod
	switch algorithm {
	case "HS256":
		signingMethod = jwt.SigningMethodHS256
	case "HS384":
		signingMethod = jwt.SigningMethodHS384
	case "HS512":
		signingMethod = jwt.SigningMethodHS512

	case "RS256":
		signingMethod = jwt.SigningMethodRS256
	case "RS384":
		signingMethod = jwt.SigningMethodRS384
	case "RS512":
		signingMethod = jwt.SigningMethodRS512

	case "ES256":
		signingMethod = jwt.SigningMethodES256
	case "ES384":
		signingMethod = jwt.SigningMethodES384
	case "ES512":
		signingMethod = jwt.SigningMethodES512
	case "EdDSA":
		signingMethod = jwt.SigningMethodEdDSA
	default:
		return nil, ErrInvalidAlgorithmType{algorithm}
	}

	switch signingMethod.(type) {
	case *jwt.SigningMethodEd25519:
		privateKey, ok := key.(ed25519.PrivateKey)
		if !ok {
			return nil, jwt.ErrInvalidKeyType
		}
		return newEdDSASigningKey(signingMethod, privateKey)
	case *jwt.SigningMethodECDSA:
		privateKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, jwt.ErrInvalidKeyType
		}
		return newECDSASigningKey(signingMethod, privateKey)
	case *jwt.SigningMethodRSA:
		privateKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, jwt.ErrInvalidKeyType
		}
		return newRSASigningKey(signingMethod, privateKey)
	default:
		secret, ok := key.([]byte)
		if !ok {
			return nil, jwt.ErrInvalidKeyType
		}
		return hmacSigningKey{signingMethod, secret}, nil
	}
}

// LoadOrCreateAsymmetricKey checks if the configured private key exists.
// If it does not exist a new random key gets generated and saved on the configured path.
func LoadOrCreateAsymmetricKey(keyPath, algo string) (any, error) {
	isExist, err := util.IsExist(keyPath)
	if err != nil {
		log.Fatal("Unable to check if %s exists. Error: %v", keyPath, err)
	}
	if !isExist {
		err := func() error {
			key, err := func() (any, error) {
				switch {
				case strings.HasPrefix(algo, "RS"):
					return rsa.GenerateKey(rand.Reader, 4096)
				case algo == "EdDSA":
					_, pk, err := ed25519.GenerateKey(rand.Reader)
					return pk, err
				default:
					return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				}
			}()
			if err != nil {
				return err
			}

			bytes, err := x509.MarshalPKCS8PrivateKey(key)
			if err != nil {
				return err
			}

			privateKeyPEM := &pem.Block{Type: "PRIVATE KEY", Bytes: bytes}

			if err := os.MkdirAll(filepath.Dir(keyPath), os.ModePerm); err != nil {
				return err
			}

			f, err := os.OpenFile(keyPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o600)
			if err != nil {
				return err
			}
			defer func() {
				if err = f.Close(); err != nil {
					log.Error("Close: %v", err)
				}
			}()

			return pem.Encode(f, privateKeyPEM)
		}()
		if err != nil {
			log.Fatal("Error generating private key: %v", err)
			return nil, err
		}
	}

	bytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, fmt.Errorf("no valid PEM data found in %s", keyPath)
	} else if block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("expected PRIVATE KEY, got %s in %s", block.Type, keyPath)
	}

	return x509.ParsePKCS8PrivateKey(block.Bytes)
}
