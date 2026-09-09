package relay

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"

	"github.com/emersion/go-msgauth/dkim"

	"dpdp-backend/internal/delivery"
	deliveryutils "dpdp-backend/internal/delivery/utils"
)

var signedHeaders = []string{
	"From",
	"To",
	"Cc",
	"Subject",
	"Date",
	"Message-ID",
	"MIME-Version",
	"Content-Type",
	"Content-Transfer-Encoding",
}

func Sign(raw []byte, cfg delivery.TenantConfig) ([]byte, error) {
	key, err := parsePrivateKey(cfg.DKIMPrivateKey)
	if err != nil {
		return nil, err
	}

	selector := cfg.DKIMSelector
	if selector == "" {
		selector = deliveryutils.DefaultDKIMSelector
	}

	options := &dkim.SignOptions{
		Domain:                 cfg.Domain,
		Selector:               selector,
		Signer:                 key,
		Hash:                   crypto.SHA256,
		HeaderCanonicalization: dkim.CanonicalizationRelaxed,
		BodyCanonicalization:   dkim.CanonicalizationRelaxed,
		HeaderKeys:             signedHeaders,
	}

	var signed bytes.Buffer

	if err := dkim.Sign(&signed, bytes.NewReader(raw), options); err != nil {
		return nil, err
	}

	return signed.Bytes(), nil
}

func parsePrivateKey(encoded string) (crypto.Signer, error) {
	if encoded == "" {
		return nil, deliveryutils.ErrPrivateKeyMissing
	}

	block, _ := pem.Decode([]byte(encoded))
	if block == nil {
		return nil, deliveryutils.ErrPrivateKeyInvalid
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		signer, ok := key.(crypto.Signer)
		if !ok {
			return nil, deliveryutils.ErrPrivateKeyInvalid
		}

		return signer, nil
	}

	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, deliveryutils.ErrPrivateKeyInvalid
	}

	return (*rsa.PrivateKey)(key), nil
}
