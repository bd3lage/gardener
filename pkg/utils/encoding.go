// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"slices"
	"strconv"
)

// EncodeBase64 takes a byte slice and returns the Base64-encoded string.
func EncodeBase64(in []byte) string {
	return base64.StdEncoding.EncodeToString(in)
}

// DecodeBase64 takes a Base64-encoded string and returns the decoded byte slice.
func DecodeBase64(in string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(in)
}

// EncodePrivateKey takes a RSA private key object, encodes it to the PEM format, and returns it as
// a byte slice.
func EncodePrivateKey(key *rsa.PrivateKey) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
}

// EncodePrivateKeyInPKCS8 takes a RSA private key object, encodes it to the PKCS8 format, and returns it as
// a byte slice.
func EncodePrivateKeyInPKCS8(key *rsa.PrivateKey) ([]byte, error) {
	bytes, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: bytes,
	}), nil
}

// DecodeRSAPrivateKeyFromPKCS8 takes a byte slice, decodes it from the PKCS8 format, tries to convert it
// to an rsa.PrivateKey object, and returns it. In case an error occurs, it returns the error.
func DecodeRSAPrivateKeyFromPKCS8(bytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("could not decode the PEM-encoded RSA private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("the decoded key is not an RSA private key")
	}
	return rsaKey, nil
}

// DecodePrivateKey takes a byte slice, decodes it from the PEM format, converts it to an rsa.PrivateKey
// object, and returns it. In case an error occurs, it returns the error.
func DecodePrivateKey(bytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, errors.New("could not decode the PEM-encoded RSA private key")
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// EncodeCertificate takes a certificate as a byte slice, encodes it to the PEM format, and returns
// it as byte slice.
func EncodeCertificate(certificate []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificate,
	})
}

// DecodeCertificate takes a byte slice, decodes it from the PEM format, converts it to an x509.Certificate
// object, and returns it. In case an error occurs, it returns the error.
func DecodeCertificate(bytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(bytes)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("PEM block type must be CERTIFICATE")
	}
	return x509.ParseCertificate(block.Bytes)
}

// DecodeCertificateRequest parses the given PEM-encoded CSR.
func DecodeCertificateRequest(data []byte) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		return nil, errors.New("PEM block type must be CERTIFICATE REQUEST")
	}
	return x509.ParseCertificateRequest(block.Bytes)
}

// SHA1 takes a byte slice and returns the sha1-hashed byte slice.
func SHA1(in []byte) []byte {
	s := sha1.New()
	_, _ = s.Write(in)
	return s.Sum(nil)
}

// SHA256 takes a byte slice and returns the sha256-hashed byte slice.
func SHA256(in []byte) []byte {
	h := sha256.Sum256(in)
	return h[:]
}

// EncodeSHA1 takes a byte slice and returns the sha1-hashed string (base64-encoded).
func EncodeSHA1(in []byte) string {
	return EncodeBase64(SHA1(in))
}

// CreateSHA1Secret takes a username and a password and returns a sha1-schemed credentials pair as bytes.
func CreateSHA1Secret(username, password []byte) []byte {
	credentials := append(username, ":{SHA}"...)
	credentials = append(credentials, EncodeSHA1(password)...)
	return credentials
}

// ComputeSHA1Hex computes the hexadecimal representation of the SHA1 hash of the given input byte
// slice <in>, converts it to a string and returns it (length of returned string is 40 characters).
func ComputeSHA1Hex(in []byte) string {
	return hex.EncodeToString(SHA1(in))
}

// ComputeSHA256Hex computes the hexadecimal representation of the SHA256 hash of the given input byte
// slice <in>, converts it to a string and returns it.
func ComputeSHA256Hex(in []byte) string {
	return hex.EncodeToString(SHA256(in))
}

// HashForMap creates a hash value for a map of type map[string]any and returns it.
func HashForMap(m map[string]any) string {
	var hash string
	keys := make([]string, 0, len(m))

	for k := range m {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	for _, k := range keys {
		switch v := m[k].(type) {
		case string:
			hash += ComputeSHA256Hex([]byte(v))
		case int:
			hash += ComputeSHA256Hex([]byte(strconv.Itoa(v)))
		case bool:
			hash += ComputeSHA256Hex([]byte(strconv.FormatBool(v)))
		case []string:
			for _, val := range v {
				hash += ComputeSHA256Hex([]byte(val))
			}
		case map[string]any:
			hash += HashForMap(v)
		case []map[string]any:
			for _, val := range v {
				hash += HashForMap(val)
			}
		}
	}

	return ComputeSHA256Hex([]byte(hash))
}
