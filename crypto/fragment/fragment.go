// Package fragment generates shell-side crypto helper code used by the PHP and
// JSP adapter shell generators. A Fragment describes the crypto mode, a key
// fingerprint, and language-specific encrypt/decrypt snippets.
package fragment

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

// Fragment is a crypto implementation expressed as generated shell code.
type Fragment interface {
	// Name is the crypto mode, e.g. "aes-gcm".
	Name() string
	// KeyFingerprint is a stable identifier of the key material.
	KeyFingerprint() string
	// DecryptPHP and EncryptPHP return complete PHP helper functions.
	DecryptPHP() string
	EncryptPHP() string
	// DecryptJava and EncryptJava return complete Java helper methods.
	DecryptJava() string
	EncryptJava() string
}

// AESGCM generates AES-GCM helpers matching crypto/aesgcm's wire format:
// base64(nonce || ciphertext || auth tag) with a 12-byte nonce and 128-bit tag.
type AESGCM struct {
	Key []byte
}

// NewAESGCM validates the key length and returns an AESGCM fragment.
func NewAESGCM(key []byte) (*AESGCM, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("fragment: AES-GCM key must be 16, 24, or 32 bytes")
	}
	return &AESGCM{Key: key}, nil
}

// Name returns the crypto mode.
func (f *AESGCM) Name() string { return "aes-gcm" }

// KeyFingerprint returns the first 64 bits of SHA-256 over the key.
func (f *AESGCM) KeyFingerprint() string {
	sum := sha256.Sum256(f.Key)
	return hex.EncodeToString(sum[:8])
}

// DecryptPHP returns a PHP function that decrypts the base64 wire body.
func (f *AESGCM) DecryptPHP() string {
	return fmt.Sprintf(
		"function __rbs_decrypt($e){$d=base64_decode($e);if($d===false)return '';$iv=substr($d,0,12);$ct=substr($d,12,-16);$tag=substr($d,-16);$k=base64_decode(\"%s\");$p=@openssl_decrypt($ct,'aes-256-gcm',$k,OPENSSL_RAW_DATA,$iv,$tag);return ($p===false)?'':$p;}",
		base64.StdEncoding.EncodeToString(f.Key),
	)
}

// EncryptPHP returns a PHP function that encrypts a string into the base64
// wire body.
func (f *AESGCM) EncryptPHP() string {
	return fmt.Sprintf(
		"function __rbs_encrypt($s){$iv=random_bytes(12);$tag='';$k=base64_decode(\"%s\");$ct=@openssl_encrypt($s,'aes-256-gcm',$k,OPENSSL_RAW_DATA,$iv,$tag);return ($ct===false)?'':base64_encode($iv.$ct.$tag);}",
		base64.StdEncoding.EncodeToString(f.Key),
	)
}

// DecryptJava returns a Java method that decrypts the base64 wire body.
func (f *AESGCM) DecryptJava() string {
	return fmt.Sprintf(
		"String dec(String e)throws Exception{byte[] d=java.util.Base64.getDecoder().decode(e);byte[] n=java.util.Arrays.copyOfRange(d,0,12);javax.crypto.Cipher a=javax.crypto.Cipher.getInstance(\"AES/GCM/NoPadding\");a.init(javax.crypto.Cipher.DECRYPT_MODE,new javax.crypto.spec.SecretKeySpec(java.util.Base64.getDecoder().decode(\"%s\"),\"AES\"),new javax.crypto.spec.GCMParameterSpec(128,n));return new String(a.doFinal(java.util.Arrays.copyOfRange(d,12,d.length)),\"UTF-8\");}",
		base64.StdEncoding.EncodeToString(f.Key),
	)
}

// EncryptJava returns a Java method that encrypts a string into the base64
// wire body.
func (f *AESGCM) EncryptJava() string {
	return fmt.Sprintf(
		"String enc(String s)throws Exception{byte[] n=new byte[12];java.security.SecureRandom.getInstanceStrong().nextBytes(n);javax.crypto.Cipher a=javax.crypto.Cipher.getInstance(\"AES/GCM/NoPadding\");a.init(javax.crypto.Cipher.ENCRYPT_MODE,new javax.crypto.spec.SecretKeySpec(java.util.Base64.getDecoder().decode(\"%s\"),\"AES\"),new javax.crypto.spec.GCMParameterSpec(128,n));byte[] c=a.doFinal(s.getBytes(\"UTF-8\"));byte[] o=new byte[12+c.length];System.arraycopy(n,0,o,0,12);System.arraycopy(c,0,o,12,c.length);return java.util.Base64.getEncoder().encodeToString(o);}",
		base64.StdEncoding.EncodeToString(f.Key),
	)
}
