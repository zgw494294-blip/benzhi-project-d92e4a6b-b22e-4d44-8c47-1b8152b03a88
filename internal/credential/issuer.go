package credential

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

type Issuer struct {
	now func() time.Time
}

func NewIssuer(now func() time.Time) *Issuer {
	if now == nil {
		now = time.Now
	}
	return &Issuer{now: now}
}

func (i *Issuer) Issue(snapshot FrozenSnapshot, issuedBy string) (ReleaseCredential, error) {
	if strings.TrimSpace(issuedBy) == "" {
		return ReleaseCredential{}, errors.New("签发人不能为空")
	}
	digest, err := SnapshotDigest(snapshot)
	if err != nil {
		return ReleaseCredential{}, err
	}
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return ReleaseCredential{}, err
	}
	issuedAt := i.now().UTC()
	idSum := sha256.Sum256([]byte(snapshot.CaseID + digest + hex.EncodeToString(nonce)))
	credentialID := "cred_" + hex.EncodeToString(idSum[:12])
	codeSum := sha256.Sum256([]byte(credentialID + ":" + digest))
	return ReleaseCredential{
		CredentialID: credentialID, CaseID: snapshot.CaseID, CaseVersion: snapshot.CaseVersion,
		SnapshotDigest: digest, IssuedAt: issuedAt, IssuedBy: strings.TrimSpace(issuedBy),
		VerificationCode: hex.EncodeToString(codeSum[:16]),
	}, nil
}
