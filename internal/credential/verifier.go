package credential

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

func Verify(value ReleaseCredential, snapshot FrozenSnapshot, suppliedCode string) Verification {
	if value.CredentialID == "" || value.CaseID == "" || value.SnapshotDigest == "" {
		return Verification{Valid: false, Reason: "凭据字段不完整"}
	}
	if value.CaseID != snapshot.CaseID || value.CaseVersion != snapshot.CaseVersion {
		return Verification{Valid: false, Reason: "凭据案例或版本与冻结快照不一致"}
	}
	digest, err := SnapshotDigest(snapshot)
	if err != nil {
		return Verification{Valid: false, Reason: err.Error()}
	}
	if digest != value.SnapshotDigest {
		return Verification{Valid: false, Reason: "冻结快照摘要不一致", CaseID: value.CaseID}
	}
	expectedSum := sha256.Sum256([]byte(value.CredentialID + ":" + digest))
	expected := hex.EncodeToString(expectedSum[:16])
	if subtle.ConstantTimeCompare([]byte(expected), []byte(suppliedCode)) != 1 || suppliedCode != value.VerificationCode {
		return Verification{Valid: false, Reason: "验证代码不正确", CaseID: value.CaseID}
	}
	return Verification{Valid: true, CaseID: value.CaseID, Digest: digest}
}
